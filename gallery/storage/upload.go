package storage

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"image/jpeg"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/husio/gallery/sq"
	"github.com/rwcarlsen/goexif/exif"
)

type Uploader struct {
	db sq.Execer
	fs *FileStore
}

func NewUploader(db sq.Execer, fs *FileStore) *Uploader {
	return &Uploader{
		db: db,
		fs: fs,
	}
}

func (u *Uploader) Upload(fd io.ReadSeeker, tags []string) error {
	now := time.Now()

	image, err := imageMeta(fd)
	if err != nil {
		return fmt.Errorf("cannot extract metadata: %s", err)
	}
	if image.Created.IsZero() {
		image.Created = now
	}

	if _, err := fd.Seek(0, os.SEEK_SET); err != nil {
		return fmt.Errorf("cannot seek: %s", err)
	}

	if err := u.fs.Put(image, fd); err != nil {
		return fmt.Errorf("cannot storage file: %s", err)
	}

	// store image in database
	image, err = CreateImage(u.db, *image)
	switch err {
	case nil:
		// all good
	case sq.ErrConflict:
		// image already exists, nothing more to do here
	default:
		return fmt.Errorf("database error: %s", err)
	}

	for _, name := range tags {
		_, err := CreateTag(u.db, Tag{
			ImageID: image.ImageID,
			Name:    name,
			Created: now,
		})
		if err != nil {
			return fmt.Errorf("database error: %s", err)
		}
	}

	return nil
}

func imageMeta(r io.ReadSeeker) (*Image, error) {
	conf, err := jpeg.DecodeConfig(r)
	if err != nil {
		return nil, fmt.Errorf("cannot decode JPEG: %s", err)
	}

	// compute image hash from image content
	oid := sha256.New()
	if _, err := io.Copy(oid, r); err != nil {
		return nil, fmt.Errorf("cannot compute SHA: %s", err)
	}
	img := Image{
		ImageID: encode(oid),
		Width:   conf.Width,
		Height:  conf.Height,
	}

	if _, err := r.Seek(0, os.SEEK_SET); err != nil {
		return nil, fmt.Errorf("cannot seek: %s", err)
	}
	if meta, err := exif.Decode(r); err != nil {
		log.Printf("cannot extract EXIF metadata: %s", err)
	} else {
		if orientation, err := meta.Get(exif.Orientation); err != nil {
			log.Printf("cannot extract image orientation: %s", err)
		} else {
			if o, err := orientation.Int(0); err != nil {
				log.Printf("cannot format orientation: %s", err)
			} else {
				img.Orientation = o
			}
		}
		if dt, err := meta.Get(exif.DateTimeOriginal); err != nil {
			log.Printf("cannot extract image datetime original: %s", err)
		} else {
			if raw, err := dt.StringVal(); err != nil {
				log.Printf("cannot format datetime original: %s", err)
			} else {
				img.Created, err = time.Parse("2006:01:02 15:04:05", raw)
				if err != nil {
					log.Printf("cannot parse datetime original: %s", err)
				}
			}
		}
	}

	return &img, nil
}

func encode(h hasher) string {
	s := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return strings.TrimRight(s, "=")
}

type hasher interface {
	Sum(b []byte) []byte
}
