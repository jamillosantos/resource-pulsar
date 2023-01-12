package consume

import (
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
)

// MessageHandler is a function that will be called everytime a message arrives at a consumer.
type MessageHandler func(msg pulsar.Message) MessageHandlerResult

// MessageHandlerResult will be returned by the MessageHandler. The MessageHandlerResult will Ack/Nack the message.
type MessageHandlerResult func(consumer pulsar.Consumer, msg pulsar.Message) error

// Ack should be returned by a MessageHandler whenever the message needs to be acknowledged.
func Ack() MessageHandlerResult {
	return func(consumer pulsar.Consumer, msg pulsar.Message) error {
		return consumer.Ack(msg)
	}
}

// Nack should be returned by a MessageHandler whenever the message needs to be NOT acknowledged.
func Nack() MessageHandlerResult {
	return func(consumer pulsar.Consumer, msg pulsar.Message) error {
		consumer.Nack(msg)
		return nil
	}
}

// Later should be returned by a MessageHandler whenever the message needs to be re-consumed later.
func Later(delay time.Duration) MessageHandlerResult {
	return func(consumer pulsar.Consumer, msg pulsar.Message) error {
		consumer.ReconsumeLater(msg, delay)
		return nil
	}
}
