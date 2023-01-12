package consume

import (
	"github.com/apache/pulsar-client-go/pulsar"
)

type Option func(*pulsar.ConsumerOptions)

func WithCustomConsumerOptions(f Option) Option {
	return func(options *pulsar.ConsumerOptions) {
		f(options)
	}
}
