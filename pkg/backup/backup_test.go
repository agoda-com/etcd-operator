package backup

import (
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/agoda-com/etcd-operator/pkg/etcd"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	etcdv3 "go.etcd.io/etcd/client/v3"
)

func TestBackupRestore(t *testing.T) {
	if testing.Short() || os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("envtest is not configured")
	}

	env := LoadEnv()
	if len(env) == 0 {
		t.Skip("backup is not configured")
	}

	scl, err := NewClient(t.Context())
	if err != nil {
		t.Fatal("s3 client:", err)
	}

	db := &envtest.Etcd{
		Path: filepath.Join(os.Getenv("KUBEBUILDER_ASSETS"), "etcd"),
	}
	ecl := setupEtcd(t, db)

	location := Location{
		Bucket: os.Getenv("AWS_BUCKET_NAME"),
		Key:    path.Join("backup-test", time.Now().Format(DateFormat)),
	}
	err = Backup(t.Context(), ecl, scl, location)
	if err != nil {
		t.Fatal("backup:", err)
	}

	dataDir := t.TempDir()
	config := &etcd.Config{
		Name:                     "peer0",
		InitialCluster:           "peer0=http://localhost:2380",
		InitialAdvertisePeerURLs: "http://localhost:2380",
		InitialClusterState:      etcd.InitialStateNew,
		InitialClusterToken:      "example",
		DataDir:                  dataDir,
	}
	err = Restore(t.Context(), scl, config, location)
	if err != nil {
		t.Fatal("restore:", err)
	}

	// start etcd from restored data dir
	db = &envtest.Etcd{
		Path:    db.Path,
		DataDir: dataDir,
	}
	setupEtcd(t, db)
}

func setupEtcd(t testing.TB, db *envtest.Etcd) *etcdv3.Client {
	err := db.Start()
	if err != nil {
		t.Fatal("start etcd:", err)
	}
	t.Cleanup(func() {
		err := db.Stop()
		if err != nil {
			t.Error("stop etcd:", err)
		}
	})

	endpoint := db.URL.String()
	ecl, err := etcdv3.New(etcdv3.Config{
		Context:   t.Context(),
		Endpoints: []string{endpoint},
		DialOptions: []grpc.DialOption{
			grpc.WithDisableRetry(),
		},
		Logger: zap.NewNop(),
	})
	if err != nil {
		t.Fatal("etcd client:", err)
	}

	return ecl
}
