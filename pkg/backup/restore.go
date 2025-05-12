package backup

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/agoda-com/etcd-operator/pkg/etcd"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/klauspost/pgzip"
	"go.etcd.io/etcd/etcdutl/v3/snapshot"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func Restore(ctx context.Context, scl *s3.Client, config *etcd.Config, location Location) error {
	if location.Bucket == "" || location.Key == "" {
		return ErrInvalidLocation
	}

	logger := log.FromContext(ctx,
		"bucket", location.Bucket,
		"key", location.Key,
	)

	if config.InitialClusterState != etcd.InitialStateNew {
		logger.Info("skipping existing cluster")
		return nil
	}

	dir, err := os.MkdirTemp(os.TempDir(), "restore.*")
	if err != nil {
		return err
	}
	defer func() {
		err := os.RemoveAll(dir)
		if err != nil {
			logger.Error(err, "remove temporary directory", "name", dir)
		}
	}()

	compressed := filepath.Join(dir, "snapshot.tar.gz")
	err = DownloadSnapshot(ctx, scl, compressed, location)
	if err != nil {
		return fmt.Errorf("download snapshot: %w", err)
	}

	logger.Info("downloaded snapshot",
		"target", compressed,
	)

	decompressed := filepath.Join(dir, "snapshot.db")
	err = DecompressSnapshot(compressed, decompressed)
	if err != nil {
		return fmt.Errorf("decompress %q: %w", compressed, err)
	}

	logger.Info("decompressed snapshot",
		"source", compressed,
		"target", decompressed,
	)

	sm := snapshot.NewV3(zap.NewNop())
	err = sm.Restore(snapshot.RestoreConfig{
		SnapshotPath:        decompressed,
		Name:                config.Name,
		OutputDataDir:       config.DataDir,
		PeerURLs:            strings.Split(config.InitialAdvertisePeerURLs, ","),
		InitialCluster:      config.InitialCluster,
		InitialClusterToken: config.InitialClusterToken,
	})
	if err != nil {
		return fmt.Errorf("restore %q: %w", decompressed, err)
	}

	logger.Info("restored from snapshot",
		"snapshot", decompressed,
		"data", config.DataDir,
	)

	return nil
}

func DownloadSnapshot(ctx context.Context, client manager.DownloadAPIClient, target string, location Location) error {
	// create target file
	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()

	// download snapshot into file
	downloader := manager.NewDownloader(client)
	_, err = downloader.Download(ctx, f, &s3.GetObjectInput{
		Bucket: aws.String(location.Bucket),
		Key:    aws.String(location.Key),
	})
	if err != nil {
		return err
	}

	return nil
}

func DecompressSnapshot(source, target string) error {
	reader, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, reader.Close())
	}()

	gzipReader, err := pgzip.NewReader(reader)
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			return err
		}

		// found a snapshot file
		if header.Name == "snapshot.db" {
			break
		}
	}

	writer, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("create target file: %w", err)
	}
	defer func() {
		err = errors.Join(err, writer.Close())
	}()

	_, err = io.Copy(writer, tarReader)
	return err
}
