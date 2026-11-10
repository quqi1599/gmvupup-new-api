package minimax

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestH3ImageRoutesAndJSONFields(t *testing.T) {
	for _, model := range []string{"MiniMax-H3-Image", "MiniMax-H3-Image-Highspeed"} {
		for _, mode := range []int{relayconstant.RelayModeImagesGenerations, relayconstant.RelayModeImagesEdits} {
			t.Run(fmt.Sprintf("%s/%d", model, mode), func(t *testing.T) {
				info := &relaycommon.RelayInfo{OriginModelName: "public-alias", RelayMode: mode,
					ChannelMeta: &relaycommon.ChannelMeta{ChannelBaseUrl: "https://api.minimax.cn/v1/", UpstreamModelName: model}}
				path := "/v1/images/generations"
				if mode == relayconstant.RelayModeImagesEdits {
					path = "/v1/images/edits"
				}
				u, err := GetRequestURL(info)
				require.NoError(t, err)
				require.Equal(t, "https://api.minimax.cn"+path, u)
				body := `{"model":"` + model + `","prompt":"synthetic image","n":2,"size":"auto","area":"2K","aspect_ratio":"16:9","disable_prompt_enhanced":false,"aigc_watermark":false,"stream":false,"output_format":"webp","output_compression":0,"response_format":"b64_json","images":[{"image_url":"mm_file://test"}],"unknown_vendor_field":123}`
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
				c.Request.Header.Set("Content-Type", "application/json")
				t.Cleanup(func() { common.CleanupBodyStorage(c) })
				req, err := helper.GetAndValidOpenAIImageRequest(c, mode)
				require.NoError(t, err)
				req, err = common.DeepCopy(req)
				require.NoError(t, err)
				converted, err := (&Adaptor{}).ConvertImageRequest(c, info, *req)
				require.NoError(t, err)
				wire, err := common.Marshal(converted)
				require.NoError(t, err)
				for _, field := range []string{"model", "prompt", "n", "size", "area", "aspect_ratio", "disable_prompt_enhanced", "aigc_watermark", "stream", "output_format", "output_compression", "response_format", "images"} {
					require.Equal(t, gjson.Get(body, field).Raw, gjson.GetBytes(wire, field).Raw, field)
				}
				require.False(t, gjson.GetBytes(wire, "unknown_vendor_field").Exists())
			})
		}
	}
}

func TestH3ImageRejectsInvalidCount(t *testing.T) {
	for _, n := range []string{"0", "11", "-1", "1.5"} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", strings.NewReader(`{"model":"MiniMax-H3-Image","prompt":"test","n":`+n+`}`))
		c.Request.Header.Set("Content-Type", "application/json")
		_, err := helper.GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesGenerations)
		require.Error(t, err, n)
		common.CleanupBodyStorage(c)
	}
}

func TestH3MultipartEditHTTP(t *testing.T) {
	service.InitHttpClient()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	for field, value := range map[string]string{"model": "alias", "prompt": "combine", "stream": "false", "n": "2", "area": "2K", "aspect_ratio": "3:2", "output_compression": "0", "aigc_watermark": "false"} {
		require.NoError(t, w.WriteField(field, value))
	}
	for _, data := range []string{"first image", "second image"} {
		part, err := w.CreateFormFile("image[]", "input.png")
		require.NoError(t, err)
		_, err = io.WriteString(part, data)
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	type received struct {
		path, auth, model, area, compression, watermark string
		images                                          []string
		err                                             error
	}
	got := make(chan received, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := received{path: r.URL.Path, auth: r.Header.Get("Authorization")}
		result.err = r.ParseMultipartForm(1 << 20)
		if result.err == nil {
			defer r.MultipartForm.RemoveAll()
			result.model, result.area, result.compression, result.watermark = r.FormValue("model"), r.FormValue("area"), r.FormValue("output_compression"), r.FormValue("aigc_watermark")
			for _, part := range r.MultipartForm.File["image[]"] {
				f, err := part.Open()
				if err != nil {
					result.err = err
					break
				}
				data, err := io.ReadAll(f)
				f.Close()
				result.err = err
				result.images = append(result.images, string(data))
			}
		}
		got <- result
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{}`)
	}))
	defer server.Close()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
	c.Request.Header.Set("Content-Type", w.FormDataContentType())
	t.Cleanup(func() { common.CleanupBodyStorage(c) })
	req, err := helper.GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)
	require.NoError(t, err)
	req.Model = "MiniMax-H3-Image-Highspeed"
	info := &relaycommon.RelayInfo{RelayMode: relayconstant.RelayModeImagesEdits, OriginModelName: "alias", ChannelMeta: &relaycommon.ChannelMeta{ChannelBaseUrl: server.URL, UpstreamModelName: req.Model, ApiKey: "synthetic-key"}}
	a := &Adaptor{}
	converted, err := a.ConvertImageRequest(c, info, *req)
	require.NoError(t, err)
	resp, err := a.DoRequest(c, info, converted.(io.Reader))
	require.NoError(t, err)
	resp.(*http.Response).Body.Close()
	result := <-got
	require.NoError(t, result.err)
	require.Equal(t, "/v1/images/edits", result.path)
	require.Equal(t, "Bearer synthetic-key", result.auth)
	require.Equal(t, req.Model, result.model)
	require.Equal(t, "2K", result.area)
	require.Equal(t, "0", result.compression)
	require.Equal(t, "false", result.watermark)
	require.Equal(t, []string{"first image", "second image"}, result.images)
}
