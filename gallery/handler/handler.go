package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/husio/gallery/gallery/storage"
	"github.com/husio/gallery/sq"
	"github.com/husio/gallery/web"
)

func PhotoList(
	db sq.Selector,
	listImages func(sq.Selector, storage.ImagesOpts) ([]*storage.Image, error),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const perPage = 20
		offset, _ := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64)
		images, err := listImages(db, storage.ImagesOpts{
			Offset: offset,
			Limit:  perPage,
			Tags:   r.URL.Query()["tag"],
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
	uploadFile func(fd io.ReadSeeker, tags []string) error,
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

		var tags []string
		for i := 1; i < 20; i++ {
			name := r.FormValue(fmt.Sprintf("tag_%d", i))
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			tags = append(tags, name)
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
	openImage func(year, orientation int, id string) (io.ReadCloser, error),
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

		fd, err := openImage(img.Created.Year(), img.Orientation, img.ImageID)
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

		io.Copy(w, fd)
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
