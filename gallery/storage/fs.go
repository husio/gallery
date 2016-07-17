package storage

import (
	"encoding/json"
	"fmt"
	"image/jpeg"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
)

type FileStore struct {
	photos     string
	thumbnails string
}

func NewFileStore(photosRoot, thumbnailsRoot string) *FileStore {
	return &FileStore{
		photos:     photosRoot,
		thumbnails: thumbnailsRoot,
	}
}

func (fs *FileStore) Put(img *Image, content io.Reader) error {
	dir := filepath.Join(fs.photos, fmt.Sprint(img.Created.Year()))

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
	dir := filepath.Join(fs.photos, fmt.Sprint(img.Created.Year()))
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

func (fs *FileStore) Read(year, _ int, imageID string) (io.ReadCloser, error) {
	path := filepath.Join(fs.photos, fmt.Sprint(year), imageID+".jpg")
	return os.Open(path)
}

func (fs *FileStore) ReadThumbnail(year, orientation int, imageID string) (io.ReadCloser, error) {
	path := filepath.Join(fs.thumbnails, fmt.Sprint(year), imageID+".jpg")
	fd, err := os.Open(path)
	if err == nil {
		return fd, nil
	}

	img, err := fs.Read(year, orientation, imageID)
	if err != nil {
		return nil, fmt.Errorf("cannot read photo file: %s", err)
	}
	image, err := jpeg.Decode(img)
	img.Close()
	if err != nil {
		return nil, fmt.Errorf("cannot decode image: %s", err)
	}
	switch orientation {
	case 1:
		// all good
	case 3:
		image = imaging.Rotate180(image)
	case 8:
		image = imaging.Rotate90(image)
	case 6:
		image = imaging.Rotate270(image)
	default:
		log.Printf("unknown image orientation: %s", imageID)
	}
	image = imaging.Fill(image, 100, 100, imaging.Center, imaging.Linear)

	os.MkdirAll(filepath.Dir(path), 0777)

	fd, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return nil, fmt.Errorf("cannot store thumbnail: %s", err)
	}
	err = imaging.Encode(fd, image, imaging.JPEG)
	fd.Close()
	if err != nil {
		return nil, fmt.Errorf("cannot write thumbnail: %s", err)
	}

	return os.Open(path)
}

func (fs *FileStore) ReadMeta(year int, imageID string) (*Image, error) {
	path := filepath.Join(fs.photos, fmt.Sprint(year), imageID+".jpg")
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
