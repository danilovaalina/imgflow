package minio

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func Client(endpoint, accessKey, secretKey string) (*minio.Client, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false, // ставим true, если есть SSL
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Проверяем соединение через ListBuckets
	_, err = client.ListBuckets(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to minio")
	}

	return client, nil
}
