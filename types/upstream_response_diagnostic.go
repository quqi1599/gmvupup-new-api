package types

// UpstreamResponseDiagnostic contains only bounded, body-free observations.
// ContentType and BodyPrefix are classifications, never upstream strings.
// ReadBytes counts bytes read from the response body, not network wire bytes.
type UpstreamResponseDiagnostic struct {
	StatusCode     int    `json:"status_code"`
	ContentType    string `json:"content_type"`
	ContentLength  int64  `json:"content_length"`
	ReadBytes      int64  `json:"read_bytes"`
	StreamExpected bool   `json:"stream_expected"`
	RequestStream  string `json:"request_stream"`
	BodyPrefix     string `json:"body_prefix"`
	UTF8BOM        bool   `json:"utf8_bom"`
	DataLines      int64  `json:"data_lines"`
	CommentLines   int64  `json:"comment_lines"`
	EventLines     int64  `json:"event_lines"`
	OtherLines     int64  `json:"other_lines"`
	BlankLines     int64  `json:"blank_lines"`
	EndReason      string `json:"end_reason"`
	SoftErrors     int    `json:"soft_errors"`
}
