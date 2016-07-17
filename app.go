package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/husio/gallery/gallery/handler"
	"github.com/husio/gallery/gallery/storage"
	"github.com/husio/gallery/web"

	"github.com/husio/x/envconf"
	"github.com/jmoiron/sqlx"
)

type configuration struct {
	HTTP         string
	Database     string
	UploadDir    string
	ThumbnailDir string
}

func main() {
	conf := configuration{
		HTTP:         "localhost:5000",
		Database:     "/tmp/gallery/db.sqlite3",
		UploadDir:    "/tmp/gallery/photos",
		ThumbnailDir: "/tmp/gallery/thumbnails",
	}
	envconf.Must(envconf.LoadEnv(&conf))

	os.MkdirAll(conf.UploadDir, 0777)
	os.MkdirAll(conf.ThumbnailDir, 0777)
	os.MkdirAll(filepath.Dir(conf.Database), 0777)

	if err := run(conf); err != nil {
		log.Fatalf("application error: %s", err)
	}
}

func run(conf configuration) error {
	db, err := sqlx.Open("sqlite3", conf.Database)
	if err != nil {
		return fmt.Errorf("cannot open database: %s", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return fmt.Errorf("cannot ping database: %s", err)
	}

	fs := storage.NewFileStore(conf.UploadDir, conf.ThumbnailDir)
	uploader := storage.NewUploader(db, fs)

	rt := web.NewRouter()
	rt.Add(`/`, "GET", handler.PhotoList(db, storage.Images))
	rt.Add(`/upload`, "GET,POST", handler.PhotoUpload(db, storage.TagGroups, uploader.Upload))
	rt.Add(`/photo/(name)`, "GET", handler.ServePhoto(db, storage.ImageByID, fs.Read))
	rt.Add(`/thumbnail/(name)\.jpg`, "GET", handler.ServePhoto(db, storage.ImageByID, fs.ReadThumbnail))

	log.Printf("running HTTP server: %s", conf.HTTP)
	if err := http.ListenAndServe(conf.HTTP, rt); err != nil {
		return fmt.Errorf("server error: %s", err)
	}
	return nil
}
