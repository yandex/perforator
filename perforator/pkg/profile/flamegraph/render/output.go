package render

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"html/template"
	"io"

	"github.com/yandex/perforator/library/go/core/resource"
)

// WrapJSONInHTML takes flamegraph JSON data and wraps it in the HTML template.
// The JSON is gzip-compressed and base64-encoded before embedding.
func WrapJSONInHTML(jsonData []byte, w io.Writer) error {
	// Compress JSON
	buf := new(bytes.Buffer)
	compressor := gzip.NewWriter(buf)
	if _, err := compressor.Write(jsonData); err != nil {
		return err
	}
	if err := compressor.Close(); err != nil {
		return err
	}

	// Prepare template data
	jsCode := template.HTML("<script>" + string(resource.Get("viewer.js")) + "</script>")
	jsonHtml := template.HTML("<script>window.__data__=\"" + base64.StdEncoding.EncodeToString(buf.Bytes()) + "\"</script>")

	return tmpl.Execute(w, &struct {
		Json   template.HTML
		Script template.HTML
	}{
		Json:   jsonHtml,
		Script: jsCode,
	})
}

// RenderJSONAsHTML renders a JSONRenderer's output as HTML.
func RenderJSONAsHTML(renderer JSONRenderer, w io.Writer) error {
	var jsonBuf bytes.Buffer
	if err := renderer.RenderJSON(&jsonBuf); err != nil {
		return err
	}
	return WrapJSONInHTML(jsonBuf.Bytes(), w)
}
