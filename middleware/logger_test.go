package middleware

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAccessAndPanicLogsFollowLogRotation(t *testing.T) {
	previousMode, previousWriter, previousErrorWriter := gin.Mode(), gin.DefaultWriter, gin.DefaultErrorWriter
	previousLogDir := *common.LogDir
	gin.SetMode(gin.TestMode)
	*common.LogDir = t.TempDir()
	t.Cleanup(func() {
		gin.SetMode(previousMode)
		common.LogWriterMu.Lock()
		gin.DefaultWriter, gin.DefaultErrorWriter = previousWriter, previousErrorWriter
		common.LogWriterMu.Unlock()
		*common.LogDir = previousLogDir
	})

	logger.SetupLogger()
	router := gin.New()
	router.Use(PanicRecover())
	SetUpLogger(router)
	router.GET("/ok/:step", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.GET("/panic", func(c *gin.Context) { panic("rotation-panic-marker") })

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ok/before-rotation", nil))
	content, err := os.ReadFile(logger.GetCurrentLogPath())
	require.NoError(t, err)
	require.Contains(t, string(content), "/ok/before-rotation")

	// A different directory forces a new file even within the same second.
	*common.LogDir = t.TempDir()
	logger.SetupLogger()
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ok/after-rotation?api_key=ROTATION_SECRET&alt=sse", nil))
	panicResponse := httptest.NewRecorder()
	router.ServeHTTP(panicResponse, httptest.NewRequest(http.MethodGet, "/panic", nil))
	require.Equal(t, http.StatusInternalServerError, panicResponse.Code)
	require.Contains(t, panicResponse.Body.String(), `"type":"new_api_panic"`)

	content, err = os.ReadFile(logger.GetCurrentLogPath())
	require.NoError(t, err)
	require.Contains(t, string(content), "/ok/after-rotation")
	require.Contains(t, string(content), "alt=sse")
	require.NotContains(t, string(content), "ROTATION_SECRET")
	require.NotContains(t, string(content), "/ok/before-rotation")
	require.Contains(t, string(content), "[Recovery]")
	require.Contains(t, string(content), "rotation-panic-marker\n")
	require.Contains(t, string(content), "middleware/logger_test.go")

	// Exercise the swap/close boundary while requests are writing. Every marker
	// must survive exactly once across the files created by concurrent rotations.
	var requests sync.WaitGroup
	start := make(chan struct{})
	for worker := 0; worker < 4; worker++ {
		requests.Add(1)
		go func() {
			defer requests.Done()
			<-start
			for request := 0; request < 20; request++ {
				target := fmt.Sprintf("/ok/concurrent-%d-%d", worker, request)
				router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, target, nil))
			}
		}()
	}
	close(start)
	for rotation := 0; rotation < 10; rotation++ {
		logger.SetupLogger()
	}
	requests.Wait()
	paths, err := filepath.Glob(filepath.Join(*common.LogDir, "*.log"))
	require.NoError(t, err)
	var allLogs strings.Builder
	for _, path := range paths {
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		allLogs.Write(data)
	}
	for worker := 0; worker < 4; worker++ {
		for request := 0; request < 20; request++ {
			marker := fmt.Sprintf("/ok/concurrent-%d-%d\n", worker, request)
			require.Equal(t, 1, strings.Count(allLogs.String(), marker), marker)
		}
	}
}

func TestBatch1AccessLogCredentials(t *testing.T) {
	writer := gin.DefaultWriter
	t.Cleanup(func() { gin.DefaultWriter = writer })
	for _, target := range []string{
		"/v1/models?%6bey=SECRET&key=SECOND&API-Key=THIRD&session_token=FOURTH&alt=sse",
		"/v1/models?key=SECRET%ZZ&token=SECOND;other=true&alt=sse",
		"/v1/models%23test%25?key=SECRET&alt=sse",
		"/%61pi/oauth/test?code=SECRET&state=SECOND",
		"/oauth/test?code=SECRET&state=SECOND",
		"/v1/image-results/SECRET?download=1",
		"/v1/%69mage-results/SECRET?download=1",
	} {
		t.Run(target, func(t *testing.T) {
			var output bytes.Buffer
			gin.DefaultWriter = &output
			r := gin.New()
			SetUpLogger(r)
			req := httptest.NewRequest("GET", target, nil)
			original := *req.URL
			r.NoRoute(func(c *gin.Context) {
				require.Equal(t, original, *c.Request.URL)
				require.Equal(t, target, c.Request.RequestURI)
				c.Status(401)
			})
			response := httptest.NewRecorder()
			r.ServeHTTP(response, req)
			require.Equal(t, 401, response.Code)
			for _, secret := range []string{"SECRET", "SECOND", "THIRD", "FOURTH"} {
				require.NotContains(t, output.String(), secret)
			}
			require.Equal(t, original, *req.URL)
			if strings.Contains(target, "alt=sse") && !strings.Contains(target, "%ZZ") {
				require.Contains(t, output.String(), "alt=sse")
			}
		})
	}
}

func TestBatch1AccessLogPathControls(t *testing.T) {
	for _, target := range []string{"/v1/models", "/v1/models?alt=sse&api-version=2024-02-01", "//v1%2Fmodels?alt=sse", "/v1/models%23test%25?alt=%23sse"} {
		require.Equal(t, target, accessLogPath(target))
	}
	for _, key := range []string{"key", "API_KEY", "api-key", "apikey", "x-api-key", "x-goog-api-key", "authorization", "auth", "password", "passwd", "sig", "AWSAccessKeyId", "x-amz-credential", "download_token", "client_secret", "x-amz-signature"} {
		require.NotContains(t, accessLogPath("/v1/models?"+key+"=SECRET"), "SECRET", key)
	}
	require.Equal(t, "/api/oauth/test", accessLogPath("/api/oauth/test?code=SECRET&state=SECRET"))
	require.Equal(t, "/v1/models", accessLogPath("/v1/models?key=SECRET%ZZ"))
	require.Equal(t, "[invalid request URI]", accessLogPath("/%ZZ?key=SECRET"))
}

func TestBatch1AccessLogRetainsClientPathAfterAdapterRewrite(t *testing.T) {
	writer := gin.DefaultWriter
	t.Cleanup(func() { gin.DefaultWriter = writer })
	var output bytes.Buffer
	gin.DefaultWriter = &output
	r := gin.New()
	SetUpLogger(r)
	r.POST("/client/video", func(c *gin.Context) {
		c.Request.URL.Path = "/v1/video/generations"
		c.Status(200)
	})
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/client/video?key=SECRET&alt=sse", nil))
	require.Contains(t, output.String(), "/client/video?")
	require.NotContains(t, output.String(), "/v1/video/generations")
	require.NotContains(t, output.String(), "SECRET")
}
