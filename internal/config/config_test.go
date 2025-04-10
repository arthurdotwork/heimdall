package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/arthurdotwork/heimdall/internal/config"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func createTempConfig(t *testing.T, config map[string]any) string {
	file, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer file.Close() //nolint:errcheck

	yamlMarshalled, err := yaml.Marshal(config)
	require.NoError(t, err)

	_, err = file.Write(yamlMarshalled)
	require.NoError(t, err)

	err = file.Sync()
	require.NoError(t, err)

	return file.Name()
}

func TestConfig_LoadFromFile(t *testing.T) {
	t.Parallel()

	t.Run("it should return an error if it can not open the config file", func(t *testing.T) {
		cfg, err := config.LoadFromFile("nonexistent.yaml")
		require.Error(t, err)
		require.Nil(t, cfg)
	})

	t.Run("it should return an error if the config is invalid", func(t *testing.T) {
		cfg := map[string]any{"gateway": map[string]any{"port": "invalid-port"}}

		configPath := createTempConfig(t, cfg)
		loadedCfg, err := config.LoadFromFile(configPath)
		require.Error(t, err)
		require.Nil(t, loadedCfg)
	})

	t.Run("it should parse and return the config", func(t *testing.T) {
		cfg := map[string]any{"gateway": map[string]any{"port": 8080}}

		configPath := createTempConfig(t, cfg)
		loadedCfg, err := config.LoadFromFile(configPath)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		require.Equal(t, 8080, loadedCfg.Gateway.Port)
	})
}

func TestConfig_WithDefaults(t *testing.T) {
	t.Parallel()

	t.Run("it should set default values for missing fields", func(t *testing.T) {
		cfg := &config.Config{
			Gateway: config.GatewayConfig{},
		}

		cfg = cfg.WithDefaults()

		require.Equal(t, 8080, cfg.Gateway.Port)
		require.Equal(t, 5*time.Second, cfg.Gateway.ReadTimeout)
		require.Equal(t, 5*time.Second, cfg.Gateway.WriteTimeout)
		require.Equal(t, 10*time.Second, cfg.Gateway.ShutdownTimeout)
	})

	t.Run("it should not override existing values", func(t *testing.T) {
		cfg := &config.Config{
			Gateway: config.GatewayConfig{
				Port:            9090,
				ReadTimeout:     10 * time.Second,
				WriteTimeout:    15 * time.Second,
				ShutdownTimeout: 20 * time.Second,
			},
		}

		cfg = cfg.WithDefaults()

		require.Equal(t, 9090, cfg.Gateway.Port)
		require.Equal(t, 10*time.Second, cfg.Gateway.ReadTimeout)
		require.Equal(t, 15*time.Second, cfg.Gateway.WriteTimeout)
		require.Equal(t, 20*time.Second, cfg.Gateway.ShutdownTimeout)
	})
}
