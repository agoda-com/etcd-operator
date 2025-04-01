package sidecar

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/cert-manager/cert-manager/pkg/util/pki"

	apiv1 "github.com/agoda-com/etcd-operator/api/v1"
	"github.com/agoda-com/etcd-operator/pkg/etcd"
	"github.com/agoda-com/etcd-operator/pkg/resources"
)

func (s *Sidecar) GenerateCredentials(ctx context.Context) error {
	logger := log.FromContext(ctx)

	pod := &s.pod
	err := s.kcl.Get(ctx, s.config.ObjectKey, pod)
	if err != nil {
		return err
	}

	cluster, ok := apiv1.ParseCluster(s.pod.Labels)
	if !ok {
		return errors.New("no valid cluster label found")
	}

	start := time.Now()
	renewAt := apiv1.ParseRenewAt(s.pod.Annotations)
	switch {
	case renewAt.After(start):
		return nil
	case !renewAt.IsZero():
		logger.Info("expired", "renewAt", renewAt)
	}

	b := resources.NewBuilder(pod).
		Label(apiv1.ClusterLabel, s.pod.Labels[apiv1.ClusterLabel])

	// peer certificate prototype
	peerCert := b.Certificate("peer").
		Duration(cmv1.DefaultCertificateDuration).
		Issuer(cluster.Name, "peer-ca").
		Usages(cmv1.UsageServerAuth, cmv1.UsageClientAuth).
		IP(s.pod.Status.PodIP)

	// server cert prototype
	serverCert := b.Certificate("server").
		Duration(cmv1.DefaultCertificateDuration).
		Issuer(cluster.Name, "server-ca").
		Usages(cmv1.UsageServerAuth, cmv1.UsageClientAuth).
		IP(s.pod.Status.PodIP).
		DNS(s.pod.Name, cluster.Name, cluster.Namespace, "svc.cluster.local").
		DNS(cluster.Name, cluster.Namespace, "svc.cluster.local").
		IP("127.0.0.1").
		DNS("localhost").
		DNS(s.pod.Name)

	// we only build those objects, they are not applied
	err = b.Build(s.kcl.Scheme())
	if err != nil {
		return fmt.Errorf("build certificates: %w", err)
	}

	wg, wctx := errgroup.WithContext(ctx)

	// generate peer credentials
	wg.Go(func() error {
		creds, err := GenerateCredentials(wctx, s.kcl, peerCert.Certificate, s.config.Interval)
		if err != nil {
			return fmt.Errorf("peer: %w", err)
		}

		// write out the credentials and update config
		err = creds.WriteTransportSecurity(*s.etcdConfig.PeerTransportSecurity)
		if err != nil {
			return fmt.Errorf("write credentials: %w", err)
		}

		logger.Info("generated peer credentials", "cert", s.etcdConfig.PeerTransportSecurity.CertFile)

		return nil
	})

	// generate server credentials
	wg.Go(func() error {
		creds, err := GenerateCredentials(wctx, s.kcl, serverCert.Certificate, s.config.Interval)
		if err != nil {
			return fmt.Errorf("server: %w", err)
		}

		// write out the credentials and update config
		err = creds.WriteTransportSecurity(*s.etcdConfig.ClientTransportSecurity)
		if err != nil {
			return fmt.Errorf("write credentials: %w", err)
		}

		renewAt = creds.RenewAt
		logger.Info("generated server credentials", "cert", s.etcdConfig.ClientTransportSecurity.CertFile, "renewAt", renewAt)

		return creds.TLSConfig(&s.tlsConfig)
	})

	err = wg.Wait()
	if err != nil {
		return err
	}

	// update renew-at annotation
	if !renewAt.IsZero() {
		patch := client.StrategicMergeFrom(s.pod.DeepCopy())
		if s.pod.Annotations == nil {
			s.pod.Annotations = map[string]string{}
		}
		s.pod.Annotations[apiv1.RenewAtAnnotation] = apiv1.FormatRenewAt(renewAt)
		err = s.kcl.Patch(ctx, pod, patch)
		if err != nil {
			return fmt.Errorf("patch pod annotations: %w", err)
		}
	}

	// try to restart etcd container if CA has changed
	// https://github.com/etcd-io/etcd/pull/16500 is stuck in purgatory
	reload := false
	caFiles := []string{
		s.etcdConfig.ClientTransportSecurity.TrustedCAFile,
		s.etcdConfig.PeerTransportSecurity.TrustedCAFile,
	}
	for _, name := range caFiles {
		info, err := os.Stat(name)
		if err != nil {
			return fmt.Errorf("stat ca: %w", err)
		}

		if info.ModTime().After(start) {
			reload = true
			break
		}
	}
	if reload {
		err = syscall.Kill(1, syscall.SIGKILL)
		if err != nil && !errors.Is(err, syscall.ENOENT) {
			logger.Error(err, "restart etcd container")
		}
	}

	return nil
}

