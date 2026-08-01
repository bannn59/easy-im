package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/domain"
	"easy-im/backend/internal/hub"
)

// ConversationStore persists conversations.
type ConversationStore interface {
	Create(ctx context.Context, c domain.Conversation, memberIDs []string) error
	ListForUser(ctx context.Context, userID string) ([]domain.Conversation, error)
	GetIfMember(ctx context.Context, conversationID, userID string) (domain.Conversation, error)
	MarkRead(ctx context.Context, conversationID, userID string, seq int64) (int64, error)
	FindDirectBetween(ctx context.Context, userID1, userID2 string) (domain.Conversation, error)
	ListMemberIDs(ctx context.Context, conversationID string) ([]string, error)
	AddMembers(ctx context.Context, conversationID string, userIDs []string) error
	RemoveMember(ctx context.Context, conversationID, userID string) error
	SetOwner(ctx context.Context, conversationID, newOwnerID string) error
	SetTitle(ctx context.Context, conversationID, title string) error
}

// ConversationUserLookup resolves users for open-DM.
type ConversationUserLookup interface {
	FindByID(ctx context.Context, id string) (domain.User, error)
}

// FriendshipChecker reports whether two users are accepted friends.
type FriendshipChecker interface {
	AreFriends(ctx context.Context, userID1, userID2 string) (bool, error)
}

// ConversationService implements open-DM / list / get with membership ACL.
type ConversationService struct {
	convs    ConversationStore
	users    ConversationUserLookup
	friends  FriendshipChecker
	rt       RealtimePublisher // optional; nil-safe
	readPub  ReadEventPublisher
	groupPub GroupEventPublisher
	now      func() time.Time
}

// ReadEventPublisher publishes read-cursor events to the bus for cross-node
// fanout. Optional; nil-safe.
type ReadEventPublisher interface {
	PublishMessageRead(ctx context.Context, conversationID, userID string, lastReadSeq int64) error
}

// GroupEventPublisher publishes group membership/rename events to the bus for
// cross-node fanout. Optional; nil-safe.
type GroupEventPublisher interface {
	PublishMembersChanged(ctx context.Context, conversationID, action, actorID string, members []string) error
	PublishConversationRenamed(ctx context.Context, conversationID, title string, updatedAt time.Time) error
}

func NewConversationService(convs ConversationStore, users ConversationUserLookup, friends FriendshipChecker, rt RealtimePublisher) *ConversationService {
	return &ConversationService{convs: convs, users: users, friends: friends, rt: rt, now: time.Now}
}

// WithReadPublisher attaches a bus adapter; mark-read then publishes a read
// event for cross-node delivery.
func (s *ConversationService) WithReadPublisher(p ReadEventPublisher) *ConversationService {
	s.readPub = p
	return s
}

// WithGroupEventPublisher attaches a bus adapter; group ops then publish
// membership/rename events for cross-node delivery.
func (s *ConversationService) WithGroupEventPublisher(p GroupEventPublisher) *ConversationService {
	s.groupPub = p
	return s
}

// OpenDirect get-or-creates the unique 1:1 conversation between self and peer.
// Requires an accepted friendship. Does not gate historical non-friend conversations elsewhere.
func (s *ConversationService) OpenDirect(ctx context.Context, selfUserID, peerUserID string) (domain.Conversation, error) {
	if selfUserID == "" {
		return domain.Conversation{}, apperr.Unauthorized("missing credentials")
	}
	if peerUserID == "" {
		return domain.Conversation{}, apperr.Invalid("peer user id required")
	}
	if peerUserID == selfUserID {
		return domain.Conversation{}, apperr.Invalid("cannot open conversation with yourself")
	}
	if s.convs == nil || s.users == nil || s.friends == nil {
		return domain.Conversation{}, apperr.Unavailable("database not configured")
	}

	if _, err := s.users.FindByID(ctx, peerUserID); err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return domain.Conversation{}, apperr.NotFound("user not found")
		}
		return domain.Conversation{}, err
	}

	ok, err := s.friends.AreFriends(ctx, selfUserID, peerUserID)
	if err != nil {
		return domain.Conversation{}, err
	}
	if !ok {
		return domain.Conversation{}, apperr.Forbidden("not friends")
	}

	existing, err := s.convs.FindDirectBetween(ctx, selfUserID, peerUserID)
	if err == nil {
		return s.convs.GetIfMember(ctx, existing.ID, selfUserID)
	}
	if !errors.Is(err, apperr.ErrNotFound) {
		return domain.Conversation{}, err
	}

	now := s.now().UTC()
	c := domain.Conversation{
		ID:        uuid.NewString(),
		Title:     nil,
		CreatedBy: selfUserID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.convs.Create(ctx, c, []string{selfUserID, peerUserID}); err != nil {
		return domain.Conversation{}, err
	}
	return s.convs.GetIfMember(ctx, c.ID, selfUserID)
}

