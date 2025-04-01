package cmd

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"iter"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	cosiapi "sigs.k8s.io/container-object-storage-interface-api/apis"
)

func LoadBucketInfo(name string) (*cosiapi.BucketInfo, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}

	info := &cosiapi.BucketInfo{}
	err = json.Unmarshal([]byte(data), info)
	if err != nil {
		return nil, err
	}

	return info, nil
}

func NewClient(ctx context.Context, bucket *cosiapi.SecretS3) (*s3.Client, error) {
	creds := credentials.NewStaticCredentialsProvider(bucket.AccessKeyID, bucket.AccessSecretKey, "")
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(creds),
		config.WithRegion(bucket.Region),
	)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = &bucket.Endpoint
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
func LatestBackup(objects iter.Seq2[*types.Object, error]) (string, time.Time, error) {
	var (
		key    string
		latest time.Time
	)
	for obj, err := range objects {
		if err != nil {
			return "", time.Time{}, err
		}

		if obj.Key == nil {
			continue
		}

		// skip objects with key not matching backup date format
		name := path.Base(*obj.Key)
		ts, err := time.Parse(DateFormat, name)
		if err != nil {
			continue
		}

		if ts.After(latest) {
			key = *obj.Key
			latest = ts
		}
	}

	return key, latest, nil
}
