package etcd

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/fs"
	"os"
	"path"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	DefaultKeyFile    = "tls.key"
	DefaultCertFile   = "tls.crt"
	DefaultCACertFile = "ca.crt"
)

type Credentials struct {
	Key     []byte
	Cert    []byte
	CACert  []byte
	RenewAt time.Time
}

func TLSConfig(creds *Credentials, err error) (*tls.Config, error) {
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{}
	err = creds.TLSConfig(tlsConfig)
	if err != nil {
		return nil, err
	}

	return tlsConfig, nil
}

func LoadSecret(ctx context.Context, kcl client.Client, key client.ObjectKey) (*Credentials, error) {
	secret := &corev1.Secret{}
	err := kcl.Get(ctx, key, secret)
	if err != nil {
		return nil, err
	}

	return &Credentials{
		Key:    secret.Data[DefaultKeyFile],
		Cert:   secret.Data[DefaultCertFile],
		CACert: secret.Data[DefaultCACertFile],
	}, nil
}

func LoadDir(fsys fs.FS) (*Credentials, error) {
	c := &Credentials{}

	files := map[string]*[]byte{
		DefaultKeyFile:    &c.Key,
		DefaultCertFile:   &c.Cert,
		DefaultCACertFile: &c.CACert,
	}
	for name, out := range files {
		data, err := fs.ReadFile(fsys, name)
		switch {
		case os.IsNotExist(err):
			continue
		case err != nil:
			return nil, err
		default:
			*out = data
		}
	}

	return c, nil
}

func LoadTransportSecurity(ts TransportSecurity) (*Credentials, error) {
	c := &Credentials{}

	files := map[string]*[]byte{
		ts.KeyFile:       &c.Key,
		ts.CertFile:      &c.Cert,
		ts.TrustedCAFile: &c.CACert,
	}
	for name, out := range files {
		data, err := os.ReadFile(name)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			continue
		case err != nil:
			return nil, err
		default:
			*out = data
		}
	}

	return c, nil
}

func (c *Credentials) TLSConfig(tlsConfig *tls.Config) error {
	if c.Cert == nil || c.Key == nil {
		return errors.New("required cert and key")
	}

	var ca *x509.CertPool
	if c.CACert != nil {
		ca = x509.NewCertPool()
		if !ca.AppendCertsFromPEM(c.CACert) {
			return errors.New("setup ca cert pool")
		}

		tlsConfig.RootCAs = ca
	}

	cert, err := tls.X509KeyPair(c.Cert, c.Key)
	if err != nil {
		return err
	}

	tlsConfig.Certificates = []tls.Certificate{cert}

	return nil
}

func (c *Credentials) WriteTransportSecurity(ts TransportSecurity) error {
	err := os.MkdirAll(path.Dir(ts.KeyFile), 0700)
	if err != nil {
		return err
	}

	files := []struct {
		name string
		data []byte
		perm os.FileMode
	}{
		{
			name: ts.KeyFile,
			data: c.Key,
			perm: 0600,
		},
		{
			name: ts.CertFile,
			data: c.Cert,
			perm: 0644,
		},
		{
			name: ts.TrustedCAFile,
			data: c.CACert,
			perm: 0644,
		},
	}
	for _, file := range files {
		// skip write if file didn't change
		data, err := os.ReadFile(file.name)
		switch {
		case err != nil && !errors.Is(err, fs.ErrNotExist):
			return err
		case bytes.Equal(data, file.data):
			continue
		}

		err = os.WriteFile(file.name, file.data, file.perm)
		if err != nil {
			return err
		}
	}

	return nil
}