// GenerateCredentials generates tls credentials using CertificateRequest created from provided Certificate
func GenerateCredentials(ctx context.Context, kcl client.Client, crt *cmv1.Certificate, interval time.Duration) (*etcd.Credentials, error) {
	pk, err := pki.GeneratePrivateKeyForCertificate(crt)
	if err != nil {
		return nil, err
	}

	pkData, err := pki.EncodePrivateKey(pk, cmv1.PKCS1)
	if err != nil {
		return nil, err
	}

	cr, err := GenerateCertificateRequest(crt, pk)
	if err != nil {
		return nil, fmt.Errorf("build certificate request: %w", err)
	}

	if cr.Name != "" {
		err = kcl.Delete(ctx, cr)
		if client.IgnoreNotFound(err) != nil {
			return nil, err
		}
	}

	err = kcl.Create(ctx, cr)
	if err != nil {
		return nil, err
	}
	cr = cr.DeepCopy()

	err = Poll(ctx, kcl, cr, interval, CertificateRequestReady)
	if err != nil {
		return nil, err
	}

	return &etcd.Credentials{
		Key:     pkData,
		Cert:    cr.Status.Certificate,
		CACert:  cr.Status.CA,
		RenewAt: cr.CreationTimestamp.Add(cr.Spec.Duration.Duration * 3 / 4),
	}, nil
}

func GenerateCertificateRequest(crt *cmv1.Certificate, signer crypto.Signer) (*cmv1.CertificateRequest, error) {
	csr, err := pki.GenerateCSR(crt)
	if err != nil {
		return nil, err
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, csr, signer)
	if err != nil {
		return nil, fmt.Errorf("error creating x509 certificate: %s", err.Error())
	}

	request := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})
	return &cmv1.CertificateRequest{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       crt.Namespace,
			Name:            crt.Name,
			GenerateName:    crt.GenerateName,
			Labels:          crt.Labels,
			Annotations:     crt.Annotations,
			OwnerReferences: crt.OwnerReferences,
		},
		Spec: cmv1.CertificateRequestSpec{
			Duration:  crt.Spec.Duration,
			IssuerRef: crt.Spec.IssuerRef,
			Usages:    crt.Spec.Usages,
			Request:   request,
		},
	}, nil
}

func CertificateRequestReady(cr *cmv1.CertificateRequest) (bool, error) {
	for _, cond := range cr.Status.Conditions {
		switch {
		case cond.Status != cmmetav1.ConditionTrue:
			continue
		case cond.Type == cmv1.CertificateRequestConditionInvalidRequest || cond.Type == cmv1.CertificateRequestConditionDenied:
			return true, fmt.Errorf("certificate request rejected: %s", cond.Type)
		case cond.Type == cmv1.CertificateRequestConditionReady:
			return true, nil
		}
	}

	return false, nil
}
