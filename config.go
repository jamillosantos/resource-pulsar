package rscpulsar

import (
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
)

type Timeouts struct {
	Connection time.Duration `config:"connection"`
	Operation  time.Duration `config:"operation"`
}

type PlatformConfig struct {
	URL      string   `config:"url,secret"`
	Timeouts Timeouts `config:"timeouts"`
}

type SubscriptionPlatformConfig struct {
	Name             string   `config:"name"`
	SubscriptionName string   `config:"subscription_name,required"`
	Topic            string   `config:"topic"`
	Topics           []string `config:"topics"`
	TopicsPattern    string   `config:"topics_pattern"`

	// Type is the Pulsar subscription type. When zero (which happens to
	// coincide with pulsar.Exclusive), the lib falls back to Shared to
	// match the pre-existing default — set Type explicitly to anything
	// other than zero/Shared and use WithCustomConsumerOptions if you
	// need to override that legacy default.
	Type pulsar.SubscriptionType `config:"-"`

	// Workers caps the number of concurrent in-process message handlers.
	// Zero or one preserves the original single-goroutine receive loop.
	// When > 1, the Consumer dispatches each message to one of N lanes
	// selected by Keyer (or by a safe per-Type default).
	Workers int `config:"workers"`

	// Keyer chooses the lane for a message. Optional; the lib selects a
	// default based on Type when nil:
	//   Shared              -> round-robin counter (no key derived).
	//   KeyShared           -> msg.Key() (falling back to msg.OrderingKey()).
	//   Exclusive, Failover -> constant "" (single effective lane —
	//                          preserves topic-level ordering).
	Keyer Keyer `config:"-"`
}
