package http

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gorilla/mux"

	"github.com/donmikel/karma8/applications/server"
	"github.com/donmikel/karma8/applications/server/domain"
)

func NewRouter(svc server.FileService, logger log.Logger) http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/file", PutFileHandler(svc, logger)).Methods(http.MethodPut)
	r.HandleFunc("/file/{filename}", GetFileHandler(svc, logger)).Methods(http.MethodGet)
	return r
}

func PutFileHandler(svc server.FileService, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseMultipartForm(100 << 20)

		file, header, err := r.FormFile("file")
		if err != nil {
			level.Error(logger).Log("msg", "FormFile error",
				"err", err,
			)
			writeErr(w, err, http.StatusInternalServerError)
			return
		}
		defer file.Close()

		if r.ContentLength == -1 {
			level.Error(logger).Log("msg", "wrong ContentLength",
				"err", err,
			)
			writeErr(w, err, http.StatusBadRequest)
			return
		}

		up := domain.File{
			Meta: domain.FileMeta{
				Name:          header.Filename,
				ContentLength: r.ContentLength,
			},
			Body: file,
		}

		err = svc.PutFile(r.Context(), up)
		if err != nil {
			level.Error(logger).Log("msg", "PutFile error",
				"err", err,
			)
			writeErr(w, err, http.StatusInternalServerError)
			return
		}
	}
}

func GetFileHandler(svc server.FileService, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := mux.Vars(r)["filename"]
		if filename == "" {
			writeErr(w, errors.New("empty filename"), http.StatusBadRequest)
			return
		}

		file, err := svc.GetFile(r.Context(), filename)
		if err != nil {
			writeErr(w, err, http.StatusInternalServerError)
			return
		}
		defer file.Body.Close()

		w.Header().Set("Content-Length", strconv.FormatInt(file.Meta.ContentLength, 10))

		if _, err = io.Copy(w, file.Body); err != nil {
			level.Error(logger).Log("msg", "error body copy", "err", err)
			//writeErr(w, err, http.StatusInternalServerError)
			return
		}
	}
}

func writeErr(w http.ResponseWriter, err error, status int) {
	w.WriteHeader(status)
	_, err = w.Write([]byte(err.Error()))
	if err != nil {
		fmt.Println("can't write response ", err)
	}
}
