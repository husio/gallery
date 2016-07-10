package handler

import (
	"fmt"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/husio/gallery/gallery/storage"
	"github.com/husio/gallery/sq"
	"github.com/husio/gallery/web"
)

func PhotoList(
	db sq.Selector,
	listImages func(sq.Selector, storage.ImagesOpts) ([]*storage.Image, error),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var tags []storage.KeyValue
		for _, raw := range r.URL.Query()["tag"] {
			rawpair, err := url.QueryUnescape(raw)
			if err != nil {
				log.Printf("cannot unescape %q: %s", raw, err)
				continue
			}
			pair := strings.SplitN(rawpair, "=", 2)
			if len(pair) != 2 {
				continue
			}
			tags = append(tags, storage.KeyValue{
				Key:   pair[0],
				Value: pair[1],
			})
		}

		const perPage = 200
		images, err := listImages(db, storage.ImagesOpts{
			Limit: perPage,
			Tags:  tags,
		})
		if err != nil {
			renderErr(w, err.Error())
			return
		}

		context := struct {
			Title  string
			Images []*storage.Image
		}{
			Title:  "listing",
			Images: images,
		}
		renderOK(w, "photo-list", context)
	}
}

func PhotoUpload(
	db sq.Selector,
	tagGroups func(sq.Selector) ([]*storage.TagGroup, error),
	uploadFile func(fd io.ReadSeeker, tags map[string]string) error,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			tags, err := tagGroups(db)
			if err != nil {
				renderErr(w, err.Error())
				return
			}

			context := struct {
				Title string
				Tags  []*storage.TagGroup
			}{
				Title: "Upload photos",
				Tags:  tags,
			}
			renderOK(w, "upload", context)
			return
		}

		const megabyte = 1e6
		if err := r.ParseMultipartForm(100 * megabyte); err != nil {
			renderErr(w, err.Error())
		}

		tags := make(map[string]string)
		for i := 1; i < 20; i++ {
			name := r.FormValue(fmt.Sprintf("tag_name_%d", i))
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			value := r.FormValue(fmt.Sprintf("tag_value_%d", i))
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}

			tags[name] = value
		}

		for _, f := range r.MultipartForm.File["photos"] {
			fd, err := f.Open()
			if err != nil {
				renderErr(w, err.Error())
				return
			}
			err = uploadFile(fd, tags)
			fd.Close()
			if err != nil {
				renderErr(w, err.Error())
				return
			}

		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func ServePhoto(
	db sq.Getter,
	imageByID func(sq.Getter, string) (*storage.Image, error),
	openImage func(year int, id string) (io.ReadCloser, error),
) web.Handler {
	return func(w http.ResponseWriter, r *http.Request, arg web.PathArg) {
		img, err := imageByID(db, arg(0))
		switch err {
		case nil:
			// all good
		case sq.ErrNotFound:
			renderErr(w, "not found")
			return
		default:
			log.Printf("cannot get %q image: %s", arg(0), err)
			renderErr(w, err.Error())
			return
		}

		if checkLastModified(w, r, img.Created) {
			return
		}

		fd, err := openImage(img.Created.Year(), img.ImageID)
		if err != nil {
			log.Printf("cannot read %q image file: %s", img.ImageID, err)
			renderErr(w, err.Error())
			return
		}
		defer fd.Close()

		w.Header().Set("X-Image-ID", img.ImageID)
		w.Header().Set("X-Image-Width", fmt.Sprint(img.Width))
		w.Header().Set("X-Image-Height", fmt.Sprint(img.Height))
		w.Header().Set("X-Image-Created", img.Created.Format(time.RFC3339))
		w.Header().Set("Content-Type", "image/jpeg")

		if r.URL.Query().Get("resize") == "" {
			io.Copy(w, fd)
			return
		}

		image, err := jpeg.Decode(fd)
		if err != nil {
			log.Printf("cannot read %q image file: %s", img.ImageID, err)
			renderErr(w, err.Error())
			return
		}
		var width, height int
		if _, err := fmt.Sscanf(r.URL.Query().Get("resize"), "%dx%d", &width, &height); err != nil {
			log.Printf("cannot resize %q image: %s", img.ImageID, err)
		} else {
			switch img.Orientation {
			case 1:
				// all good
			case 3:
				image = imaging.Rotate180(image)
			case 8:
				image = imaging.Rotate90(image)
			case 6:
				image = imaging.Rotate270(image)
			default:
				log.Printf("unknown image orientation: %s", img.ImageID)
			}
			image = imaging.Fill(image, width, height, imaging.Center, imaging.Linear)
		}
		imaging.Encode(w, image, imaging.JPEG)
	}
}

func checkLastModified(w http.ResponseWriter, r *http.Request, modtime time.Time) bool {
	// https://golang.org/src/net/http/fs.go#L273
	ms, err := time.Parse(http.TimeFormat, r.Header.Get("If-Modified-Since"))
	if err == nil && modtime.Before(ms.Add(1*time.Second)) {
		h := w.Header()
		delete(h, "Content-Type")
		delete(h, "Content-Length")
		w.WriteHeader(http.StatusNotModified)
		return true
	}
	w.Header().Set("Last-Modified", modtime.UTC().Format(http.TimeFormat))
	return false
}
