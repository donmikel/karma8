package services

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/donmikel/karma8/applications/server"
	"github.com/donmikel/karma8/applications/server/domain"
	"github.com/donmikel/karma8/applications/server/interfaces"
)

const (
	defaultPartsNumToSplit     = 5
	defaultMinChunkSizeInBytes = 10 * 1024 // 10 kB
)

type service struct {
	fileMetaStorage     interfaces.FileMetaStorage
	storageManager      interfaces.StorageManager
	partsNumToSplit     int
	minChunkSizeInBytes int64
}

func NewService(fileMetaStorage interfaces.FileMetaStorage, storageManager interfaces.StorageManager) server.FileService {
	return &service{
		fileMetaStorage:     fileMetaStorage,
		storageManager:      storageManager,
		partsNumToSplit:     defaultPartsNumToSplit,
		minChunkSizeInBytes: defaultMinChunkSizeInBytes,
	}
}

func (s *service) PutFile(ctx context.Context, file domain.File) error {
	partSizes := s.calculatePartsSize(file.Meta.ContentLength, s.partsNumToSplit)

	storages, err := s.storageManager.GetStorages(ctx, len(partSizes))
	if err != nil {
		return fmt.Errorf("can't get storages error: %w", err)
	}

	fileParts := s.getFileParts(storages, file.Meta, partSizes)
	file.Meta.Parts = fileParts

	if err = s.fileMetaStorage.StartProcessingFileMeta(ctx, file.Meta); err != nil {
		return fmt.Errorf("can't put starting file meta: %w", err)
	}

	for _, filePart := range fileParts {
		body := io.LimitReader(file.Body, filePart.ContentLength)
		storage, err := s.storageManager.GetStorage(ctx, filePart.StorageURL)
		if err != nil {
			return fmt.Errorf("can't get storage error: %w", err)
		}

		if err = storage.UploadFilePart(ctx, filePart.Path, body); err != nil {
			return fmt.Errorf("can't upload file part: %w", err)
		}
	}

	if err = s.fileMetaStorage.CompleteFileMeta(ctx, file.Meta.Name); err != nil {
		return fmt.Errorf("can't complete file meta: %w", err)
	}

	return nil
}

type filePartsReader struct {
	currentPart     int
	storageManger   interfaces.StorageManager
	currentPartBody io.ReadCloser
	meta            domain.FileMeta
	ctx             context.Context
}

func (f *filePartsReader) getNextStorage() error {
	if f.currentPart >= len(f.meta.Parts) {
		return io.EOF
	}

	part := f.meta.Parts[f.currentPart]
	storage, err := f.storageManger.GetStorage(f.ctx, part.StorageURL)
	if err != nil {
		return fmt.Errorf("can't get storage by URL, error: %w", err)
	}

	body, err := storage.ReadFilePart(f.ctx, part.Path)
	if err != nil {
		f.currentPartBody = nil
		return fmt.Errorf("can't read part from storage, error: %w", err)
	}

	f.currentPartBody = body
	f.currentPart++

	return nil
}

func (f *filePartsReader) Read(p []byte) (n int, err error) {
	if f.currentPartBody == nil {
		if err = f.getNextStorage(); err != nil {
			return 0, fmt.Errorf("1. can't get next storage, error: %w", err)
		}
	}

	n, err = f.currentPartBody.Read(p)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			return 0, err
		}

		if err = f.currentPartBody.Close(); err != nil {
			return n, fmt.Errorf("can't close body, error: %w", err)
		}

		if err = f.getNextStorage(); err != nil {
			return n, err //fmt.Errorf("2. can't get next storage, error: %w", err)
		}

		return n, nil
	}

	return n, nil
}

func (f *filePartsReader) Close() error {
	if f.currentPartBody != nil {
		return f.currentPartBody.Close()
	}

	return nil
}

func (s *service) GetFile(ctx context.Context, id string) (domain.File, error) {
	meta, err := s.fileMetaStorage.GetFileMeta(ctx, id)
	if err != nil {
		return domain.File{}, fmt.Errorf("can't get file metadata, error: %w", err)
	}

	return domain.File{
		Meta: meta,
		Body: &filePartsReader{
			storageManger: s.storageManager,
			meta:          meta,
			ctx:           ctx,
		},
	}, nil
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func (s *service) getFileParts(storages []interfaces.Storage, fileMeta domain.FileMeta, partSizes []int64) []domain.FilePart {
	fileParts := make([]domain.FilePart, 0, len(partSizes))

	for i, size := range partSizes {
		fileParts = append(fileParts, domain.FilePart{
			StorageURL:    storages[i].GetStorageURL(),
			Path:          fileMeta.Name,
			ContentLength: size,
		})
	}

	return fileParts
}

func (s *service) calculatePartsSize(total int64, splitCount int) []int64 {
	result := make([]int64, 0, splitCount)

	remain := total
	for remain > 0 {
		partSize := remain / int64(splitCount)
		if remain%int64(splitCount) != 0 {
			partSize++
		}

		if partSize <= s.minChunkSizeInBytes {
			partSize = minInt64(remain, s.minChunkSizeInBytes)
		}

		result = append(result, partSize)

		remain -= partSize
		splitCount--
	}

	return result
}
