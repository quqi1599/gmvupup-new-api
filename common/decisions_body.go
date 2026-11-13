package common

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

// DecisionsBody translates SDK multipart images into inline JSON once. Routing
// and relay validation consume the same model and assets, never form controls.
func DecisionsBody(c *gin.Context) ([]byte, error) {
	if v, ok := c.Get("decisions_json_body"); ok {
		return v.([]byte), nil
	}
	storage, err := GetBodyStorage(c)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
		return storage.Bytes()
	}
	form, err := ParseMultipartFormReusable(c)
	if err != nil {
		return nil, err
	}
	defer form.RemoveAll()
	if len(form.Value) != 1 || len(form.Value["request"]) != 1 {
		return nil, errors.New("multipart decisions needs exactly one request JSON field")
	}
	seen := map[string]bool{}
	duplicate := false
	gjson.Parse(form.Value["request"][0]).ForEach(func(key, value gjson.Result) bool {
		if seen[key.Str] {
			duplicate = true
			return false
		}
		seen[key.Str] = true
		return true
	})
	if duplicate {
		return nil, errors.New("duplicate decisions request field")
	}
	var body map[string]json.RawMessage
	if err = Unmarshal([]byte(form.Value["request"][0]), &body); err != nil {
		return nil, err
	}
	var assets map[string]map[string]json.RawMessage
	if raw, ok := body["assets"]; ok {
		if err = Unmarshal(raw, &assets); err != nil {
			return nil, err
		}
	}
	used := map[string]bool{}
	for _, asset := range assets {
		var part string
		if raw, ok := asset["part"]; ok {
			if err = Unmarshal(raw, &part); err != nil {
				return nil, err
			}
		}
		if part == "" {
			continue
		}
		files := form.File[part]
		if len(files) != 1 || files[0].Size > 20<<20 {
			return nil, errors.New("image part missing, duplicated or larger than 20 MiB")
		}
		file, err := files[0].Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(io.LimitReader(file, (20<<20)+1))
		file.Close()
		if err != nil {
			return nil, err
		}
		if len(data) > 20<<20 {
			return nil, errors.New("image too large")
		}
		format := ""
		if len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n" {
			format = "png"
		} else if len(data) >= 3 && data[0] == 255 && data[1] == 216 && data[2] == 255 {
			format = "jpg"
		}
		if format == "" {
			return nil, errors.New("only PNG and JPEG image parts are supported")
		}
		asset["format"], _ = Marshal(format)
		asset["data_base64"], _ = Marshal(base64.StdEncoding.EncodeToString(data))
		delete(asset, "part")
		used[part] = true
	}
	for name := range form.File {
		if !used[name] {
			return nil, errors.New("unreferenced image part")
		}
	}
	if assets != nil {
		body["assets"], err = Marshal(assets)
		if err != nil {
			return nil, err
		}
	}
	data, err := Marshal(body)
	if err != nil {
		return nil, err
	}
	c.Set("decisions_json_body", data)
	return data, nil
}
