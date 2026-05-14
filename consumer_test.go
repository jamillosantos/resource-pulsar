package rscpulsar

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jamillosantos/resource-pulsar/consume"
)

type fakeMessage struct {
	pulsar.Message
	key         string
	payload     []byte
	orderingKey string
}

func (f *fakeMessage) Key() string         { return f.key }
func (f *fakeMessage) Payload() []byte     { return f.payload }
func (f *fakeMessage) OrderingKey() string { return f.orderingKey }

type fakeRawConsumer struct {
	pulsar.Consumer
	msgs   chan pulsar.Message
	acks   atomic.Int64
	nacks  atomic.Int64
	closed atomic.Bool
}

func newFakeRaw(buf int) *fakeRawConsumer {
	return &fakeRawConsumer{msgs: make(chan pulsar.Message, buf)}
}

func (f *fakeRawConsumer) Receive(ctx context.Context) (pulsar.Message, error) {
	select {
	case m, ok := <-f.msgs:
		if !ok {
			return nil, errors.New("closed")
		}
		return m, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (f *fakeRawConsumer) Ack(_ pulsar.Message) error { f.acks.Add(1); return nil }
func (f *fakeRawConsumer) Nack(_ pulsar.Message)      { f.nacks.Add(1) }
func (f *fakeRawConsumer) Close()                     { f.closed.Store(true) }

func newTestConsumer(raw pulsar.Consumer, h consume.MessageHandler, workers int, keyer Keyer) *Consumer {
	return &Consumer{
		Consumer:      raw,
		handleMessage: h,
		workers:       workers,
		keyer:         keyer,
	}
}

func TestConsumer_singleWorker_processesSerially(t *testing.T) {
	raw := newFakeRaw(10)
	var (
		mu   sync.Mutex
		seen []string
	)
	h := func(_ context.Context, m pulsar.Message) consume.MessageHandlerResult {
		mu.Lock()
		seen = append(seen, m.Key())
		mu.Unlock()
		return consume.Ack()
	}
	c := newTestConsumer(raw, h, 1, nil)
	require.NoError(t, c.Listen(context.Background()))

	keys := []string{"a", "b", "c", "d", "e"}
	for _, k := range keys {
		raw.msgs <- &fakeMessage{key: k}
	}

	require.Eventually(t, func() bool {
		return raw.acks.Load() == int64(len(keys))
	}, time.Second, 10*time.Millisecond)
	require.NoError(t, c.Close(context.Background()))

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, keys, seen)
	assert.True(t, raw.closed.Load())
}

func TestConsumer_workersZero_usesSinglePath(t *testing.T) {
	raw := newFakeRaw(2)
	var processed atomic.Int64
	h := func(_ context.Context, _ pulsar.Message) consume.MessageHandlerResult {
		processed.Add(1)
		return consume.Ack()
	}
	c := newTestConsumer(raw, h, 0, nil)
	require.NoError(t, c.Listen(context.Background()))

	raw.msgs <- &fakeMessage{key: "x"}
	require.Eventually(t, func() bool { return processed.Load() == 1 }, time.Second, 10*time.Millisecond)
	require.NoError(t, c.Close(context.Background()))

	assert.Nil(t, c.lanes, "Workers<=1 must not allocate lanes")
}

func TestConsumer_pool_perKeyOrderPreserved(t *testing.T) {
	raw := newFakeRaw(200)
	var (
		mu        sync.Mutex
		seenByKey = map[string][]int{}
	)
	h := func(_ context.Context, m pulsar.Message) consume.MessageHandlerResult {
		var seq int
		_, _ = fmt.Sscanf(string(m.Payload()), "%d", &seq)
		time.Sleep(2 * time.Millisecond) // give scheduler a chance to interleave
		mu.Lock()
		seenByKey[m.Key()] = append(seenByKey[m.Key()], seq)
		mu.Unlock()
		return consume.Ack()
	}
	c := newTestConsumer(raw, h, 4, func(m pulsar.Message) string { return m.Key() })
	require.NoError(t, c.Listen(context.Background()))

	keys := []string{"alpha", "beta", "gamma", "delta"}
	const perKey = 20
	expected := make(map[string][]int, len(keys))
	for _, k := range keys {
		expected[k] = make([]int, 0, perKey)
	}
	for i := 0; i < perKey; i++ {
		for _, k := range keys {
			raw.msgs <- &fakeMessage{key: k, payload: []byte(fmt.Sprintf("%d", i))}
			expected[k] = append(expected[k], i)
		}
	}

	require.Eventually(t, func() bool {
		return raw.acks.Load() == int64(perKey*len(keys))
	}, 5*time.Second, 10*time.Millisecond)
	require.NoError(t, c.Close(context.Background()))

	mu.Lock()
	defer mu.Unlock()
	for _, k := range keys {
		assert.Equal(t, expected[k], seenByKey[k], "order broken for key %q", k)
	}
}

func TestConsumer_pool_concurrentAcrossLanes(t *testing.T) {
	raw := newFakeRaw(100)
	const workers = 8
	var inFlight, maxInFlight atomic.Int32
	h := func(_ context.Context, _ pulsar.Message) consume.MessageHandlerResult {
		cur := inFlight.Add(1)
		defer inFlight.Add(-1)
		for {
			prev := maxInFlight.Load()
			if cur <= prev || maxInFlight.CompareAndSwap(prev, cur) {
				break
			}
		}
		time.Sleep(30 * time.Millisecond)
		return consume.Ack()
	}
	c := newTestConsumer(raw, h, workers, nil)
	require.NoError(t, c.Listen(context.Background()))

	for i := 0; i < workers*2; i++ {
		raw.msgs <- &fakeMessage{}
	}
	require.Eventually(t, func() bool { return raw.acks.Load() == int64(workers*2) }, 3*time.Second, 10*time.Millisecond)
	require.NoError(t, c.Close(context.Background()))

	assert.GreaterOrEqual(t, maxInFlight.Load(), int32(workers), "expected all lanes in flight concurrently")
}

func TestConsumer_pool_nackOnHandlerNack(t *testing.T) {
	raw := newFakeRaw(2)
	h := func(_ context.Context, _ pulsar.Message) consume.MessageHandlerResult {
		return consume.Nack()
	}
	c := newTestConsumer(raw, h, 2, nil)
	require.NoError(t, c.Listen(context.Background()))
	raw.msgs <- &fakeMessage{}
	require.Eventually(t, func() bool { return raw.nacks.Load() == 1 }, time.Second, 10*time.Millisecond)
	require.NoError(t, c.Close(context.Background()))
	assert.EqualValues(t, 0, raw.acks.Load())
}

func TestConsumer_pool_closeDrainsInFlight(t *testing.T) {
	raw := newFakeRaw(2)
	started := make(chan struct{})
	release := make(chan struct{})
	h := func(_ context.Context, _ pulsar.Message) consume.MessageHandlerResult {
		started <- struct{}{}
		<-release
		return consume.Ack()
	}
	c := newTestConsumer(raw, h, 2, nil)
	require.NoError(t, c.Listen(context.Background()))
	raw.msgs <- &fakeMessage{}
	<-started

	done := make(chan struct{})
	go func() {
		_ = c.Close(context.Background())
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("Close returned before in-flight handler released")
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Close did not return after handler released")
	}
	assert.EqualValues(t, 1, raw.acks.Load())
	assert.True(t, raw.closed.Load())
}

func TestLaneFor_sameKeyAlwaysSameLane(t *testing.T) {
	c := newTestConsumer(nil, nil, 4, func(m pulsar.Message) string { return m.Key() })
	first := c.laneFor(&fakeMessage{key: "alpha"})
	for i := 0; i < 50; i++ {
		assert.Equal(t, first, c.laneFor(&fakeMessage{key: "alpha"}))
	}
}

func TestLaneFor_nilKeyerRoundRobins(t *testing.T) {
	c := newTestConsumer(nil, nil, 4, nil)
	seen := map[int]bool{}
	for i := 0; i < 16; i++ {
		seen[c.laneFor(&fakeMessage{})] = true
	}
	assert.Len(t, seen, 4, "round-robin should hit every lane")
}

func TestDefaultKeyerFor(t *testing.T) {
	assert.Nil(t, defaultKeyerFor(pulsar.Shared))

	keyer := defaultKeyerFor(pulsar.KeyShared)
	require.NotNil(t, keyer)
	assert.Equal(t, "k", keyer(&fakeMessage{key: "k"}))
	assert.Equal(t, "ord", keyer(&fakeMessage{orderingKey: "ord"}))

	for _, st := range []pulsar.SubscriptionType{pulsar.Exclusive, pulsar.Failover} {
		k := defaultKeyerFor(st)
		require.NotNil(t, k, "subscription type %v", st)
		assert.Equal(t, "", k(&fakeMessage{key: "anything"}), "subscription type %v", st)
	}
}
