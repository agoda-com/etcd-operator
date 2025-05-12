package resources

import (
	"context"
	"fmt"
	"maps"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func DefaultCopySecret(src, dst *corev1.Secret) {
	dst.Labels = src.Labels
	dst.Annotations = src.Annotations
	dst.Data = maps.Clone(src.Data)

	// dst.Type is immutable
	if dst.Generation == 0 {
		dst.Type = src.Type
	}
}

func CopySecret(ctx context.Context, cl client.Client, src, dst client.ObjectKey, f func(src, dst *corev1.Secret)) (ctrlutil.OperationResult, error) {
	if src == dst {
		return ctrlutil.OperationResultNone, nil
	}

	if f == nil {
		f = DefaultCopySecret
	}

	source := &corev1.Secret{}
	if err := cl.Get(ctx, src, source); err != nil {
		return ctrlutil.OperationResultNone, fmt.Errorf("get src secret: %w", err)
	}
	source = source.DeepCopy()

	target := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: dst.Namespace,
			Name:      dst.Name,
		},
	}

	return ctrlutil.CreateOrPatch(ctx, cl, target, func() error {
		f(source, target)
		return nil
	})
}
