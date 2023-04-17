package server

import (
	"context"

	"github.com/donmikel/karma8/applications/server/domain"
)

type FileService interface {
	PutFile(ctx context.Context, file domain.File) error
	GetFile(ctx context.Context, id string) (domain.File, error)
}
