package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type FileStore struct {
	root string
}

func NewFileStore(root string) *FileStore {
	return &FileStore{root: root}
}

func (fs *FileStore) Put(img *Image, content io.Reader) error {
	dir := filepath.Join(fs.root, fmt.Sprint(img.Created.Year()))

	os.MkdirAll(dir, 0776)

	imgPath := filepath.Join(dir, fmt.Sprintf("%s.jpg", img.ImageID))
	fd, err := os.OpenFile(imgPath, os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		return fmt.Errorf("cannot create %q: %s", imgPath, err)
	}
	defer fd.Close()
	if _, err := io.Copy(fd, content); err != nil {
		return fmt.Errorf("cannot write image: %s", err)
	}

	if err := fs.PutMeta(img); err != nil {
		return err
	}

	return nil
}

func (fs *FileStore) PutMeta(img *Image) error {
	dir := filepath.Join(fs.root, fmt.Sprint(img.Created.Year()))
	path := filepath.Join(dir, fmt.Sprintf("%s.json", img.ImageID))

	fd, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		return err
	}
	defer fd.Close()
	b, err := json.MarshalIndent(img, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot encode metadata: %s", err)
	}
	if _, err := fd.Write(b); err != nil {
		return fmt.Errorf("cannot write metadata: %s", err)
	}
	return nil
}

func (fs *FileStore) Read(year int, imageID string) (io.ReadCloser, error) {
	path := filepath.Join(fs.root, fmt.Sprint(year), imageID+".jpg")
	return os.Open(path)
}

func (fs *FileStore) ReadMeta(year int, imageID string) (*Image, error) {
	path := filepath.Join(fs.root, fmt.Sprint(year), imageID+".jpg")
	fd, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read meta file: %s", err)
	}
	defer fd.Close()

	var img Image
	if err := json.NewDecoder(fd).Decode(&img); err != nil {
		return nil, fmt.Errorf("cannot decode meta: %s", err)
	}
	return &img, nil
}
