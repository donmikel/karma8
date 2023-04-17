package inmemory

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"github.com/dustin/go-humanize"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"github.com/donmikel/karma8/applications/server/interfaces"
)

const defaultFreeSpaceInBytes = 100 * 1024 * 1024 // 100 Mb

type inMemoryStorage struct {
	dataByPath map[string][]byte
	freeSpace  int
	url        string
	log        log.Logger
	mutex      sync.RWMutex
}

func NewStorage(url string, logger log.Logger) interfaces.Storage {
	return &inMemoryStorage{
		url:        url,
		log:        logger,
		dataByPath: map[string][]byte{},
		freeSpace:  defaultFreeSpaceInBytes,
	}
}

func (m *inMemoryStorage) GetStorageURL() string {
	return m.url
}

func (m *inMemoryStorage) UploadFilePart(ctx context.Context, path string, body io.Reader) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()
	dataLen := binary.Size(data)
	if dataLen > m.freeSpace {
		return fmt.Errorf("not enough free space")
	}

	m.dataByPath[path] = data
	m.freeSpace -= dataLen

	level.Info(m.log).Log("msg", "file part uploaded",
		"path", path,
		"storage", m.url,
		"size", humanize.Bytes(uint64(dataLen)),
		"free_space", humanize.Bytes(uint64(m.freeSpace)),
	)

	return nil
}

func (m *inMemoryStorage) ReadFilePart(ctx context.Context, path string) (io.ReadCloser, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	level.Info(m.log).Log("msg", "file part read",
		"path", path,
		"storage", m.url,
	)

	return io.NopCloser(bytes.NewReader(m.dataByPath[path])), nil
}

func (m *inMemoryStorage) DeleteFilePart(ctx context.Context, path string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	dataLen := len(m.dataByPath[path])
	delete(m.dataByPath, path)
	m.freeSpace += dataLen

	return nil
}

func (m *inMemoryStorage) GetFreeSpace() (int, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.freeSpace, nil
}
