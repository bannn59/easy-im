package app

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"easy-im/backend/internal/handler"
	"easy-im/backend/internal/hub"
	"easy-im/backend/internal/mq"
	"easy-im/backend/internal/repo"
	"easy-im/backend/internal/service"
)

// APIOptions configures the API HTTP stack.
type APIOptions struct {
	Pool               *pgxpool.Pool
	Log                *slog.Logger
	AuthJWTSecret      string
	AuthTokenTTL       time.Duration
	CORSAllowedOrigins []string
	CookieSecure       bool
	CookieDomain       string
	KafkaBrokers       []string
	VAPIDPublicKey     string
}

// NewAPIHandler wires HTTP handlers for cmd/api.
func NewAPIHandler(opts APIOptions) http.Handler {
	rtHub := hub.New()
	var auth *service.AuthService
	var conv *service.ConversationService
	var msg *service.MessageService
	var friends *service.FriendService
	var pushSvc *service.PushService
	var members service.MembershipChecker

	// Event bus producer. Kafka is optional: when brokers are not configured
	// the api keeps working and simply never publishes offline-push events.
	producer := mq.NoopProducer
	if len(opts.KafkaBrokers) > 0 {
		p, err := mq.NewKafkaProducer(mq.ProducerOpts{
			Brokers: opts.KafkaBrokers,
			ClientID: "easyim-api",
			OnError: func(err error) {
				slog.Warn("kafka produce failed", "service", "api", "error", err)
			},
		})
		if err != nil {
			slog.Warn("kafka producer unavailable; offline push disabled", "service", "api", "error", err)
		} else {
			producer = p
		}
	}
	nodeID := nodeIDFor()
	msgAdapter := &messageEventAdapter{producer: producer, nodeID: nodeID}

	if opts.Pool != nil && opts.AuthJWTSecret != "" {
		users := repo.NewUserRepo(opts.Pool)
		convs := repo.NewConversationRepo(opts.Pool)
		messages := repo.NewMessageRepo(opts.Pool)
		friendRepo := repo.NewFriendRepo(opts.Pool)
		subs := repo.NewPushSubscriptionRepo(opts.Pool)
		members = convs
		auth = service.NewAuthService(
			users,
			service.AuthConfig{
				JWTSecret: []byte(opts.AuthJWTSecret),
				TokenTTL:  opts.AuthTokenTTL,
			},
		)
		conv = service.NewConversationService(convs, users, friendRepo, rtHub).
			WithReadPublisher(msgAdapter).
			WithGroupEventPublisher(msgAdapter)
		msg = service.NewMessageService(messages, convs, rtHub).WithEventPublisher(msgAdapter)
		friends = service.NewFriendService(friendRepo, users)
		pushSvc = service.NewPushService(subs)

		// Publish presence transitions to the bus so the worker can track
		// online state. Non-blocking best effort; the immediate friend
		// broadcast is set separately by the WS handler.
		rtHub.PresenceEventPublisher = func(userID string, online bool) {
			_ = producer.Publish(context.Background(), mq.TopicPresence, userID, mq.PresenceEvent{
				UserID: userID,
				Online: online,
				At:     time.Now().UTC(),
			})
		}

		// Cross-node realtime fanout: this node consumes every message event on
		// the bus and re-delivers it to its own online members, skipping events
		// it produced (local broadcast already handled those).
		startFanoutConsumer(FanoutConsumerOpts{
			Brokers: opts.KafkaBrokers,
			Log:     opts.Log,
			NodeID:  nodeID,
			Members: members,
			Hub:     rtHub,
			Msg:     msg,
			Conv:    conv,
		})
	}
	return handler.NewMux(handler.Deps{
		Pool:               opts.Pool,
		Log:                opts.Log,
		Auth:               auth,
		Conv:               conv,
		Msg:                msg,
		Friends:            friends,
		Push:               pushSvc,
		VAPIDPublicKey:     opts.VAPIDPublicKey,
		Hub:                rtHub,
		Members:            members,
		CORSAllowedOrigins: opts.CORSAllowedOrigins,
		CookieSecure:       opts.CookieSecure,
		CookieDomain:       opts.CookieDomain,
	})
}
