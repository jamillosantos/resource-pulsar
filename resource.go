package rscpulsar

import (
	"context"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"

	"github.com/jamillosantos/resource-pulsar/consume"
)

type Resource struct {
	Client     pulsar.Client
	name       string
	clientOpts pulsar.ClientOptions
}

func defaultOpts() pulsar.ClientOptions {
	return pulsar.ClientOptions{
		ConnectionTimeout: time.Second * 10,
		OperationTimeout:  time.Second * 30,
	}
}

// New builds a new services.Resource for connecting to an Apache Pulsar server.
func New(cfg PlatformConfig, opts ...Option) *Resource {
	r := &Resource{
		clientOpts: defaultOpts(),
	}
	applyConfig(&r.clientOpts, cfg)
	for _, o := range opts {
		o(r)
	}
	return r
}

func applyConfig(clientOpts *pulsar.ClientOptions, cfg PlatformConfig) {
	clientOpts.URL = cfg.URL
	if cfg.Timeouts.Connection != 0 {
		clientOpts.ConnectionTimeout = cfg.Timeouts.Connection
	}
	if cfg.Timeouts.Operation != 0 {
		clientOpts.OperationTimeout = cfg.Timeouts.Operation
	}
}

func (r *Resource) Name() string {
	return r.name
}

func (r *Resource) Start(_ context.Context) error {
	client, err := pulsar.NewClient(r.clientOpts)
	if err != nil {
		return err
	}
	r.Client = client
	return nil
}

// Stop stops the resource using the default pulsar.Client.Close method.
func (r *Resource) Stop(_ context.Context) error {
	r.Client.Close()
	return nil
}

func (r *Resource) Subscribe(cfg SubscriptionPlatformConfig, handler consume.MessageHandler, opts ...consume.Option) (*Consumer, error) {
	consumerOpts := defaultConsumerOpts()
	applyConsumerConfig(cfg, &consumerOpts)
	for _, o := range opts {
		o(&consumerOpts)
	}
	consumer, err := r.Client.Subscribe(consumerOpts)
	if err != nil {
		return nil, err
	}

	keyer := cfg.Keyer
	if keyer == nil && cfg.Workers > 1 {
		keyer = defaultKeyerFor(consumerOpts.Type)
	}

	return &Consumer{
		name:          cfg.Name,
		Consumer:      consumer,
		handleMessage: handler,
		workers:       cfg.Workers,
		keyer:         keyer,
	}, nil
}

func applyConsumerConfig(cfg SubscriptionPlatformConfig, opts *pulsar.ConsumerOptions) {
	opts.Name = cfg.Name
	opts.Topic = cfg.Topic
	opts.Topics = cfg.Topics
	opts.TopicsPattern = cfg.TopicsPattern
	opts.SubscriptionName = cfg.SubscriptionName
	// Zero is ambiguous (pulsar.Exclusive == 0 and "field unset" look the
	// same). Preserve the legacy default — leave Type alone unless the
	// caller chose a non-zero value.
	if cfg.Type != 0 {
		opts.Type = cfg.Type
	}
}

// defaultKeyerFor returns a Keyer that matches the ordering semantics of
// the subscription type. Returning nil means "use the round-robin
// counter path" (only safe for Shared subscriptions, where no order is
// guaranteed).
func defaultKeyerFor(t pulsar.SubscriptionType) Keyer {
	switch t {
	case pulsar.KeyShared:
		return keyByMessageKey
	case pulsar.Exclusive, pulsar.Failover:
		return constantKeyer
	default:
		return nil
	}
}

func keyByMessageKey(m pulsar.Message) string {
	if k := m.Key(); k != "" {
		return k
	}
	return m.OrderingKey()
}

func constantKeyer(_ pulsar.Message) string { return "" }
