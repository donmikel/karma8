package http

import (
	"net/http"

	"github.com/go-kit/log"

	"github.com/donmikel/karma8/applications/server"
	"github.com/donmikel/karma8/applications/server/config"
)

func NewHTTPServer(conf config.Api, fileService server.FileService, logger log.Logger) *http.Server {
	mux := NewRouter(fileService, logger)
	return &http.Server{
		Addr:    conf.HTTPAddr,
		Handler: mux,
	}
}
