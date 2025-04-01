package cmd

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/klauspost/pgzip"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"go.etcd.io/etcd/etcdutl/v3/snapshot"

	"github.com/agoda-com/etcd-operator/pkg/etcd"
)

func RestoreCommand() *cobra.Command {
	config := &etcd.Config{
		Name:                     os.Getenv("ETCD_NAME"),
		DataDir:                  os.Getenv("ETCD_DATA_DIR"),
		InitialCluster:           os.Getenv("ETCD_INITIAL_CLUSTER"),
		InitialClusterToken:      os.Getenv("ETCD_INITIAL_CLUSTER_TOKEN"),
		InitialClusterState:      etcd.InitialState(os.Getenv("ETCD_INITIAL_CLUSTER_STATE")),
		InitialAdvertisePeerURLs: os.Getenv("ETCD_INITIAL_ADVERTISE_PEER_URLS"),
	}

	cmd := &cobra.Command{
		Short: "Restore database from bucket object.",
		Long:  "When prefix is specified latest backup file will be used.",
		Use:   "restore [--config=FILE] [--bucket-info=FILE] [--prefix=PREFIX | --key=KEY]",
	}

	flags := cmd.Flags()

	bucketInfoPath := flags.String("bucket-info", "", "object storage bucket info file")
	configPath := flags.String("config", "", "ETCD config file path")

	params := RestoreParams{}
	flags.StringVar(&params.Key, "key", "", "S3 backup object")
	flags.StringVar(&params.Prefix, "prefix", "", "S3 backup object prefix to search for latest backup")

	_ = cmd.MarkFlagRequired("config")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if (params.Key == "" && params.Prefix == "") || (params.Key != "" && params.Prefix != "") {
			return errors.New("either --prefix or --key have to be specified")
		}

		bucketInfo, err := LoadBucketInfo(*bucketInfoPath)
		if err != nil {
			return fmt.Errorf("load bucket: %w", err)
		}
		params.Bucket = bucketInfo.Spec.BucketName

		scl, err := NewClient(ctx, bucketInfo.Spec.S3)
		if err != nil {
			return err
		}

		err = etcd.LoadConfig(*configPath, config)
		if err != nil {
			return err
		}

		return Restore(ctx, scl, config, params)
	}

	return cmd
}

type RestoreParams struct {
	Bucket string
	Key    string
	Prefix string
}

func Restore(ctx context.Context, scl *s3.Client, config *etcd.Config, params RestoreParams) error {
	logger := log.FromContext(ctx)

	if config.InitialClusterState != etcd.InitialStateNew {
		logger.Info("skipping existing cluster")
		return nil
	}

	// if key not found find latest backup by prefix
	// first file matching backup date format will be latest as they are ordered lexicographically
	if params.Key == "" {
		key, _, err := LatestBackup(ListObjects(ctx, scl, params.Bucket, params.Prefix))
		switch {
		case err != nil:
			return fmt.Errorf("latest backup: %w", err)
		case key == "":
			return errors.New("backup not found")
		}

		params.Key = key
		logger.Info("using latest backup", "key", params.Key)
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
	err = DownloadSnapshot(ctx, scl, compressed, params)
	if err != nil {
		return fmt.Errorf("download snapshot: %w", err)
	}

	logger.Info("downloaded snapshot",
		"bucket", params.Bucket,
		"key", params.Key,
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

func DownloadSnapshot(ctx context.Context, client manager.DownloadAPIClient, target string, params RestoreParams) error {
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
		Bucket: aws.String(params.Bucket),
		Key:    aws.String(params.Key),
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
