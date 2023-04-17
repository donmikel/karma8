package interfaces

import (
	"context"
	"io"
)

type Storage interface {
	UploadFilePart(ctx context.Context, path string, body io.Reader) error
	ReadFilePart(ctx context.Context, path string) (io.ReadCloser, error)
	DeleteFilePart(ctx context.Context, path string) error
	GetFreeSpace() (int, error)
	GetStorageURL() string
}

type StorageManager interface {
	GetStorages(ctx context.Context, count int) ([]Storage, error)
	GetStorage(ctx context.Context, storageURL string) (Storage, error)
	AddStorage(ctx context.Context, storageURL string, storage Storage) error
}
