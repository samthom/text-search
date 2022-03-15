package storage

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/notification"
)

// Storage operations interface
// Add, Delete files, Watch bucket for notifications
type Storage interface {
	Add(ctx context.Context, ObjectName string, reader io.Reader, size int64, opts minio.PutObjectOptions) (ETag string, err error)
	Get(ctx context.Context, objectName string) (reader io.Reader, err error)
	GetAll(ctx context.Context) <-chan minio.ObjectInfo
	Delete(ctx context.Context, objectName string) (err error)
	Watch(operations []string) <-chan notification.Info
}

// minioStorage struct
// Concrete type that implements the Storage interface
type minioStorage struct {
	Client     minio.Client
	BucketName string
}

// NewStorage returns new instance of storage
// uses the options to create new instance and return configured session
// @TODO: create empty interface Cmdable add function to mock for mocking purposes in unit testing
func NewStorage(endpoint string, bucketName string, opts *minio.Options) (Storage, error) {
	Client, err := minio.New(endpoint, opts)
	if err != nil {
		return nil, err
	}
	return &minioStorage{
		Client:     *Client,
		BucketName: bucketName,
	}, nil
}

// Add method to add new file to the storage
// returns the ETag of the new object
func (s *minioStorage) Add(ctx context.Context, objectName string, reader io.Reader, size int64, opts minio.PutObjectOptions) (string, error) {
	r, err := s.Client.PutObject(ctx, s.BucketName, objectName, reader, size, opts)
	if err != nil {
		return "", err
	}
	return r.ETag, nil
}

// GetAll method returns a readonly channel
func (s *minioStorage) GetAll(ctx context.Context) <-chan minio.ObjectInfo {
	return s.Client.ListObjects(ctx, s.BucketName, minio.ListObjectsOptions{
		Recursive: true,
	})
}

// Get methods with object name return readable stream of the object requested
func (s *minioStorage) Get(ctx context.Context, objectName string) (io.Reader, error) {
	return s.Client.GetObject(ctx, s.BucketName, objectName, minio.GetObjectOptions{})
}

// Delete methods deletes the object given and returns error if failed
func (s *minioStorage) Delete(ctx context.Context, objectName string) error {
	return s.Client.RemoveObject(ctx, s.BucketName, objectName, minio.RemoveObjectOptions{})
}

// Watch function is for listening the storage system for any operations
// This helps to build index of the file independent of the method of upload
// 	- Accepts the operations to watch as slice of string
// 	 eg: "s3:ObjectCreated:*", "s3:ObjectRemoved"
// 	- Watch returns a readable channel to read notifications
func (s *minioStorage) Watch(operations []string) <-chan notification.Info {
	return s.Client.ListenBucketNotification(context.Background(), s.BucketName, "", "", operations)
}
