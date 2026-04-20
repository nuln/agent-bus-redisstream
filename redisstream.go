// Package redisstream implements an EventBus using Redis Streams.
package redisstream

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	agent "github.com/nuln/agent-core"
	"github.com/redis/go-redis/v9"
)

func init() {
	agent.RegisterPluginConfigSpec(agent.PluginConfigSpec{
		PluginName:  "redisstream",
		PluginType:  "bus",
		Description: "Redis Streams publish-subscribe event bus",
		Fields: []agent.ConfigField{
			{Key: "url", EnvVar: "REDIS_URL", Description: "Redis URL", Required: true, Type: agent.ConfigFieldString},
			{Key: "group", EnvVar: "REDIS_STREAM_GROUP", Description: "Consumer group name", Default: "agent", Type: agent.ConfigFieldString},
		},
	})

	agent.RegisterEventBus("redisstream", func(opts map[string]any) (agent.EventBus, error) {
		url, _ := opts["url"].(string)
		if url == "" {
			url = os.Getenv("REDIS_URL")
		}
		group, _ := opts["group"].(string)
		if group == "" {
			group = os.Getenv("REDIS_STREAM_GROUP")
		}
		if group == "" {
			group = "agent"
		}
		return New(url, group)
	})
}

// RedisStreamBus implements agent.EventBus using Redis Streams.
type RedisStreamBus struct {
	client *redis.Client
	group  string
}

// New creates a RedisStreamBus.
func New(redisURL, group string) (*RedisStreamBus, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("redisstream bus: parse URL: %w", err)
	}
	client := redis.NewClient(opt)
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redisstream bus: ping: %w", err)
	}
	return &RedisStreamBus{client: client, group: group}, nil
}

// Publish adds a message to a Redis stream.
func (b *RedisStreamBus) Publish(ctx context.Context, topic string, payload []byte) error {
	err := b.client.XAdd(ctx, &redis.XAddArgs{
		Stream: topic,
		Values: map[string]any{"data": payload},
	}).Err()
	if err != nil {
		return fmt.Errorf("redisstream: publish to %q: %w", topic, err)
	}
	return nil
}

// Subscribe starts a goroutine reading from the stream using consumer group.
func (b *RedisStreamBus) Subscribe(ctx context.Context, topic string, handler func([]byte)) (agent.EventSubscription, error) {
	// Ensure group exists
	b.client.XGroupCreateMkStream(ctx, topic, b.group, "0")

	sub := &redisSub{cancel: nil}
	subCtx, cancel := context.WithCancel(ctx)
	sub.cancel = cancel

	go func() {
		consumer := fmt.Sprintf("consumer-%d", time.Now().UnixNano())
		for {
			select {
			case <-subCtx.Done():
				return
			default:
			}
			msgs, err := b.client.XReadGroup(subCtx, &redis.XReadGroupArgs{
				Group:    b.group,
				Consumer: consumer,
				Streams:  []string{topic, ">"},
				Count:    10,
				Block:    2 * time.Second,
			}).Result()
			if err != nil {
				if subCtx.Err() != nil {
					return
				}
				slog.Debug("redisstream: read group", "err", err)
				continue
			}
			for _, stream := range msgs {
				for _, msg := range stream.Messages {
					if data, ok := msg.Values["data"]; ok {
						switch v := data.(type) {
						case string:
							handler([]byte(v))
						case []byte:
							handler(v)
						}
					}
					b.client.XAck(subCtx, topic, b.group, msg.ID)
				}
			}
		}
	}()

	return sub, nil
}

type redisSub struct {
	mu     sync.Mutex
	cancel context.CancelFunc
}

func (s *redisSub) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
	return nil
}
