package rscpulsar

import (
	"context"
	"hash/fnv"
	"sync"
	"sync/atomic"

	"github.com/apache/pulsar-client-go/pulsar"

	"github.com/jamillosantos/resource-pulsar/consume"
)

// Keyer chooses the lane for a given message when the Consumer runs with
// Workers > 1. Messages that produce the same key are always dispatched
// to the same lane, which preserves per-key ordering.
type Keyer func(pulsar.Message) string

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

	// Set when Workers > 1. lanes has length workers; receiver dispatches
	// each message to lanes[laneFor(msg)] and each lane is drained by its
	// own goroutine.
	workers   int
	keyer     Keyer
	lanes     []chan pulsar.Message
	rrCounter atomic.Uint64
}

func (c *Consumer) Name() string {
	return c.name
}

// Listen starts the receive loop and (when Workers > 1) the worker pool.
// Workers <= 1 keeps the original single-goroutine path: one Receive
// call followed by synchronous handler invocation. Workers > 1 spawns
// one receiver goroutine plus one goroutine per lane; the receiver
// pushes each message onto lanes[laneFor(msg)] and lane workers run the
// handler serially within their lane.
func (c *Consumer) Listen(ctx context.Context) error {
	c.context, c.contextCancelFunc = context.WithCancel(ctx)

	if c.workers <= 1 {
		c.listenWg.Add(1)
		go c.singleLoop()
		return nil
	}

	c.lanes = make([]chan pulsar.Message, c.workers)
	for i := range c.lanes {
		c.lanes[i] = make(chan pulsar.Message, 1)
		c.listenWg.Add(1)
		go c.laneWorker(c.lanes[i])
	}

	c.listenWg.Add(1)
	go c.receiver()

	return nil
}

func (c *Consumer) singleLoop() {
	defer c.listenWg.Done()
	for {
		c.handleError(c.receiveAndProcessMessage())

		select {
		case <-c.context.Done(): // if context has been cancelled, it means we need to leave the reader routine.
			return
		default:
		}
	}
}

func (c *Consumer) receiver() {
	defer c.listenWg.Done()
	defer func() {
		for _, lane := range c.lanes {
			close(lane)
		}
	}()
	for {
		msg, err := c.Consumer.Receive(c.context)
		if err != nil {
			return
		}
		lane := c.laneFor(msg)
		select {
		case c.lanes[lane] <- msg:
		case <-c.context.Done():
			// Shutting down mid-dispatch: nack so the broker redelivers
			// instead of waiting for ack-timeout.
			c.Consumer.Nack(msg)
			return
		}
	}
}

func (c *Consumer) laneWorker(lane <-chan pulsar.Message) {
	defer c.listenWg.Done()
	for msg := range lane {
		if h := c.handleMessage(c.context, msg); h != nil {
			c.handleError(h(c.Consumer, msg))
		}
	}
}

// laneFor returns the lane index for msg. When keyer is nil (the default
// for Shared subscriptions), an atomic counter round-robins across lanes
// for even distribution. Otherwise the key is FNV-hashed and reduced
// modulo the worker count, giving stable per-key affinity.
func (c *Consumer) laneFor(msg pulsar.Message) int {
	if c.keyer == nil {
		return int(c.rrCounter.Add(1) % uint64(c.workers))
	}
	k := c.keyer(msg)
	h := fnv.New32a()
	_, _ = h.Write([]byte(k))
	return int(h.Sum32() % uint32(c.workers))
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

// Close cancels the receive loop, waits for in-flight handlers to drain,
// then closes the underlying pulsar.Consumer. The unified shutdown
// ordering (cancel -> drain -> close) is required for the pool path and
// is safe for the legacy single-goroutine path: Pulsar's Receive(ctx)
// honors context cancellation.
func (c *Consumer) Close(_ context.Context) error {
	if c.contextCancelFunc != nil {
		c.contextCancelFunc()
	}
	c.listenWg.Wait()
	c.Consumer.Close()
	return nil
}

func (c *Consumer) handleError(err error) {
	if err == nil {
		return
	}
	// TODO Implement error handling.
}
