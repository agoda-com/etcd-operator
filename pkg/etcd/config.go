package etcd

import (
	"encoding/json"
	"os"
	"time"

	"go.uber.org/zap"
)

type Config struct {
	Name    string `json:"name,omitempty"`
	DataDir string `json:"data-dir,omitempty"`
	WALDir  string `json:"wal-dir,omitempty"`

	SnapshotCount     int64 `json:"snapshot-count,omitempty"`
	HeartbeatInterval int64 `json:"heartbeat-interval,omitempty"`
	ElectionTimeout   int64 `json:"election-timeout,omitempty"`
	QuotaBackendBytes int64 `json:"quota-backend-bytes,omitempty"`

	ListenPeerURLs    string `json:"listen-peer-urls,omitempty"`
	ListenClientURLs  string `json:"listen-client-urls,omitempty"`
	ListenMetricsURLs string `json:"listen-metrics-urls,omitempty"`

	InitialAdvertisePeerURLs string `json:"initial-advertise-peer-urls,omitempty"`
	AdvertiseClientURLs      string `json:"advertise-client-urls,omitempty"`

	InitialCluster      string       `json:"initial-cluster,omitempty"`
	InitialClusterToken string       `json:"initial-cluster-token,omitempty"`
	InitialClusterState InitialState `json:"initial-cluster-state,omitempty"`

	ClientTransportSecurity *TransportSecurity `json:"client-transport-security,omitempty"`
	PeerTransportSecurity   *TransportSecurity `json:"peer-transport-security,omitempty"`

	StrictReconfigCheck bool     `json:"strict-reconfig-check,omitempty"`
	EnablePProf         bool     `json:"enable-pprof,omitempty"`
	LogLevel            LogLevel `json:"log-level,omitempty"`

	AutoCompactionMode      string `json:"auto-compaction-mode,omitempty"`
	AutoCompactionRetention string `json:"auto-compaction-retention,omitempty"`

	ExpInitialCorruptCheck         bool          `json:"experimental-initial-corrupt-check,omitempty"`
	ExpWatchProgressNotifyInterval time.Duration `json:"experimental-watch-progress-notify-interval,omitempty"`
}

type TransportSecurity struct {
	CertFile       string `json:"cert-file"`
	KeyFile        string `json:"key-file"`
	ClientCertAuth bool   `json:"client-cert-auth"`
	TrustedCAFile  string `json:"trusted-ca-file,omitempty"`
	AutoTLS        bool   `json:"auto-tls"`
}

type InitialState string

const (
	InitialStateNew       InitialState = "new"
	InitialStateExisiting InitialState = "existing"
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelPanic LogLevel = "panic"
	LogLevelFatal LogLevel = "fatal"
)

func LoadConfig(name string, config *Config) error {
	data, err := os.ReadFile(name)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, config)
}

type ConnectOpt func(c *connectConfig)

func WithLogger(logger *zap.Logger) ConnectOpt {
	return func(c *connectConfig) {
		c.logger = logger
	}
}

func WithDialTimeout(timeout time.Duration) ConnectOpt {
	return func(c *connectConfig) {
		c.dialTimeout = timeout
	}
}

func (c Config) IsZero() bool {
	return c == Config{}
}

type connectConfig struct {
	logger      *zap.Logger
	dialTimeout time.Duration
}

var defaultConfig = connectConfig{
	dialTimeout: 5 * time.Second,
}
