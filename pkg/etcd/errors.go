package etcd

import "errors"

var (
	ErrReadConfig      = errors.New("etcd: read config")
	ErrWriteConfig     = errors.New("ercd: write config")
	ErrMarshalConfig   = errors.New("etcd: marshal config")
	ErrUnmarshalConfig = errors.New("etcd: unmarshal config")
)
