package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "github.com/agoda-com/etcd-operator/api/v1"
)

func TestBackup(t *testing.T) {
	ctx := context.Background()
	kcl := kubeClient(t, client.Options{})

	key := client.ObjectKey{
		Namespace: *namespace,
		Name:      *backupSecretName,
	}
	secret := &corev1.Secret{}
	err := kcl.Get(ctx, key, secret)
	switch {
	case client.IgnoreNotFound(err) != nil:
		t.Fatal("get secret:", err)
	case err != nil:
		t.Skipf("secret %q not found", key)
	}

	cluster := createCluster(t, ctx, kcl, 3*time.Minute, apiv1.EtcdClusterSpec{
		Version:   "v3.5.14",
		Replicas:  1,
		Resources: resources,
		Backup: apiv1.BackupSpec{
			SecretName: secret.Name,
		},
	})

	ecl := etcdClient(t, ctx, kcl, cluster)

	// create some data
	k, v := "leela", "turanga"
	if _, err := ecl.KV.Put(ctx, k, v); err != nil {
		t.Fatal("failed to put key:", err)
	}

	// create backup
	key = client.ObjectKey{
		Namespace: cluster.Namespace,
		Name:      cluster.Name + "-backup",
	}
	triggerCronJob(t, ctx, kcl, key, 5*time.Minute)

	// delete data
	if _, err := ecl.KV.Delete(ctx, k); err != nil {
		t.Fatal("failed to delete key:", err)
	}

	// restore from backup
	cluster = createCluster(t, ctx, kcl, 3*time.Minute, apiv1.EtcdClusterSpec{
		Version:   "v3.5.14",
		Replicas:  1,
		Resources: resources,
		Restore: &apiv1.RestoreSpec{
			SecretName: *backupSecretName,
			Prefix:     fmt.Sprintf("%s/%s/", cluster.Namespace, cluster.Name),
		},
	})

	ecl = etcdClient(t, ctx, kcl, cluster)

	// check if value is restored from backup
	resp, err := ecl.Get(ctx, k)
	if err != nil {
		t.Fatalf("get %q: %v", k, err)
	}

	var got string
	if len(resp.Kvs) != 1 {
		t.Fatalf("key %q: expected exactly one value", k)
	}

	if got = string(resp.Kvs[0].Value); got != v {
		t.Errorf("key %q: expected %q, got %q", k, got, v)
	}
}
