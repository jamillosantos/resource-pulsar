package rscpulsar

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResource_Start(t *testing.T) {
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	require.NoError(t, err, "failed connecting to docker")

	// pulls an image, creates a container based on it and runs it
	dockerRsc, err := pool.Run("apachepulsar/pulsar", "2.10.2", []string{})
	require.NoError(t, err, "failed starting redis")
	t.Cleanup(func() {
		dockerRsc.Close()
	})

	rsc := New(PlatformConfig{
		URL: fmt.Sprintf("pulsar://%s", dockerRsc.GetHostPort("6650/tcp")),
	})

	require.Eventuallyf(t, func() bool {
		ctx := context.Background()
		err := rsc.Start(ctx)
		return assert.NoError(t, err)
	}, 60*time.Second, 1*time.Second, "pulsar is not ready")

	require.NoError(t, rsc.Stop(context.Background()), "failed closing resource")
}
