package resources

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RoleBindingBuilder struct {
	*rbacv1.RoleBinding
}

func (b *Builder) RoleBinding(names ...string) RoleBindingBuilder {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.name(names),
			Namespace: b.owner.GetNamespace(),
		},
	}
	b.add(roleBinding)

	return RoleBindingBuilder{
		RoleBinding: roleBinding,
	}
}

func (b RoleBindingBuilder) ServiceAccountSubject(serviceAccount *corev1.ServiceAccount) RoleBindingBuilder {
	b.Subjects = append(b.Subjects, rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      serviceAccount.Name,
		Namespace: serviceAccount.Namespace,
	})

	return b
}

func (b RoleBindingBuilder) ClusterRoleRef(name string) RoleBindingBuilder {
	b.RoleRef = rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     name,
	}

	return b
}
