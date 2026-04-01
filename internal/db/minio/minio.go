package minio

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type ClientOptions struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
}

func Client(opts ClientOptions) (*minio.Client, error) {
	client, err := minio.New(opts.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(opts.AccessKey, opts.SecretKey, ""),
		Secure: false, // ставим true, если есть SSL
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, opts.Bucket)
	if err != nil {
		return nil, err
	}

	if !exists {
		err = client.MakeBucket(ctx, opts.Bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, err
		}
	}

	return client, nil
}
