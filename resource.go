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
	return &Consumer{
		name:          cfg.Name,
		Consumer:      consumer,
		handleMessage: handler,
	}, nil
}

func applyConsumerConfig(cfg SubscriptionPlatformConfig, opts *pulsar.ConsumerOptions) {
	opts.Name = cfg.Name
	opts.Topic = cfg.Topic
	opts.Topics = cfg.Topics
	opts.TopicsPattern = cfg.TopicsPattern
	opts.SubscriptionName = cfg.SubscriptionName
}
