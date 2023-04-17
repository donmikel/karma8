package domain

import "io"

type FilePart struct {
	StorageURL    string
	Path          string
	ContentLength int64
}

type FileMeta struct {
	Name          string
	Parts         []FilePart
	ContentLength int64
}

type File struct {
	Meta FileMeta
	Body io.ReadCloser
}
