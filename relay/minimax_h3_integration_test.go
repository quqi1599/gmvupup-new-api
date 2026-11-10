package relay

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestH3ImageHTTPFailuresDisableReplay(t *testing.T) {
	service.InitHttpClient()
	settings := model_setting.GetGlobalSettings()
	previous := settings.PassThroughRequestEnabled
	t.Cleanup(func() { settings.PassThroughRequestEnabled = previous })
	for _, passthrough := range []bool{false, true} {
		for _, edit := range []bool{false, true} {
			for _, status := range []int{400, 429, 500, 0} {
				t.Run(fmt.Sprintf("pass=%t/edit=%t/status=%d", passthrough, edit, status), func(t *testing.T) {
					settings.PassThroughRequestEnabled = passthrough
					var calls atomic.Int32
					path := "/v1/images/generations"
					mode := relayconstant.RelayModeImagesGenerations
					if edit {
						path = "/v1/images/edits"
						mode = relayconstant.RelayModeImagesEdits
					}
					paths := make(chan string, 1)
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						paths <- r.URL.Path
						io.Copy(io.Discard, r.Body)
						if status == 0 {
							conn, _, err := w.(http.Hijacker).Hijack()
							if err == nil {
								conn.Close()
							}
							return
						}
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(status)
						io.WriteString(w, `{"error":{"message":"synthetic failure","type":"server_error"}}`)
					}))
					defer server.Close()
					modelName := "MiniMax-H3-Image"
					if !passthrough {
						modelName = "public-image-alias"
					}
					c, _ := gin.CreateTestContext(httptest.NewRecorder())
					c.Request = httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"model":"`+modelName+`","prompt":"synthetic","images":[{"image_url":"https://example.com/input.png"}]}`))
					c.Request.Header.Set("Content-Type", "application/json")
					t.Cleanup(func() { common.CleanupBodyStorage(c) })
					request, err := helper.GetAndValidOpenAIImageRequest(c, mode)
					require.NoError(t, err)
					common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeMiniMax)
					common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, server.URL)
					common.SetContextKey(c, constant.ContextKeyOriginalModel, modelName)
					if !passthrough {
						c.Set("model_mapping", `{"public-image-alias":"MiniMax-H3-Image-Highspeed"}`)
					}
					info := &relaycommon.RelayInfo{OriginModelName: modelName, Request: request, RelayMode: mode, RelayFormat: types.RelayFormatOpenAI, RequestURLPath: path}
					apiErr := ImageHelper(c, info)
					require.NotNil(t, apiErr)
					require.True(t, types.IsSkipRetryError(apiErr), "H3 HTTP errors and unknown transport outcomes must not be replayed")
					require.EqualValues(t, 1, calls.Load())
					require.Equal(t, path, <-paths)
				})
			}
		}
	}
}
