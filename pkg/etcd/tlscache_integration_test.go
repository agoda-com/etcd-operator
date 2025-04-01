package etcd

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	_ "embed"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	//go:embed testdata/existing.json
	existingData []byte

	//go:embed testdata/updated.json
	updatedData []byte
)

func TestCacheTLSSource(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	if testing.Short() || os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.SkipNow()
	}

	env := &envtest.Environment{}
	kubeconfig, err := env.Start()
	if err != nil {
		t.Fatal("start envtest: ", err)
	}
	t.Cleanup(func() {
		if err := env.Stop(); err != nil {
			t.Log("stop envtest: ", err)
		}
	})

	kcl, err := client.New(kubeconfig, client.Options{})
	if err != nil {
		t.Fatal("create client: ", err)
	}

	key := client.ObjectKey{
		Namespace: "default",
		Name:      "test-secret",
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: key.Namespace,
			Name:      key.Name,
		},
		Type: corev1.SecretTypeTLS,
		Data: parseData(t, existingData),
	}
	err = kcl.Create(ctx, secret)
	if err != nil {
		t.Fatal("create secret:", err)
	}

	cache, err := NewTLSCache(kcl, 1000)
	if err != nil {
		t.Fatal("cache:", err)
	}

	t.Run("secret not found", func(t *testing.T) {
		key := client.ObjectKey{Namespace: "default", Name: "imaginary"}
		_, err := cache.Get(ctx, key)
		if !apierrors.IsNotFound(err) {
			t.Fatal("expected not found, got:", err)
		}
	})

	t.Run("subsequent calls return same config", func(t *testing.T) {
		key := client.ObjectKeyFromObject(secret)
		c1, err := cache.Get(ctx, key)
		if err != nil {
			t.Fatal("source:", err)
		}

		c2, err := cache.Get(ctx, key)
		if err != nil {
			t.Fatal("source:", err)
		}

		if c1 != c2 {
			t.Fatal("expected to get same config")
		}
	})

	t.Run("updating secret should return new config", func(t *testing.T) {
		key := client.ObjectKeyFromObject(secret)
		c1, err := cache.Get(ctx, key)
		if err != nil {
			t.Fatal("source:", err)
		}

		secret = secret.DeepCopy()
		patch := client.MergeFrom(secret.DeepCopy())
		secret.Data = parseData(t, updatedData)
		if err := kcl.Patch(ctx, secret, patch); err != nil {
			t.Fatal("update:", err)
		}

		c2, err := cache.Get(ctx, key)
		if err != nil {
			t.Fatal("source:", err)
		}

		if c1 == c2 {
			t.Fatal("expected to get different config")
		}
	})
}

func parseData(t testing.TB, src []byte) map[string][]byte {
	t.Helper()

	dst := make(map[string][]byte)
	if err := json.Unmarshal(src, &dst); err != nil {
		t.Fatal("unmarshal:", err)
	}

	return dst
}
