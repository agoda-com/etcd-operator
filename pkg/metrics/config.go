package metrics

import (
	"os"
	"strconv"
	"time"
)

// Config holds configuration for OpenTelemetry metrics
type Config struct {
	// OTLPEndpoint is the endpoint for OTLP metrics exporter
	OTLPEndpoint string
	// ServiceName is the name of the service
	ServiceName string
	// Namespace is used as a prefix for metrics names
	Namespace string
	// ExportInterval defines how often metrics are exported
	ExportInterval time.Duration
	// Insecure determines whether to use insecure connection for OTLP exporter
	Insecure bool
}

const (
	DefaultOTLPEndpoint   = "localhost:4317"
	DefaultServiceName    = "fleet-etcd"
	DefaultNamespace      = "fleet.etcd"
	DefaultExportInterval = 30 * time.Second
	DefaultInsecure       = false

	// Environment variable names
	EnvOTLPEndpoint     = "OTEL_EXPORTER_OTLP_ENDPOINT"
	EnvServiceName      = "OTEL_SERVICE_NAME"
	EnvMetricsNamespace = "OTEL_METRICS_NAMESPACE"
	EnvMetricsInterval  = "OTEL_METRICS_EXPORT_INTERVAL"
	EnvExporterInsecure = "OTEL_EXPORTER_OTLP_INSECURE"
)

// DefaultConfig provides sensible defaults
func DefaultConfig() Config {
	return Config{
		OTLPEndpoint:   DefaultOTLPEndpoint,
		ServiceName:    DefaultServiceName,
		Namespace:      DefaultNamespace,
		ExportInterval: DefaultExportInterval,
		Insecure:       DefaultInsecure,
	}
}

// ConfigFromEnv creates a Config from environment variables
func ConfigFromEnv() Config {
	config := DefaultConfig()

	// Configure from environment variables
	if endpoint := os.Getenv(EnvOTLPEndpoint); endpoint != "" {
		config.OTLPEndpoint = endpoint
	}

	if serviceName := os.Getenv(EnvServiceName); serviceName != "" {
		config.ServiceName = serviceName
	}

	if namespace := os.Getenv(EnvMetricsNamespace); namespace != "" {
		config.Namespace = namespace
	}

	if intervalStr := os.Getenv(EnvMetricsInterval); intervalStr != "" {
		if interval, err := time.ParseDuration(intervalStr); err == nil {
			config.ExportInterval = interval
		}
	}

	if insecureStr := os.Getenv(EnvExporterInsecure); insecureStr != "" {
		if insecure, err := strconv.ParseBool(insecureStr); err == nil {
			config.Insecure = insecure
		}
	}

	return config
}
