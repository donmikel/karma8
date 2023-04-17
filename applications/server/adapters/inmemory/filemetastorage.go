package inmemory

import (
	"context"
	"fmt"
	"sync"

	"github.com/donmikel/karma8/applications/server/domain"
	"github.com/donmikel/karma8/applications/server/interfaces"
)

type fileMeta struct {
	meta       domain.FileMeta
	inProgress bool
}

type inMemoryFileMetaStorage struct {
	metaData map[string]fileMeta
	mutex    sync.RWMutex
}

func NewFileMetaStorage() interfaces.FileMetaStorage {
	return &inMemoryFileMetaStorage{
		metaData: map[string]fileMeta{},
	}
}

func (i *inMemoryFileMetaStorage) StartProcessingFileMeta(ctx context.Context, meta domain.FileMeta) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	i.metaData[meta.Name] = fileMeta{
		meta:       meta,
		inProgress: true,
	}

	return nil
}

func (i *inMemoryFileMetaStorage) CompleteFileMeta(ctx context.Context, id string) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	m, ok := i.metaData[id]
	if !ok {
		return fmt.Errorf("file with id = %s not found", id)
	}

	m.inProgress = false

	i.metaData[id] = m

	return nil
}

func (i *inMemoryFileMetaStorage) GetFileMeta(ctx context.Context, id string) (domain.FileMeta, error) {
	i.mutex.RLock()
	defer i.mutex.RUnlock()

	m, ok := i.metaData[id]
	if !ok {
		return domain.FileMeta{}, fmt.Errorf("file with id = %s not found", id)
	}

	return m.meta, nil
}
