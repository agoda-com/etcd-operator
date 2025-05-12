package cmd

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/klauspost/pgzip"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	etcdv3 "go.etcd.io/etcd/client/v3"

	"github.com/agoda-com/etcd-operator/pkg/etcd"
)

const (
	// Date format is default backup file name format
	// Generated timestamps are always in UTC
	DateFormat = "20060102150405"

	BackupTagHourly = "Hourly"
	BackupTagDaily  = "Daily"
)

func BackupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Short: "Backup cluster",
		Use:   "backup [--credentials-dir DIR] [--endpoint ENDPOINT] [--bucket-info FILE] [--key KEY | --prefix PREFIX] [--retention DURATION]",
	}

	flags := cmd.Flags()

	endpoint := flags.String("endpoint", "", "etcd endpoint")
	credentialsDir := flags.String("credentials-dir", "", "etcd credentials directory")
	bucketInfoPath := flags.String("bucket-info", "", "object storage bucket info file")

	params := BackupParams{}
	flags.StringVar(&params.Key, "key", "", "object key")
	flags.StringVar(&params.Prefix, "prefix", "", "object prefix")
	flags.DurationVar(&params.Retention, "retention", 0, "object retention")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		tlsConfig, err := etcd.TLSConfig(etcd.LoadDir(os.DirFS(*credentialsDir)))
		if err != nil {
			return err
		}

		ecl, err := etcd.Connect(ctx, tlsConfig, *endpoint)
		if err != nil {
			return fmt.Errorf("connect etcd: %w", err)
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

		return Backup(ctx, ecl, scl, params)
	}

	return cmd
}

type BackupParams struct {
	Bucket    string
	Key       string
	Prefix    string
	Retention time.Duration
}

func Backup(ctx context.Context, ecl *etcdv3.Client, scl *s3.Client, params BackupParams) error {
	logger := log.FromContext(ctx)

	if params.Key == "" {
		ts := time.Now().Format(DateFormat)
		params.Key = path.Join(params.Prefix, ts)
	}

	dir, err := os.MkdirTemp(os.TempDir(), "backup.*")
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, os.RemoveAll(dir))
	}()

	uncompressed := filepath.Join(dir, "snapshot.db")
	err = SaveSnapshot(ctx, ecl, uncompressed)
	if err != nil {
		return fmt.Errorf("save snapshot to %q: %w", uncompressed, err)
	}

	logger.Info("saved snapshot", "target", uncompressed)

	compressed := filepath.Join(dir, "snapshot.tar.gz")
	err = Compress(uncompressed, compressed)
	if err != nil {
		return fmt.Errorf("compress %q: %w", uncompressed, err)
	}

	logger.Info("compressed",
		"source", uncompressed,
		"target", compressed,
	)

	f, err := os.Open(compressed)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()

	uploader := manager.NewUploader(scl)
	retention := params.Retention
	tag := BackupTagHourly

	// check if its first daily backup and set retention accordingly
	ts := time.Now()
	midnight := time.Date(ts.Year(), ts.Month(), ts.Day(), 0, 0, 0, 0, ts.Location())
	_, latest, err := LatestBackup(ListObjects(ctx, scl, params.Bucket, params.Prefix))
	switch {
	case err != nil:
		return err
	case latest.Before(midnight):
		logger.Info("daily backup", "latest", latest)
		tag = BackupTagDaily
	}

	logger = logger.WithValues(
		"bucket", params.Bucket,
		"key", params.Key,
	)

	putObjInput := &s3.PutObjectInput{
		Bucket:  aws.String(params.Bucket),
		Key:     aws.String(params.Key),
		Tagging: aws.String(fmt.Sprintf("Backup=%s", tag)),
		Body:    f,
	}
	if retention != 0 {
		putObjInput.Expires = aws.Time(ts.Add(retention))
	}

	// upload snapshot
	_, err = uploader.Upload(ctx, putObjInput)
	if err != nil {
		logger.Error(err, "upload", "source", compressed)
		return fmt.Errorf("upload snapshot: %w", err)
	}

	logger.Info("uploaded", "retention", retention)
	return nil
}

func SaveSnapshot(ctx context.Context, ecl etcdv3.Maintenance, name string) (err error) {
	f, err := os.Create(name)
	if err != nil {
		return err
	}

	// get cluster snapshot
	snapshot, err := ecl.Snapshot(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, snapshot.Close())
	}()

	_, err = io.Copy(f, snapshot)
	return err
}

func Compress(source, target string) (err error) {
	writer, err := os.Create(target)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, writer.Close())
	}()

	gzipWriter := pgzip.NewWriter(writer)
	defer func() {
		err = errors.Join(err, gzipWriter.Close())
	}()

	tarWriter := tar.NewWriter(gzipWriter)
	defer func() {
		err = errors.Join(err, tarWriter.Close())
	}()

	info, err := os.Stat(source)
	if err != nil {
		return err
	}

	err = tarWriter.WriteHeader(&tar.Header{
		Name:    "snapshot.db",
		Size:    info.Size(),
		Mode:    int64(info.Mode()),
		ModTime: info.ModTime(),
	})
	if err != nil {
		return err
	}

	f, err := os.Open(source)
	if err != nil {
		return err
	}

	_, err = io.Copy(tarWriter, f)
	return err
}
