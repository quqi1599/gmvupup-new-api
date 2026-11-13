package router

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestJev114ImagesAndSDK(t *testing.T) {
	for _, multipartBody := range []bool{false, true} {
		t.Run(map[bool]string{false: "base64", true: "SDK multipart"}[multipartBody], func(t *testing.T) {
			_, token := setupJevRelayTest(t)
			var picture bytes.Buffer
			require.NoError(t, png.Encode(&picture, image.NewRGBA(image.Rect(0, 0, 32, 32))))
			asset := dto.DecisionsAsset{Type: "image", Format: "png", DataBase64: base64.StdEncoding.EncodeToString(picture.Bytes())}
			if multipartBody {
				asset = dto.DecisionsAsset{Type: "image", Part: "photo"}
			}
			body := dto.DecisionsRequest{Model: "jev-latest", State: []byte(`{"photo":"image-ref","n":0,"count":false}`), Assets: map[string]dto.DecisionsAsset{"image-ref": asset}, Questions: map[string]dto.DecisionsQuestion{
				"box":   {Type: "box", Instructions: []byte(`"Find the square"`)},
				"score": {Type: "score", Instructions: []byte(`null`), Criteria: []byte(`["one"]`)},
				"noul":  {Type: "noul", Instructions: []byte(`"Is there an image?"`), Criteria: []byte(`{"true":null,"extra":42}`)},
			}}
			raw, err := common.Marshal(body)
			require.NoError(t, err)
			calls := 0
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.Equal(t, "true", r.Header.Get("TypeSafe-Danger-Zone-Images"))
				require.Equal(t, "true", r.Header.Get("TypeSafe-Danger-Zone-Long-Context"))
				require.Equal(t, "Bearer upstream-test-key", r.Header.Get("Authorization"))
				var received dto.DecisionsRequest
				require.NoError(t, common.DecodeJson(r.Body, &received))
				require.Equal(t, base64.StdEncoding.EncodeToString(picture.Bytes()), received.Assets["image-ref"].DataBase64)
				require.JSONEq(t, string(body.State), string(received.State))
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, `{"model":"jev-1.14.0","answers":{"box":{"type":"box","box":{"top":0,"bottom":1,"left":0,"right":1},"confidence":0.9},"score":{"type":"score","score":0,"legend":{"0":"one"},"probabilities":{"0":1},"confidence":1},"noul":{"type":"noul","noul":0}},"usage":{"input_tokens":80274,"output_tokens":21}}`)
			}))
			defer upstream.Close()
			createJevChannel(t, upstream.URL, "upstream-test-key", false, "")
			contentType := "application/json"
			if multipartBody {
				var buf bytes.Buffer
				mw := multipart.NewWriter(&buf)
				require.NoError(t, mw.WriteField("request", string(raw)))
				f, err := mw.CreateFormFile("photo", "image.png")
				require.NoError(t, err)
				_, err = f.Write(picture.Bytes())
				require.NoError(t, err)
				require.NoError(t, mw.Close())
				raw = buf.Bytes()
				contentType = mw.FormDataContentType()
			}
			req := httptest.NewRequest("POST", "/v1/systemone", bytes.NewReader(raw))
			req.Header.Set("Content-Type", contentType)
			req.Header.Set("Authorization", "Bearer sk-"+token.Key)
			req.Header.Set("TypeSafe-Danger-Zone-Long-Context", "true")
			engine := gin.New()
			SetRelayRouter(engine)
			res := httptest.NewRecorder()
			engine.ServeHTTP(res, req)
			require.Equal(t, 200, res.Code, res.Body.String())
			require.Equal(t, 1, calls)
			var logs []model.Log
			require.NoError(t, model.DB.Where("type = ?", 2).Find(&logs).Error)
			require.Len(t, logs, 1)
			require.Equal(t, 80274, logs[0].PromptTokens)
			require.Equal(t, 1686, logs[0].Quota)
		})
	}
}

func TestJev114VendorError(t *testing.T) {
	_, token := setupJevRelayTest(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		io.WriteString(w, `{"detail":{"error_type":"max_tokens_exceeded","limit":32768}}`)
	}))
	defer upstream.Close()
	createJevChannel(t, upstream.URL, "key", false, "")
	req := httptest.NewRequest("POST", "/v1/decisions", strings.NewReader(jevRequest))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-"+token.Key)
	engine := gin.New()
	SetRelayRouter(engine)
	res := httptest.NewRecorder()
	engine.ServeHTTP(res, req)
	require.Equal(t, 400, res.Code)
	require.JSONEq(t, `{"detail":{"error_type":"max_tokens_exceeded","limit":32768}}`, res.Body.String())
	require.Eventually(t, func() bool {
		var current model.Token
		return model.DB.First(&current, token.Id).Error == nil && current.RemainQuota == 50000
	}, time.Second, 10*time.Millisecond)
}
