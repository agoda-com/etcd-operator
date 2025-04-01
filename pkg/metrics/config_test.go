package metrics_test

import (
	"testing"
	"time"

	"github.com/agoda-com/etcd-operator/pkg/metrics"
	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	config := metrics.DefaultConfig()

	assert.Equal(t, metrics.DefaultOTLPEndpoint, config.OTLPEndpoint)
	assert.Equal(t, metrics.DefaultServiceName, config.ServiceName)
	assert.Equal(t, metrics.DefaultNamespace, config.Namespace)
	assert.Equal(t, metrics.DefaultExportInterval, config.ExportInterval)
	assert.Equal(t, metrics.DefaultInsecure, config.Insecure)
}

func TestConfigFromEnv(t *testing.T) {
	t.Run("Valid Environment Variables", func(t *testing.T) {
		// Set environment variables
		t.Setenv(metrics.EnvOTLPEndpoint, "test-endpoint:4317")
		t.Setenv(metrics.EnvServiceName, "test-service")
		t.Setenv(metrics.EnvMetricsNamespace, "test.namespace")
		t.Setenv(metrics.EnvMetricsInterval, "15s")
		t.Setenv(metrics.EnvExporterInsecure, "true")

		config := metrics.ConfigFromEnv()

		assert.Equal(t, "test-endpoint:4317", config.OTLPEndpoint)
		assert.Equal(t, "test-service", config.ServiceName)
		assert.Equal(t, "test.namespace", config.Namespace)
		assert.Equal(t, 15*time.Second, config.ExportInterval)
		assert.True(t, config.Insecure)
	})

	t.Run("Invalid Values", func(t *testing.T) {
		// Set invalid values for duration and boolean
		t.Setenv(metrics.EnvMetricsInterval, "invalid")
		t.Setenv(metrics.EnvExporterInsecure, "invalid")

		config := metrics.ConfigFromEnv()
		assert.Equal(t, metrics.DefaultExportInterval, config.ExportInterval)
		assert.Equal(t, metrics.DefaultInsecure, config.Insecure)
	})

	t.Run("Empty Environment Variables", func(t *testing.T) {
		// Clear environment variables
		t.Setenv(metrics.EnvOTLPEndpoint, "")
		t.Setenv(metrics.EnvServiceName, "")
		t.Setenv(metrics.EnvMetricsNamespace, "")
		t.Setenv(metrics.EnvMetricsInterval, "")
		t.Setenv(metrics.EnvExporterInsecure, "")

		config := metrics.ConfigFromEnv()

		assert.Equal(t, metrics.DefaultOTLPEndpoint, config.OTLPEndpoint)
		assert.Equal(t, metrics.DefaultServiceName, config.ServiceName)
		assert.Equal(t, metrics.DefaultNamespace, config.Namespace)
		assert.Equal(t, metrics.DefaultExportInterval, config.ExportInterval)
		assert.Equal(t, metrics.DefaultInsecure, config.Insecure)
	})
}
