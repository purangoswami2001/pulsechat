package storage

import "io"

type Storage interface {
	Save(filename string, r io.Reader) (string, error)
	Delete(filename string) error
}
