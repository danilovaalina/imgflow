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

func (f *File) Put(ctx context.Context, opts PutOptions) error {
	size := opts.Size
	if size <= 0 {
		size = -1
	}

	_, err := f.client.PutObject(ctx, f.bucket, opts.ObjectName, opts.Reader, size, minio.PutObjectOptions{
		ContentType: opts.ContentType,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (f *File) Remove(ctx context.Context, objectName string) error {
	err := f.client.RemoveObject(ctx, f.bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (f *File) Get(ctx context.Context, objectName string) (io.ReadCloser, error) {
	// Сначала проверим, существует ли объект
	_, err := f.client.StatObject(ctx, f.bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Получаем сам объект
	object, err := f.client.GetObject(ctx, f.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return object, nil
}
