package rscpulsar

import (
	"context"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
)

type Resource struct {
	pulsar.Client
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

func (r Resource) Name() string {
	return r.name
}

func (r *Resource) Start(_ context.Context) error {
	opts := defaultOpts()

	client, err := pulsar.NewClient(opts)
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
