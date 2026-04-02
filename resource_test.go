package rscpulsar

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/google/uuid"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jamillosantos/resource-pulsar/consume"
)

func createRsc(t *testing.T) *Resource {
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	require.NoError(t, err, "failed connecting to docker")

	// pulls an image, creates a container based on it and runs it
	dockerRsc, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository:   "apachepulsar/pulsar",
		Tag:          "4.0.3",
		Cmd:          []string{"bin/pulsar", "standalone"},
		ExposedPorts: []string{"6650", "8080"},
	})
	require.NoError(t, err, "failed starting redis")
	t.Cleanup(func() {
		dockerRsc.Close()
	})

	var hostPort string
	require.Eventuallyf(t, func() bool {
		hostPort = dockerRsc.GetHostPort("6650/tcp")
		return hostPort != ""
	}, 600*time.Second, 1*time.Second, "pulsar docker is not ready")

	rsc := New(PlatformConfig{
		URL: fmt.Sprintf("pulsar://%s", dockerRsc.GetHostPort("6650/tcp")),
		Timeouts: Timeouts{
			Connection: 120 * time.Second,
			Operation:  10 * time.Second,
		},
	})
	err = rsc.Start(context.Background())

	require.NoError(t, err, "failed starting resource")
	return rsc
}

func TestResource_Start(t *testing.T) {
	rsc := createRsc(t)

	require.Eventuallyf(t, func() bool {
		ctx := context.Background()
		err := rsc.Start(ctx)
		return assert.NoError(t, err)
	}, 60*time.Second, 1*time.Second, "pulsar is not ready")

	require.NoError(t, rsc.Stop(context.Background()), "failed closing resource")
}

func TestResource_Subscribe(t *testing.T) {
	rsc := createRsc(t)

	topicName := uuid.New().String()
	fullTopicName := "persistent://public/default/" + topicName

	var (
		consumer *Consumer
		mu       sync.Mutex
		gotData  = make([]uuid.UUID, 0)
	)
	require.Eventuallyf(t, func() bool {
		c, err := rsc.Subscribe(SubscriptionPlatformConfig{
			SubscriptionName: "subscription-" + topicName,
			Topic:            fullTopicName,
		}, func(ctx context.Context, msg pulsar.Message) consume.MessageHandlerResult {
			d, err := uuid.FromBytes(msg.Payload())
			require.NoError(t, err, "failed parsing message payload into an UUID")
			mu.Lock()
			gotData = append(gotData, d)
			mu.Unlock()
			return consume.Ack()
		})
		if err != nil {
			return false
		}
		consumer = c
		return true
	}, 120*time.Second, time.Second, "pulsar consumer is not ready")

	err := consumer.Listen(context.Background())
	require.NoError(t, err, "failed listening to consumer")
	t.Cleanup(func() {
		_ = consumer.Close(context.Background())
	})

	producer, err := rsc.Client.CreateProducer(pulsar.ProducerOptions{
		Topic: fullTopicName,
		Name:  "producer-" + topicName,
	})
	require.NoError(t, err)

	wantData := make([]uuid.UUID, 10)
	for i := range wantData {
		wantData[i] = uuid.New()
	}

	for _, d := range wantData {
		_, err := producer.Send(context.Background(), &pulsar.ProducerMessage{
			Payload: d[:],
		})
		require.NoError(t, err)
	}

	require.Eventuallyf(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(gotData) == len(wantData)
	}, 30*time.Second, 10*time.Millisecond, "messages not received")

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, wantData, gotData)

	require.NoError(t, rsc.Stop(context.Background()), "failed closing resource")
}
