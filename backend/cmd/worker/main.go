package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"easy-im/backend/internal/config"
	"easy-im/backend/internal/db"
	"easy-im/backend/internal/mq"
	"easy-im/backend/internal/push"
	"easy-im/backend/internal/repo"
)

func main() {
	cfg := config.Load()
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if cfg.DatabaseURL == "" {
		log.Error("DATABASE_URL unset; worker requires a database", "service", "worker")
		os.Exit(1)
	}
	if len(cfg.KafkaBrokers) == 0 {
		log.Error("KAFKA_BROKERS unset; worker requires Kafka", "service", "worker")
		os.Exit(1)
	}
	if cfg.VAPIDPublicKey == "" || cfg.VAPIDPrivateKey == "" || cfg.PushSubject == "" {
		log.Error("VAPID keys / PUSH_SUBJECT unset; worker cannot send pushes", "service", "worker")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	pool, err := db.Open(ctx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		log.Error("database open failed", "service", "worker", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	log.Info("database connected", "service", "worker")

	run(ctx, pool, cfg, log)
}

func run(ctx context.Context, pool *pgxpool.Pool, cfg config.Config, log *slog.Logger) {
	sender, err := push.NewSender(cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey, cfg.PushSubject)
	if err != nil {
		log.Error("push sender config invalid", "service", "worker", "error", err)
		os.Exit(1)
	}

	convs := repo.NewConversationRepo(pool)
	subs := repo.NewPushSubscriptionRepo(pool)
	users := repo.NewUserRepo(pool)

	tracker := push.NewPresenceTracker()

	// The flusher owns the flush hook; the aggregator is what the offline
	// handler queues into.
	_, agg := push.NewFlusher(convs, tracker, subs, subs, sender, cfg.PushAggregateWindow, log)

	handler := push.NewOfflineHandler(convs, users, tracker, agg, log)

	// Presence consumer keeps the online set fresh.
	presenceConsumer, err := mq.NewConsumer(mq.ConsumerOpts{
		Brokers:  cfg.KafkaBrokers,
		Group:    "easyim-worker-presence",
		ClientID: "easyim-worker-presence",
		Topics:   []string{mq.TopicPresence},
		Log:      log,
	})
	if err != nil {
		log.Error("presence consumer failed", "service", "worker", "error", err)
		os.Exit(1)
	}
	defer presenceConsumer.Close()

	// Message consumer drives offline pushes.
	msgConsumer, err := mq.NewConsumer(mq.ConsumerOpts{
		Brokers:  cfg.KafkaBrokers,
		Group:    "easyim-worker-offline-push",
		ClientID: "easyim-worker-offline-push",
		Topics:   []string{mq.TopicMessages},
		Log:      log,
	})
	if err != nil {
		log.Error("message consumer failed", "service", "worker", "error", err)
		os.Exit(1)
	}
	defer msgConsumer.Close()

	runCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 2)
	go func() {
		errCh <- presenceConsumer.Run(runCtx, func(ctx context.Context, msg mq.Message) error {
			var ev mq.PresenceEvent
			if err := mq.DecodeInto(msg, &ev); err != nil {
				return err
			}
			tracker.Set(ev.UserID, ev.Online)
			return nil
		})
	}()
	go func() {
		errCh <- msgConsumer.Run(runCtx, func(ctx context.Context, msg mq.Message) error {
			var ev mq.MessageEvent
			if err := mq.DecodeInto(msg, &ev); err != nil {
				return err
			}
			handler.HandleMessage(ctx, ev.ToDomain())
			return nil
		})
	}()

	log.Info("worker started", "service", "worker", "brokers", cfg.KafkaBrokers)

	select {
	case <-runCtx.Done():
		log.Info("worker shutting down", "service", "worker")
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Error("worker consumer exited", "service", "worker", "error", err)
			os.Exit(1)
		}
	}
	agg.Stop()
}
