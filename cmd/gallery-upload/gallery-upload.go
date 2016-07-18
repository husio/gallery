package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cheggaaa/pb"
)

func main() {
	uploadUrlFl := flag.String("url", "http://localhost:5000/upload", "Upload handler URL")
	tagsFl := flag.String("tags", "", "Coma separated tags")
	flag.Parse()

	photos := flag.Args()
	if len(photos) == 0 {
		flag.PrintDefaults()
		return
	}

	tags := strings.Split(*tagsFl, ",")
	if err := run(*uploadUrlFl, photos, tags); err != nil {
		log.Fatal(err)
	}
}

func run(urlStr string, photos, tags []string) error {
	bar := pb.StartNew(len(photos))
	defer bar.Finish()

	for _, photo := range photos {
		bar.Prefix(filepath.Base(photo))
		if err := upload(urlStr, photo, tags); err != nil {
			return fmt.Errorf("%s: %s", photo, err)
		}
		bar.Increment()
	}

	return nil
}

func upload(urlStr, photoPath string, tags []string) error {
	fd, err := os.Open(photoPath)
	if err != nil {
		return err
	}
	defer fd.Close()

	var buf bytes.Buffer
	body := multipart.NewWriter(&buf)
	wr, err := body.CreateFormFile("photos", filepath.Base(photoPath))
	if err != nil {
		return err
	}

	if _, err := io.Copy(wr, fd); err != nil {
		return fmt.Errorf("cannot write file content: %s", err)
	}

	for i, tag := range tags {
		name := fmt.Sprintf("tag_%d", i+1)
		if err := body.WriteField(name, tag); err != nil {
			return fmt.Errorf("cannot write tag: %s", err)
		}
	}

	ct := body.FormDataContentType()
	body.Close()

	resp, err := http.Post(urlStr, ct, &buf)
	if err != nil {
		return fmt.Errorf("cannot POST: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("response %d: %s", resp.StatusCode, string(b))
	}

	return nil
}
