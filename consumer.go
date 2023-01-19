package rscpulsar

import (
	"context"
	"sync"

	"github.com/apache/pulsar-client-go/pulsar"

	"github.com/jamillosantos/resource-pulsar/consume"
)

func defaultConsumerOpts() pulsar.ConsumerOptions {
	return pulsar.ConsumerOptions{
		Type: pulsar.Shared,
	}
}

type Consumer struct {
	listenWg sync.WaitGroup

	context           context.Context
	contextCancelFunc context.CancelFunc

	handleMessage consume.MessageHandler

	name     string
	Consumer pulsar.Consumer
}

func (c *Consumer) Name() string {
	return c.name
}

// Listen performs the
func (c *Consumer) Listen(ctx context.Context) error {
	c.listenWg.Add(1)

	c.context, c.contextCancelFunc = context.WithCancel(ctx)
	go func() {
		defer c.listenWg.Done()
		for {
			c.handleError(c.receiveAndProcessMessage())

			select {
			case <-c.context.Done(): // if context has been cancelled, it means we need to leave the reader routine.
				return
			default:
			}
		}
	}()

	return nil
}

func (c *Consumer) receiveAndProcessMessage() error {
	msg, err := c.Consumer.Receive(c.context)
	if err != nil {
		// TODO To proper handle this.
		// TODO What are the errors contained here?
		return err
	}
	h := c.handleMessage(c.context, msg)
	if h != nil {
		return h(c.Consumer, msg)
	}
	return nil
}

// Close will close the consumer and wait for the Listen to finish
func (c *Consumer) Close(_ context.Context) error {
	c.Consumer.Close()
	c.contextCancelFunc()
	c.listenWg.Wait() // Wait listen to finish.
	return nil
}

func (c *Consumer) handleError(err error) {
	if err == nil {
		return
	}
	// TODO Implement error handling.
}
