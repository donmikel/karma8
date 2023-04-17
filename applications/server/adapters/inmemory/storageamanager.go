package inmemory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"github.com/donmikel/karma8/applications/server/interfaces"
)

type storages []interfaces.Storage

type sm struct {
	hostToStorage map[string]interfaces.Storage
	storages      storages
	m             sync.Mutex
	logger        log.Logger
}

func NewStorageManager(logger log.Logger) interfaces.StorageManager {
	return &sm{
		hostToStorage: map[string]interfaces.Storage{},
		storages:      []interfaces.Storage{},
		logger:        logger,
	}
}

func (s *sm) GetStorages(ctx context.Context, count int) ([]interfaces.Storage, error) {
	s.m.Lock()
	defer s.m.Unlock()

	sort.Sort(s.storages)

	result := make(storages, 0, count)
	storagesCount := len(s.storages)
	for i := storagesCount - 1; i >= storagesCount-count; i-- {
		result = append(result, s.storages[i])
	}

	level.Info(s.logger).Log("msg", "selected storages",
		"storages", result,
	)

	return result, nil
}

func (s *sm) GetStorage(ctx context.Context, storageURL string) (interfaces.Storage, error) {
	s.m.Lock()
	defer s.m.Unlock()

	st, ok := s.hostToStorage[storageURL]
	if !ok {
		return nil, fmt.Errorf("storage with URL = %s not found", storageURL)
	}

	return st, nil
}

func (s *sm) AddStorage(ctx context.Context, storageURL string, st interfaces.Storage) error {
	s.m.Lock()
	defer s.m.Unlock()

	s.storages = append(
		s.storages,
		st,
	)

	s.hostToStorage[storageURL] = st

	return nil
}

func (s storages) Len() int {
	return len(s)
}

func (s storages) Less(i, j int) bool {
	si, err := s[i].GetFreeSpace()
	if err != nil {
		return false
	}
	sj, err := s[j].GetFreeSpace()
	if err != nil {
		return false
	}

	return si < sj
}

func (s storages) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s storages) String() string {
	result := make([]string, 0, len(s))
	for _, storage := range s {
		result = append(result, storage.GetStorageURL())
	}

	return strings.Join(result, ", ")
}
