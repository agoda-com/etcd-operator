package resources

import (
	"maps"
	"strings"
	"time"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CertificateBuilder struct{ *cmv1.Certificate }

func (b *Builder) CA(names ...string) {
	b.Certificate(names...).CA().Duration(5 * 365 * 24 * time.Hour)
	b.Issuer(names...)
}

func (b *Builder) SelfSign() *cmv1.Issuer {
	issuer := &cmv1.Issuer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "self-sign",
			Namespace: b.owner.GetNamespace(),
		},
		Spec: cmv1.IssuerSpec{
			IssuerConfig: cmv1.IssuerConfig{
				SelfSigned: &cmv1.SelfSignedIssuer{},
			},
		},
	}
	b.add(issuer)

	return issuer
}

func (b *Builder) Issuer(names ...string) *cmv1.Issuer {
	name := b.name(names)
	issuer := &cmv1.Issuer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: b.owner.GetNamespace(),
		},
		Spec: cmv1.IssuerSpec{
			IssuerConfig: cmv1.IssuerConfig{
				CA: &cmv1.CAIssuer{
					SecretName: name,
				},
			},
		},
	}
	b.add(issuer)

	return issuer
}

func (c CertificateBuilder) CommonName(names ...string) CertificateBuilder {
	c.Spec.CommonName = strings.Join(names, "-")
	return c
}

func (c CertificateBuilder) Issuer(names ...string) CertificateBuilder {
	c.Spec.IssuerRef = cmmeta.ObjectReference{
		Name:  strings.Join(names, "-"),
		Kind:  cmv1.IssuerKind,
		Group: cmv1.SchemeGroupVersion.Group,
	}
	return c
}

func (c CertificateBuilder) CA() CertificateBuilder {
	c.Spec.IsCA = true
	c.Spec.IssuerRef = cmmeta.ObjectReference{
		Name:  "self-sign",
		Kind:  cmv1.ClusterIssuerKind,
		Group: cmv1.SchemeGroupVersion.Group,
	}
	return c
}

func (c CertificateBuilder) Duration(duration time.Duration) CertificateBuilder {
	c.Spec.Duration = &metav1.Duration{Duration: duration}
	return c
}

func (c CertificateBuilder) Usages(usages ...cmv1.KeyUsage) CertificateBuilder {
	c.Spec.Usages = append(c.Spec.Usages, usages...)
	return c
}

func (c CertificateBuilder) DNS(names ...string) CertificateBuilder {
	c.Spec.DNSNames = append(c.Spec.DNSNames, strings.Join(names, "."))
	return c
}

func (c CertificateBuilder) IP(ip string) CertificateBuilder {
	c.Spec.IPAddresses = append(c.Spec.IPAddresses, ip)
	return c
}

func (c CertificateBuilder) Subject(cn string, orgs ...string) CertificateBuilder {
	c.Spec.CommonName = cn
	c.Spec.Subject = &cmv1.X509Subject{Organizations: orgs}
	return c
}

func (c CertificateBuilder) SecretLabels(labels map[string]string) CertificateBuilder {
	if c.Spec.SecretTemplate == nil {
		c.Spec.SecretTemplate = &cmv1.CertificateSecretTemplate{}
	}

	if c.Spec.SecretTemplate.Labels == nil {
		c.Spec.SecretTemplate.Labels = map[string]string{}
	}

	maps.Copy(c.Spec.SecretTemplate.Labels, labels)

	return c
}

func (b *Builder) Certificate(names ...string) CertificateBuilder {
	name := b.name(names)
	cert := &cmv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: b.owner.GetNamespace(),
		},
		Spec: cmv1.CertificateSpec{
			CommonName: name,
			SecretName: name,
			PrivateKey: &cmv1.CertificatePrivateKey{
				Algorithm: cmv1.RSAKeyAlgorithm,
			},
			Usages: []cmv1.KeyUsage{
				cmv1.UsageDigitalSignature,
				cmv1.UsageKeyEncipherment,
			},
		},
	}
	b.add(cert)

	return CertificateBuilder{cert}
}
