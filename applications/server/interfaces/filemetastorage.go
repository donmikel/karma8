package interfaces

import (
	"context"

	"github.com/donmikel/karma8/applications/server/domain"
)

type FileMetaStorage interface {
	StartProcessingFileMeta(ctx context.Context, meta domain.FileMeta) error
	CompleteFileMeta(ctx context.Context, id string) error
	GetFileMeta(ctx context.Context, id string) (domain.FileMeta, error)
}
