package resources

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ServiceAccountBuilder struct {
	*corev1.ServiceAccount
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