// maxGroupSize caps group membership to bound fanout and abuse.
const maxGroupSize = 50

// CreateGroup creates a group conversation with the given member ids (plus the
// creator). Every member must be an accepted friend of the creator. The
// creator is not required to be friends with themselves.
func (s *ConversationService) CreateGroup(ctx context.Context, selfUserID string, title *string, memberIDs []string) (domain.Conversation, error) {
	if selfUserID == "" {
		return domain.Conversation{}, apperr.Unauthorized("missing credentials")
	}
	if s.convs == nil || s.users == nil || s.friends == nil {
		return domain.Conversation{}, apperr.Unavailable("database not configured")
	}

	// Dedupe and drop the creator from member_ids (they are added implicitly).
	seen := map[string]struct{}{selfUserID: {}}
	var members []string
	for _, id := range memberIDs {
		id = strings.TrimSpace(id)
		if id == "" || id == selfUserID {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		members = append(members, id)
	}
	if len(members) < 1 {
		return domain.Conversation{}, apperr.Invalid("group requires at least one other member")
	}
	if len(members)+1 > maxGroupSize {
		return domain.Conversation{}, apperr.Invalid("group too large")
	}

	for _, id := range members {
		if _, err := s.users.FindByID(ctx, id); err != nil {
			if errors.Is(err, apperr.ErrNotFound) {
				return domain.Conversation{}, apperr.NotFound("member user not found")
			}
			return domain.Conversation{}, err
		}
		ok, err := s.friends.AreFriends(ctx, selfUserID, id)
		if err != nil {
			return domain.Conversation{}, err
		}
		if !ok {
			return domain.Conversation{}, apperr.Forbidden("not friends with " + id)
		}
	}

	now := s.now().UTC()
	allMembers := append([]string{selfUserID}, members...)
	c := domain.Conversation{
		ID:        uuid.NewString(),
		Title:     title,
		CreatedBy: selfUserID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.convs.Create(ctx, c, allMembers); err != nil {
		return domain.Conversation{}, err
	}
	return s.convs.GetIfMember(ctx, c.ID, selfUserID)
}

// requireMember loads the conversation and asserts userID is a member.
// Returns the conversation and the current member id set.
func (s *ConversationService) requireMember(ctx context.Context, conversationID, userID string) (domain.Conversation, error) {
	if conversationID == "" {
		return domain.Conversation{}, apperr.Invalid("conversation id required")
	}
	if s.convs == nil {
		return domain.Conversation{}, apperr.Unavailable("database not configured")
	}
	c, err := s.convs.GetIfMember(ctx, conversationID, userID)
	if err != nil {
		return domain.Conversation{}, err
	}
	return c, nil
}

// requireOwner asserts userID is the group owner (created_by).
func (s *ConversationService) requireOwner(ctx context.Context, conversationID, userID string) (domain.Conversation, error) {
	c, err := s.requireMember(ctx, conversationID, userID)
	if err != nil {
		return domain.Conversation{}, err
	}
	if c.CreatedBy != userID {
		return domain.Conversation{}, apperr.Forbidden("not the group owner")
	}
	return c, nil
}

// memberSet returns the current member ids of a conversation as a set.
func (s *ConversationService) memberSet(ctx context.Context, conversationID string) (map[string]struct{}, error) {
	ids, err := s.convs.ListMemberIDs(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set, nil
}

// broadcastMembersChanged pushes a members.changed WS event to all current
// members (the caller passes the post-change member ids), then publishes the
// event to the bus for cross-node fanout.
func (s *ConversationService) broadcastMembersChanged(ctx context.Context, conversationID, action, userID string, members []string) {
	payload, err := json.Marshal(map[string]any{
		"conversation_id": conversationID,
		"action":          action,
		"user_id":         userID,
		"members":         members,
	})
	if err != nil {
		return
	}
	if s.rt != nil {
		s.rt.PublishToUsers(members, hub.Event{Type: "members.changed", Payload: payload})
	}
	if s.groupPub != nil {
		if err := s.groupPub.PublishMembersChanged(ctx, conversationID, action, userID, members); err != nil {
			slog.Warn("publish members.changed event failed", "conversation_id", conversationID, "error", err)
		}
	}
}

// RenameGroup updates a group's display title. Only the group owner may rename.
func (s *ConversationService) RenameGroup(ctx context.Context, conversationID, operatorID string, title *string) (domain.Conversation, error) {
	if _, err := s.requireOwner(ctx, conversationID, operatorID); err != nil {
		return domain.Conversation{}, err
	}
	if title == nil || strings.TrimSpace(*title) == "" {
		return domain.Conversation{}, apperr.Invalid("title is required")
	}
	if err := s.convs.SetTitle(ctx, conversationID, *title); err != nil {
		return domain.Conversation{}, err
	}
	updated, err := s.convs.GetIfMember(ctx, conversationID, operatorID)
	if err != nil {
		return domain.Conversation{}, err
	}
	ids, err := s.convs.ListMemberIDs(ctx, conversationID)
	if err != nil {
		return domain.Conversation{}, err
	}
	s.broadcastRenamed(ctx, conversationID, *title, updated.UpdatedAt, ids)
	return updated, nil
}

// broadcastRenamed pushes a conversation.renamed WS event to all members, then
// publishes the event to the bus for cross-node fanout.
func (s *ConversationService) broadcastRenamed(ctx context.Context, conversationID, title string, updatedAt time.Time, members []string) {
	payload, err := json.Marshal(map[string]any{
		"conversation_id": conversationID,
		"title":           title,
		"updated_at":      updatedAt.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return
	}
	if s.rt != nil {
		s.rt.PublishToUsers(members, hub.Event{Type: "conversation.renamed", Payload: payload})
	}
	if s.groupPub != nil {
		if err := s.groupPub.PublishConversationRenamed(ctx, conversationID, title, updatedAt); err != nil {
			slog.Warn("publish conversation.renamed event failed", "conversation_id", conversationID, "error", err)
		}
	}
}

// AddMembers adds friend users to a group. Any member may add their friends.
func (s *ConversationService) AddMembers(ctx context.Context, conversationID, operatorID string, userIDs []string) error {
	if _, err := s.requireMember(ctx, conversationID, operatorID); err != nil {
		return err
	}
	if s.users == nil || s.friends == nil {
		return apperr.Unavailable("database not configured")
	}
	current, err := s.memberSet(ctx, conversationID)
	if err != nil {
		return err
	}

	var added []string
	for _, id := range userIDs {
		id = strings.TrimSpace(id)
		if id == "" || id == operatorID {
			continue
		}
		if _, already := current[id]; already {
			return apperr.Conflict("user already in group")
		}
		if _, err := s.users.FindByID(ctx, id); err != nil {
			if errors.Is(err, apperr.ErrNotFound) {
				return apperr.NotFound("member user not found")
			}
			return err
		}
		ok, err := s.friends.AreFriends(ctx, operatorID, id)
		if err != nil {
			return err
		}
		if !ok {
			return apperr.Forbidden("not friends with " + id)
		}
		added = append(added, id)
	}
	if len(added) == 0 {
		return nil // all targets already members or invalid; not an error
	}
	if len(current)+len(added) > maxGroupSize {
		return apperr.Invalid("group too large")
	}
	if err := s.convs.AddMembers(ctx, conversationID, added); err != nil {
		return err
	}
	ids, err := s.convs.ListMemberIDs(ctx, conversationID)
	if err != nil {
		return err
	}
	s.broadcastMembersChanged(ctx, conversationID, "added", operatorID, ids)
	return nil
}

// LeaveGroup removes the caller from a group. The owner must transfer first.
func (s *ConversationService) LeaveGroup(ctx context.Context, conversationID, operatorID string) error {
	c, err := s.requireMember(ctx, conversationID, operatorID)
	if err != nil {
		return err
	}
	if c.CreatedBy == operatorID {
		return apperr.Conflict("owner must transfer ownership before leaving")
	}
	ids, err := s.convs.ListMemberIDs(ctx, conversationID)
	if err != nil {
		return err
	}
	if len(ids) <= 1 {
		return apperr.Conflict("last member cannot leave")
	}
	if err := s.convs.RemoveMember(ctx, conversationID, operatorID); err != nil {
		return err
	}
	remaining := make([]string, 0, len(ids)-1)
	for _, id := range ids {
		if id != operatorID {
			remaining = append(remaining, id)
		}
	}
	s.broadcastMembersChanged(ctx, conversationID, "left", operatorID, remaining)
	return nil
}

// KickMember removes a member from a group. Only the owner may kick.
func (s *ConversationService) KickMember(ctx context.Context, conversationID, operatorID, targetID string) error {
	if _, err := s.requireOwner(ctx, conversationID, operatorID); err != nil {
		return err
	}
	if targetID == operatorID {
		return apperr.Invalid("cannot kick yourself")
	}
	current, err := s.memberSet(ctx, conversationID)
	if err != nil {
		return err
	}
	if _, ok := current[targetID]; !ok {
		return apperr.NotFound("member not in conversation")
	}
	if err := s.convs.RemoveMember(ctx, conversationID, targetID); err != nil {
		return err
	}
	ids, err := s.convs.ListMemberIDs(ctx, conversationID)
	if err != nil {
		return err
	}
	s.broadcastMembersChanged(ctx, conversationID, "kicked", targetID, ids)
	return nil
}

// TransferOwner hands group ownership to another member. Only the owner may
// transfer; the previous owner becomes a regular member.
func (s *ConversationService) TransferOwner(ctx context.Context, conversationID, operatorID, newOwnerID string) error {
	if _, err := s.requireOwner(ctx, conversationID, operatorID); err != nil {
		return err
	}
	if newOwnerID == operatorID {
		return apperr.Invalid("cannot transfer to yourself")
	}
	current, err := s.memberSet(ctx, conversationID)
	if err != nil {
		return err
	}
	if _, ok := current[newOwnerID]; !ok {
		return apperr.NotFound("target not in conversation")
	}
	if err := s.convs.SetOwner(ctx, conversationID, newOwnerID); err != nil {
		return err
	}
	ids, err := s.convs.ListMemberIDs(ctx, conversationID)
	if err != nil {
		return err
	}
	s.broadcastMembersChanged(ctx, conversationID, "owner_transferred", newOwnerID, ids)
	return nil
}

func (s *ConversationService) List(ctx context.Context, userID string) ([]domain.Conversation, error) {
	if userID == "" {
		return nil, apperr.Unauthorized("missing credentials")
	}
	if s.convs == nil {
		return nil, apperr.Unavailable("database not configured")
	}
	return s.convs.ListForUser(ctx, userID)
}

func (s *ConversationService) Get(ctx context.Context, conversationID, userID string) (domain.Conversation, error) {
	if userID == "" {
		return domain.Conversation{}, apperr.Unauthorized("missing credentials")
	}
	if conversationID == "" {
		return domain.Conversation{}, apperr.Invalid("conversation id required")
	}
	if s.convs == nil {
		return domain.Conversation{}, apperr.Unavailable("database not configured")
	}
	return s.convs.GetIfMember(ctx, conversationID, userID)
}

// MarkReadResult is returned after advancing the member read cursor.
type MarkReadResult struct {
	LastReadSeq int64
	UnreadCount int64
}

// MarkRead sets last_read_seq to at least seq (default: conversation head seq).
func (s *ConversationService) MarkRead(ctx context.Context, conversationID, userID string, seq *int64) (MarkReadResult, error) {
	if userID == "" {
		return MarkReadResult{}, apperr.Unauthorized("missing credentials")
	}
	if conversationID == "" {
		return MarkReadResult{}, apperr.Invalid("conversation id required")
	}
	if s.convs == nil {
		return MarkReadResult{}, apperr.Unavailable("database not configured")
	}
	c, err := s.convs.GetIfMember(ctx, conversationID, userID)
	if err != nil {
		return MarkReadResult{}, err
	}
	target := int64(0)
	if c.LastMessageSeq != nil {
		target = *c.LastMessageSeq
	}
	if seq != nil {
		if *seq < 0 {
			return MarkReadResult{}, apperr.Invalid("invalid seq")
		}
		target = *seq
		if c.LastMessageSeq != nil && target > *c.LastMessageSeq {
			target = *c.LastMessageSeq
		}
	}
	last, err := s.convs.MarkRead(ctx, conversationID, userID, target)
	if err != nil {
		return MarkReadResult{}, err
	}
	s.broadcastRead(ctx, conversationID, userID, last)
	s.publishRead(ctx, conversationID, userID, last)
	return MarkReadResult{LastReadSeq: last, UnreadCount: 0}, nil
}

// publishRead notifies the bus of an advanced read cursor. Failures are
// best-effort and must never block or fail the HTTP path.
func (s *ConversationService) publishRead(ctx context.Context, conversationID, userID string, lastReadSeq int64) {
	if s.readPub == nil {
		return
	}
	if err := s.readPub.PublishMessageRead(ctx, conversationID, userID, lastReadSeq); err != nil {
		slog.Warn("publish message read event failed", "conversation_id", conversationID, "user_id", userID, "error", err)
	}
}

func (s *ConversationService) broadcastRead(ctx context.Context, conversationID, readerID string, lastReadSeq int64) {
	if s.rt == nil {
		return
	}
	memberIDs, err := s.convs.ListMemberIDs(ctx, conversationID)
	if err != nil || len(memberIDs) == 0 {
		return
	}
	frame, err := s.ReadFrame(conversationID, readerID, lastReadSeq)
	if err != nil {
		return
	}
	s.rt.PublishToUsers(memberIDs, frame)
}

// ReadFrame builds the WS "message.read" hub frame, sharing the payload shape
// with the HTTP/WS path so cross-node delivery never drifts.
func (s *ConversationService) ReadFrame(conversationID, readerID string, lastReadSeq int64) (hub.Event, error) {
	payload, err := json.Marshal(map[string]any{
		"conversation_id": conversationID,
		"reader_id":       readerID,
		"last_read_seq":   lastReadSeq,
	})
	if err != nil {
		return hub.Event{}, err
	}
	return hub.Event{Type: "message.read", Payload: payload}, nil
}
