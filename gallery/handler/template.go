package handler

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
)

func renderOK(w http.ResponseWriter, template string, context interface{}) {
	render(w, http.StatusOK, template, context)
}

func render(w http.ResponseWriter, code int, template string, context interface{}) {
	var b bytes.Buffer

	if err := tmpl.ExecuteTemplate(&b, template, context); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "cannot render %q template:\n%s\n", template, err)
		return
	}

	w.WriteHeader(code)
	b.WriteTo(w)
}

func renderErr(w http.ResponseWriter, text string) {
	context := struct {
		Title string
		Text  string
	}{
		Title: "error",
		Text:  text,
	}
	render(w, http.StatusInternalServerError, "error", context)
}

var tmpl = template.Must(template.New("").Parse(`

{{define "header" -}}
<!DOCTYPE html>
<html lang="en">
        <head>
                <meta charset="utf-8">
                <meta http-equiv="X-UA-Compatible" content="IE=edge">
                <meta name="viewport" content="width=device-width, initial-scale=1">
                <title>Gallery{{if .Title}}: {{.Title}}{{end}}</title>
        </head>
{{end}}


{{define "error"}}
        {{template "header" .}}
        <body>
                <div>{{.Text}}</div>
        </body>
</html>
{{end}}


{{define "upload"}}
        {{template "header" .}}
        <body>
                <a href="/">back to listing</a>
                <form enctype="multipart/form-data" action="/upload" method="POST">
                        <h3>1. tag uploaded pictures</h3>
                        <div>
                                <input type="text" name="tag_name_1" placeholder="tag name" value="description">
                                <input type="text" name="tag_value_1"  placeholder="eg. Holiday in Korea or Weekend in Gdansk" autofocus>
                        </div>
                        <div>
                                <input type="text" name="tag_name_2" placeholder="tag name" value="place">
                                <input type="text" name="tag_value_2" placeholder="eg. Croatia, Jeju or Berlin">
                        </div>
                        <div>
                                <input type="text" name="tag_name_3" placeholder="tag name">
                                <input type="text" name="tag_value_3" placeholder="value">
                        </div>
                        <div>
                                <input type="text" name="tag_name_3" placeholder="tag name">
                                <input type="text" name="tag_value_3" placeholder="value">
                        </div>
                        <h5>Exising tags</h5>
                        {{range .Tags}}
                                <div><a href="/?tag={{.Name}}%3D{{.Value}}">{{.Name}} = {{.Value}}</a> {{.Count}}</div>
                        {{end}}

                        <h3>2. select files to upload</h3>
                        <div>
                                <input type="file" name="photos" multiple="multiple" accept=".jpg,.png">
                        </div>

                        <h3>3. upload</h3>
                        <input type="submit" value="upload">
                </form>
        </body>
</html>
{{end}}


{{define "photo-list"}}
        {{template "header" .}}
        <body>
                <div>
                        <a href="/upload">Upload photos</a>
                </div>
                <div>
                        Filter photos
                        <form action="/" method="GET">
                                <input type="search" name="tag" placeholder="tag in format name=value" required>
                                <input type="submit" value="Search">
                        </form>
                </div>
                {{range .Images}}
                        <a href="/photo/{{.ImageID}}">
                                <img
                                        src="/photo/{{.ImageID}}?resize=100x100"
                                        title="{{.Created}}">
                        </a>
                {{else}}
                        <div>No photos</div>
                {{end}}
        </body>
</html>
{{end}}

`))
