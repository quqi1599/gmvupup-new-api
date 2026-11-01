package helper

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const responsePrefixLimit = 256

// Owned by the scanner goroutine; inspected only after cleanup has joined it.
// The prefix exists only while reading and is never put in a log or context.
type streamResponseObservation struct {
	diagnostic types.UpstreamResponseDiagnostic
	prefix     [responsePrefixLimit]byte
	prefixLen  int
}

func newStreamResponseObservation(resp *http.Response, streamExpected bool, requestStream string) *streamResponseObservation {
	switch requestStream {
	case "true", "false", "null", "omitted", "non_boolean", "not_observed":
	default:
		requestStream = "not_observed"
	}
	return &streamResponseObservation{diagnostic: types.UpstreamResponseDiagnostic{
		StatusCode: resp.StatusCode, ContentType: responseContentTypeClass(resp.Header.Get("Content-Type")),
		ContentLength: resp.ContentLength, StreamExpected: streamExpected, RequestStream: requestStream,
	}}
}

// RecordResponsesRequestStream observes only the final encoded field's type.
// It does not read or rewrite a passthrough reader, or retain any field value
// other than a boolean. Call after parameter overrides, before body spooling.
func RecordResponsesRequestStream(c *gin.Context, body []byte) {
	stream := gjson.GetBytes(body, "stream")
	kind := "non_boolean"
	switch {
	case !stream.Exists():
		kind = "omitted"
	case stream.Type == gjson.True:
		kind = "true"
	case stream.Type == gjson.False:
		kind = "false"
	case stream.Type == gjson.Null:
		kind = "null"
	}
	common.SetContextKey(c, constant.ContextKeyUpstreamRequestStream, kind)
}

// Write observes the bytes TeeReader already read; it never reads ahead,
// rejects a byte, changes EOF, or retains an unbounded response body.
func (o *streamResponseObservation) Write(p []byte) (int, error) {
	o.diagnostic.ReadBytes += int64(len(p))
	o.prefixLen += copy(o.prefix[o.prefixLen:], p)
	return len(p), nil
}

func (o *streamResponseObservation) observeLine(line string) {
	line = strings.TrimSpace(line)
	switch {
	case line == "":
		o.diagnostic.BlankLines++
	case strings.HasPrefix(line, "data:"):
		o.diagnostic.DataLines++
	case strings.HasPrefix(line, ":"):
		o.diagnostic.CommentLines++
	case strings.HasPrefix(line, "event:"):
		o.diagnostic.EventLines++
	default:
		o.diagnostic.OtherLines++
	}
}

func (o *streamResponseObservation) snapshot() types.UpstreamResponseDiagnostic {
	prefix := bytes.TrimSpace(o.prefix[:o.prefixLen])
	o.diagnostic.UTF8BOM = bytes.HasPrefix(prefix, []byte{0xef, 0xbb, 0xbf})
	if o.diagnostic.UTF8BOM {
		prefix = bytes.TrimSpace(prefix[3:])
	}
	switch {
	case o.diagnostic.ReadBytes == 0:
		o.diagnostic.BodyPrefix = "empty"
	case len(prefix) == 0:
		o.diagnostic.BodyPrefix = "whitespace"
	case bytes.HasPrefix(prefix, []byte("[DONE]")):
		o.diagnostic.BodyPrefix = "done_marker"
	case prefix[0] == '{' || prefix[0] == '[':
		o.diagnostic.BodyPrefix = "json_like"
	case prefix[0] == '<':
		o.diagnostic.BodyPrefix = "markup_like"
		lower := bytes.ToLower(prefix)
		if bytes.HasPrefix(lower, []byte("<!doctype html")) || bytes.HasPrefix(lower, []byte("<html")) {
			o.diagnostic.BodyPrefix = "html_like"
		}
	case bytes.HasPrefix(prefix, []byte("data:")) || bytes.HasPrefix(prefix, []byte("event:")) || prefix[0] == ':' || bytes.HasPrefix(prefix, []byte("id:")) || bytes.HasPrefix(prefix, []byte("retry:")):
		o.diagnostic.BodyPrefix = "sse_like"
	default:
		o.diagnostic.BodyPrefix = "other"
	}
	clear(o.prefix[:])
	o.prefixLen = 0
	return o.diagnostic
}

func responseContentTypeClass(value string) string {
	mediaType, _, _ := strings.Cut(value, ";")
	mediaType = strings.TrimSpace(mediaType)
	if mediaType == "" {
		return "missing"
	}
	// Do not retain arbitrary media types or parameters supplied by upstream.
	if len(mediaType) > 128 {
		return "other"
	}
	mediaType = strings.ToLower(mediaType)
	switch mediaType {
	case "text/event-stream", "application/json", "text/html", "text/plain":
		return mediaType
	default:
		if strings.HasPrefix(mediaType, "application/") && strings.HasSuffix(mediaType, "+json") {
			return "application/json"
		}
		return "other"
	}
}
