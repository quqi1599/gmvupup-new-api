package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/andybalholm/brotli"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestDecompressRequestMiddlewareClearsCompressedContentLength(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalBody := bytes.Repeat([]byte(`{"input":"compressible"}`), 256)
	for _, tc := range []struct {
		encoding string
		writer   func(io.Writer) io.WriteCloser
	}{
		{"gzip", func(w io.Writer) io.WriteCloser { return gzip.NewWriter(w) }},
		{"br", func(w io.Writer) io.WriteCloser { return brotli.NewWriter(w) }},
	} {
		t.Run(tc.encoding, func(t *testing.T) {
			var wire bytes.Buffer
			writer := tc.writer(&wire)
			_, err := writer.Write(originalBody)
			require.NoError(t, err)
			require.NoError(t, writer.Close())

			router := gin.New()
			router.Use(DecompressRequestMiddleware())
			router.POST("/", func(c *gin.Context) {
				require.Equal(t, int64(-1), c.Request.ContentLength)
				require.Empty(t, c.GetHeader("Content-Encoding"))

				storage, err := common.CreateBodyStorageFromReader(c.Request.Body, c.Request.ContentLength, 1<<20)
				require.NoError(t, err)
				defer storage.Close()
				decoded, err := storage.Bytes()
				require.NoError(t, err)
				require.Equal(t, originalBody, decoded)
				c.Status(http.StatusNoContent)
			})

			request := httptest.NewRequest(http.MethodPost, "/", &wire)
			request.Header.Set("Content-Encoding", tc.encoding)
			require.Greater(t, request.ContentLength, int64(0))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			require.Equal(t, http.StatusNoContent, response.Code)
		})
	}
}
