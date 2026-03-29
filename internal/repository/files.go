package repository

import (
	"context"
	"io"

	"github.com/cockroachdb/errors"
	"github.com/minio/minio-go/v7"
)

type File struct {
	client *minio.Client
	bucket string
}

func NewFile(client *minio.Client, bucket string) *File {
	return &File{
		client: client,
		bucket: bucket,
	}
}

type PutOptions struct {
	ObjectName  string
	Reader      io.Reader
	Size        int64
	ContentType string
}

func (s *File) Put(ctx context.Context, opts PutOptions) error {
	_, err := s.client.PutObject(ctx, s.bucket, opts.ObjectName, opts.Reader, opts.Size, minio.PutObjectOptions{
		ContentType: opts.ContentType,
	})
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (s *File) Remove(ctx context.Context, objectName string) error {
	err := s.client.RemoveObject(ctx, s.bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (r *File) Get(ctx context.Context, objectName string) (io.ReadCloser, error) {
	// Сначала проверим, существует ли объект
	_, err := r.client.StatObject(ctx, r.bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Получаем сам объект
	object, err := r.client.GetObject(ctx, r.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return object, nil
}
