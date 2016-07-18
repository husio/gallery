gallery-server:
	@go build --ldflags '-extldflags "-static"' -o gallery-server github.com/husio/gallery

gallery-upload:
	@CGO_ENABLED=0 go build -o gallery-upload github.com/husio/gallery/cmd/gallery-upload


.PHONY: galleryd gallery-upload
