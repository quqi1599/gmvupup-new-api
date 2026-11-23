package claude

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestChatToClaudeExplicitCacheBreakpoints(t *testing.T) {
	var req dto.GeneralOpenAIRequest
	require.NoError(t, common.UnmarshalJsonStr(`{"model":"fixture","messages":[{"role":"system","content":[{"type":"text","text":"s","cache_control":{"type":"ephemeral"}}]},{"role":"user","content":[{"type":"text","text":"u","cache_control":{"type":"ephemeral","ttl":"1h"}},{"type":"text","text":"uncached"}]}],"tools":[{"type":"function","function":{"name":"f","parameters":{"type":"object"}},"cache_control":{"type":"ephemeral"}}]}`, &req))
	got, err := RequestOpenAI2ClaudeMessage(nil, req)
	require.NoError(t, err)
	wire, err := common.Marshal(got)
	require.NoError(t, err)
	for _, p := range []string{"system.0", "messages.0.content.0", "tools.0"} {
		require.Equal(t, "ephemeral", gjson.GetBytes(wire, p+".cache_control.type").String(), p)
	}
	require.Equal(t, "1h", gjson.GetBytes(wire, "messages.0.content.0.cache_control.ttl").String())
	require.False(t, gjson.GetBytes(wire, "messages.0.content.1.cache_control").Exists())
	require.False(t, gjson.GetBytes(wire, "cache_control").Exists())
}
