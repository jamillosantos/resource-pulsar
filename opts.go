package rscpulsar

import (
	"github.com/apache/pulsar-client-go/pulsar"
)

// Option are modifications that will be applied to the resource.
type Option func(*Resource)

// WithCustomPulsarConfig allows you to change the pulsar.ClientOptions before the client is created. If you need a
// config that is not supported by the rscpulsar package, you can use this option to apply your own config.
func WithCustomPulsarConfig(f func(opts *pulsar.ClientOptions)) Option {
	return func(r *Resource) {
		f(&r.clientOpts)
	}
}
