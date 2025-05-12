package backup

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

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/klauspost/pgzip"
	etcdv3 "go.etcd.io/etcd/client/v3"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type BackupTag string

const (
	BackupTagHourly BackupTag = "Hourly"
	BackupTagDaily  BackupTag = "Daily"
)

func Backup(ctx context.Context, ecl *etcdv3.Client, scl *s3.Client, location Location) error {
	if location.Bucket == "" || location.Key == "" {
		return ErrInvalidLocation
	}

	logger := log.FromContext(ctx,
		"bucket", location.Bucket,
		"key", location.Key,
	)

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
	tag := BackupTagHourly

	// check if its first daily backup and set retention accordingly
	prefix := path.Dir(location.Key)
	if prefix != "" {
		ts := time.Now()
		midnight := time.Date(ts.Year(), ts.Month(), ts.Day(), 0, 0, 0, 0, ts.Location())
		obj, err := LatestBackup(ctx, scl, location.Bucket, prefix)
		switch {
		case err != nil:
			return err
		case obj == nil:
			logger.Info("daily backup")
			tag = BackupTagDaily
		case obj.LastModified.Before(midnight):
			logger.Info("daily backup", "latest", obj.LastModified)
			tag = BackupTagDaily
		}
	}

	// upload snapshot
	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:  aws.String(location.Bucket),
		Key:     aws.String(location.Key),
		Tagging: aws.String(fmt.Sprintf("Backup=%s", tag)),
		Body:    f,
	})
	if err != nil {
		logger.Error(err, "upload", "source", compressed)
		return fmt.Errorf("upload snapshot: %w", err)
	}

	logger.Info("uploaded")
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
		Name:    filepath.Base(source),
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
