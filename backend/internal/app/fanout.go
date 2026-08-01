package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"easy-im/backend/internal/hub"
	"easy-im/backend/internal/metrics"
	"easy-im/backend/internal/mq"
	"easy-im/backend/internal/service"
)

// nodeIDFor returns a process-unique origin tag (hostname:pid). It tags bus
// events so the fanout consumer can skip events this process produced.
func nodeIDFor() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "unknown"
	}
	return fmt.Sprintf("%s:%d", host, os.Getpid())
}

// startFanoutConsumer launches the per-node realtime fanout consumer on a
// background context tied to process lifetime. Kafka-less runs start nothing.
func startFanoutConsumer(opts FanoutConsumerOpts) {
	c, err := NewFanoutConsumer(opts)
	if err != nil {
		slog.Warn("realtime fanout consumer unavailable; cross-node delivery disabled", "service", "api", "error", err)
		return
	}
	if c == nil {
		return
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	ctx := context.Background()
	go func() {
		log.Info("realtime fanout consumer started", "service", "api", "node_id", opts.NodeID, "group", "easyim-realtime-"+opts.NodeID)
		if err := c.Run(ctx, func(ctx context.Context, msg mq.Message) error {
			return FanoutHandler(ctx, opts, msg)
		}); err != nil {
			log.Warn("realtime fanout consumer exited", "service", "api", "error", err)
		}
	}()
}

// FanoutConsumerOpts wires a per-node realtime fanout consumer.
type FanoutConsumerOpts struct {
	Brokers []string
	Log     *slog.Logger
	// NodeID is this process's origin tag; events produced by this node are
	// skipped to avoid double delivery (local broadcast already covered them).
	NodeID string
	// Members resolves conversation membership for delivery scoping.
	Members service.MembershipChecker
	// Hub publishes the fanned-out event to this node's online connections.
	Hub FanoutHub
	// Msg builds "message.created/edited/recalled" frames from stored messages.
	Msg *service.MessageService
	// Conv builds "message.read" frames for cross-node read receipts.
	Conv *service.ConversationService
}

// FanoutHub is the delivery surface the fanout consumer needs from the hub.
type FanoutHub interface {
	PublishToUsers(userIDs []string, event hub.Event)
}

// NewFanoutConsumer creates a per-node Kafka consumer that re-delivers bus
// message events to this node's online connections. Returns nil (no consumer)
// when brokers are not configured so the API degrades to local broadcast only.
//
// Each node joins its own consumer group (easyim-realtime-<nodeID>) so every
// node reads the full stream — Kafka consumer groups are competing consumers,
// so a shared group would deliver each record to only one node. New groups
// start at the latest offset: realtime fanout must not replay history on first
// join; committed offsets are still resumed on restart.
func NewFanoutConsumer(opts FanoutConsumerOpts) (*mq.Consumer, error) {
	if len(opts.Brokers) == 0 {
		return nil, nil
	}
	return mq.NewConsumer(mq.ConsumerOpts{
		Brokers:    opts.Brokers,
		Group:      "easyim-realtime-" + opts.NodeID,
		ClientID:   "easyim-realtime-" + opts.NodeID,
		Topics:     []string{mq.TopicMessages},
		StartAtEnd: true,
		Log:        opts.Log,
	})
}

// FanoutHandler handles one bus message event, re-delivering it to this node's
// online members. Events whose origin is this node are skipped.
func FanoutHandler(ctx context.Context, opts FanoutConsumerOpts, msg mq.Message) error {
	var ev mq.MessageEvent
	if err := mq.DecodeInto(msg, &ev); err != nil {
		return err
	}
	eventType := string(ev.EventType())
	if ev.Origin == opts.NodeID {
		metrics.FanoutSkippedTotal.WithLabelValues("own_origin").Inc()
		return nil // local broadcast already delivered on this node
	}

	var frame hub.Event
	// deliveryIDs scopes the fan-out. members.changed carries its own member
	// list (post-change); other events query current membership.
	var deliveryIDs []string
	switch ev.EventType() {
	case mq.MessageCreated, mq.MessageEdited, mq.MessageRecalled:
		if opts.Msg == nil {
			metrics.FanoutSkippedTotal.WithLabelValues("no_msg").Inc()
			return nil
		}
		f, err := opts.Msg.FanoutMessage(ctx, ev.ID, wsTypeFor(ev.EventType()))
		if err != nil {
			return err
		}
		frame = f
	case mq.MessageRead:
		if opts.Conv == nil {
			metrics.FanoutSkippedTotal.WithLabelValues("no_conv").Inc()
			return nil
		}
		f, err := opts.Conv.ReadFrame(ev.ConversationID, ev.ReadByUserID, ev.LastReadSeq)
		if err != nil {
			return err
		}
		frame = f
	case mq.GroupMembersChanged:
		payload, err := json.Marshal(map[string]any{
			"conversation_id": ev.ConversationID,
			"action":          ev.Action,
			"user_id":         ev.ActorID,
			"members":         ev.MemberIDs,
		})
		if err != nil {
			return err
		}
		frame = hub.Event{Type: "members.changed", Payload: payload}
		deliveryIDs = ev.MemberIDs
	case mq.GroupConversationRenamed:
		payload, err := json.Marshal(map[string]any{
			"conversation_id": ev.ConversationID,
			"title":           ev.Title,
			"updated_at":      ev.UpdatedAt.UTC().Format(time.RFC3339),
		})
		if err != nil {
			return err
		}
		frame = hub.Event{Type: "conversation.renamed", Payload: payload}
	default:
		metrics.FanoutSkippedTotal.WithLabelValues("unknown_type").Inc()
		return nil
	}

	if opts.Members == nil || opts.Hub == nil {
		metrics.FanoutSkippedTotal.WithLabelValues("no_delivery").Inc()
		return nil
	}
	if len(deliveryIDs) == 0 {
		ids, err := opts.Members.ListMemberIDs(ctx, ev.ConversationID)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}
		deliveryIDs = ids
	}
	metrics.FanoutEventsTotal.WithLabelValues(eventType).Inc()
	opts.Hub.PublishToUsers(deliveryIDs, frame)
	return nil
}

func wsTypeFor(t mq.MessageEventType) string {
	switch t {
	case mq.MessageEdited:
		return "message.edited"
	case mq.MessageRecalled:
		return "message.recalled"
	default:
		return "message.created"
	}
}
