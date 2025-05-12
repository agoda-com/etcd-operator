package resources

import (
	"maps"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ServiceAccountBuilder struct {
	*corev1.ServiceAccount
}

type SecretBuilder struct {
	*corev1.Secret
}

func (b *Builder) ServiceAccount(names ...string) ServiceAccountBuilder {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.name(names),
			Namespace: b.owner.GetNamespace(),
		},
	}
	b.add(sa)

	return ServiceAccountBuilder{
		ServiceAccount: sa,
	}
}

func (b *Builder) Secret(names ...string) SecretBuilder {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.name(names),
			Namespace: b.owner.GetNamespace(),
		},
	}
	b.add(secret)

	return SecretBuilder{
		Secret: secret,
	}
}

func (b SecretBuilder) StringData(data map[string]string) SecretBuilder {
	if b.Secret.StringData == nil {
		b.Secret.StringData = map[string]string{}
	}

	maps.Copy(b.Secret.StringData, data)

	return b
}
