package typesafe

import (
	"github.com/QuantumNous/new-api/common"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/stretchr/testify/require"
)

func TestDecisionsUpstreamPath(t *testing.T) {
	for _, tc := range []struct{ path, want string }{
		{"", "https://api.typesafe.ai/v1/systemone"},
		{"/alpha/decisions", "https://api.typesafe.ai/alpha/decisions"},
		{"v1/systemone", "https://api.typesafe.ai/v1/systemone"},
		{"//other.example/api", ""},
		{"/../other", ""},
		{"/%2e%2e/other", ""},
		{"/v1/systemone?secret=value", ""},
		{"/v1/systemone#fragment", ""},
	} {
		info := &relaycommon.RelayInfo{RelayMode: relayconstant.RelayModeDecisions, ChannelMeta: &relaycommon.ChannelMeta{ChannelBaseUrl: "https://api.typesafe.ai/", ChannelSetting: dto.ChannelSettings{DecisionsUpstreamPath: tc.path}}}
		got, err := (&Adaptor{}).GetRequestURL(info)
		if tc.want == "" {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		}
	}
}

func TestDecisionsBoxBounds(t *testing.T) {
	for _, box := range []string{`{"top":0,"bottom":1,"left":0,"right":1}`, `{"top":0.8,"bottom":0.2,"left":0,"right":1}`, `{"top":0,"bottom":1,"left":0}`, `{"top":0,"bottom":1,"left":0,"right":2}`} {
		a := &Adaptor{request: &dto.DecisionsRequest{Questions: map[string]dto.DecisionsQuestion{"q": {Type: "box"}}}}
		var response dto.DecisionsResponse
		require.NoError(t, common.Unmarshal([]byte(`{"model":"jev-1.14.0","answers":{"q":{"type":"box","confidence":0,"box":`+box+`}},"usage":{"input_tokens":80274,"output_tokens":0}}`), &response))
		require.Equal(t, box == `{"top":0,"bottom":1,"left":0,"right":1}`, a.validResponse(&response))
	}
}
