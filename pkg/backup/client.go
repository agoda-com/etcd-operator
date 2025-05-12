package backup

import (
	"context"
	"crypto/tls"
	"errors"
	"iter"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const (
	// Date format is default backup file name format
	// Generated timestamps are always in UTC
	DateFormat = "20060102150405"
)

func NewClient(ctx context.Context) (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.HTTPClient = httpClient
		o.UsePathStyle = true
	}), nil
}

// ListObjects returns an iterator over paged results
func ListObjects(ctx context.Context, client manager.ListObjectsV2APIClient, bucket, prefix string) iter.Seq2[*types.Object, error] {
	return func(yield func(*types.Object, error) bool) {
		var token *string
		for {
			resp, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
				Bucket:            aws.String(bucket),
				Prefix:            aws.String(prefix),
				ContinuationToken: token,
			})
			if err != nil {
				yield(nil, err)
				return
			}

			for _, obj := range resp.Contents {
				if !yield(&obj, nil) {
					return
				}
			}

			// advance to next page if available
			token = resp.NextContinuationToken
			if token == nil {
				return
			}
		}
	}
}

// LatestBackup returns first backup object that matches backup date format.
func LatestBackup(ctx context.Context, client manager.ListObjectsV2APIClient, bucket, prefix string) (*types.Object, error) {
	if bucket == "" {
		return nil, errors.New("AWS_BUCKET_NAME is required")
	}

	objects := ListObjects(ctx, client, bucket, prefix)

	var latest *types.Object
	for obj, err := range objects {
		if err != nil {
			return nil, err
		}
		if obj.Key == nil || obj.LastModified == nil {
			continue
		}

		if latest == nil || latest.LastModified.Before(*obj.LastModified) {
			latest = obj
		}
	}

	return latest, nil
}
