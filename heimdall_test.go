package heimdall_test

import (
	"context"
	"testing"
	"time"

	"github.com/arthurdotwork/heimdall"
	"github.com/stretchr/testify/require"
)

func TestNewGateway(t *testing.T) {
	t.Parallel()

	t.Run("it should return an error if it can not load the config", func(t *testing.T) {
		_, err := heimdall.New("nonexistent.yaml")
		require.Error(t, err)
	})

	t.Run("it should return an error if it can not create the router", func(t *testing.T) {
		config := map[string]any{
			"endpoints": []map[string]any{
				{
					"target": "://invalid-url",
				},
			},
		}

		configPath := createTempConfig(t, config)

		_, err := heimdall.New(configPath)
		require.Error(t, err)
	})

	t.Run("it should create the gateway with default values", func(t *testing.T) {
		config := map[string]any{
			"gateway": map[string]any{
				"port": 8080,
			},
			"endpoints": []map[string]any{
				{
					"name":   "test-endpoint",
					"path":   "/test",
					"target": "http://example.com",
					"method": "GET",
				},
			},
		}

		configPath := createTempConfig(t, config)

		gateway, err := heimdall.New(configPath)
		require.NoError(t, err)
		require.NotNil(t, gateway)
		require.Equal(t, 5*time.Second, gateway.Config().Gateway.ReadTimeout)
	})
}

func TestGateway_Start(t *testing.T) {
	t.Parallel()

	t.Run("it should return an error if the server fails to start", func(t *testing.T) {
		config := map[string]any{
			"gateway": map[string]any{
				"port": -1, // Invalid port
			},
		}

		configPath := createTempConfig(t, config)

		gateway, err := heimdall.New(configPath)
		require.NoError(t, err)

		err = gateway.Start(context.Background())
		t.Log(err)
		require.Error(t, err)
	})

	t.Run("it should start the gateway", func(t *testing.T) {
		config := map[string]any{
			"gateway": map[string]any{
				"port": 8088,
			},
		}

		configPath := createTempConfig(t, config)
		gateway, err := heimdall.New(configPath)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ch := make(chan error, 1)
		go func() {
			ch <- gateway.Start(ctx)
		}()

		<-time.After(time.Second)
		cancel()
		require.NoError(t, <-ch)
	})
}

func TestGateway_Config(t *testing.T) {
	t.Parallel()

	t.Run("it should return the config", func(t *testing.T) {
		config := map[string]any{
			"gateway": map[string]any{
				"port": 8080,
			},
		}

		configPath := createTempConfig(t, config)
		gateway, err := heimdall.New(configPath)
		require.NoError(t, err)

		cfg := gateway.Config()
		require.NotNil(t, cfg)
		require.Equal(t, 8080, cfg.Gateway.Port)
	})
}
