package etcd

import (
	"context"
	"crypto/tls"
	"fmt"

	etcdv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func Connect(ctx context.Context, tlsConfig *tls.Config, endpoint string, opts ...ConnectOpt) (*etcdv3.Client, error) {
	conf := defaultConfig
	for _, opt := range opts {
		opt(&conf)
	}

	if endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}

	if conf.logger == nil {
		conf.logger = zap.NewNop()
	}

	return etcdv3.New(etcdv3.Config{
		Context:     ctx,
		Endpoints:   []string{endpoint},
		TLS:         tlsConfig,
		DialTimeout: conf.dialTimeout,
		DialOptions: []grpc.DialOption{
			grpc.WithDisableRetry(),
		},
		Logger: conf.logger,
	})
}
