package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func providerClientFallbackFormat(format types.RelayFormat) bool {
	return format == types.RelayFormatOpenAI || format == types.RelayFormatClaude || format == types.RelayFormatOpenAIResponses
}

// Another configured provider may accept this client, but must not receive
// state owned by a different provider. The original request is never rewritten.
func providerClientFallbackRequestSafe(c *gin.Context, info *relaycommon.RelayInfo, format types.RelayFormat) bool {
	if c == nil || c.Request == nil || info == nil || !providerClientFallbackFormat(format) || c.Request.Method != http.MethodPost || c.Request.Context().Err() != nil {
		return false
	}
	if info.ChannelMeta == nil || len(info.ParamOverride) != 0 || len(common.GetContextKeyStringMap(c, constant.ContextKeyChannelParamOverride)) != 0 {
		return false
	}
	if web := info.ManagedWeb; web != nil && (web.ModelRounds > 0 || web.SearchRequests > 0 || web.FetchRequests > 0 || web.Outcome != "") {
		return false
	}
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return false
	}
	body, err := storage.Bytes()
	if err != nil || !gjson.ValidBytes(body) {
		return false
	}
	root := gjson.ParseBytes(body)
	if !root.IsObject() || root.Get("background").Bool() {
		return false
	}
	return !providerBoundState(root, 0)
}

func providerBoundState(value gjson.Result, depth int) bool {
	if depth > 64 {
		return true
	}
	if !value.IsObject() && !value.IsArray() {
		return false
	}
	blocked := false
	value.ForEach(func(key, item gjson.Result) bool {
		if value.IsObject() {
			if key.String() == "metadata" || depth == 0 && (key.String() == "text" || key.String() == "response_format") {
				return true
			}
			if key.String() == "tools" && depth == 0 && item.IsArray() {
				for _, tool := range item.Array() {
					if kind := tool.Get("type").String(); kind != "function" && kind != "custom" && providerBoundState(tool, depth+1) {
						blocked = true
						break
					}
				}
				return !blocked
			}
			switch key.String() {
			case "previous_response_id", "conversation":
				blocked = depth == 0 && item.Type != gjson.Null && item.Raw != `""`
			case "container":
				blocked = item.Type != gjson.Null && item.Raw != `""`
			case "prompt":
				blocked = depth == 0 && item.Get("id").String() != ""
			case "background":
				blocked = depth == 0 && item.Bool()
			case "encrypted_content", "signature", "file_id":
				blocked = item.Type == gjson.String && item.String() != ""
			case "vector_store_ids":
				blocked = item.IsArray() && len(item.Array()) > 0
			case "type":
				switch item.String() {
				case "item_reference", "compaction", "redacted_thinking", "web_search_call", "file_search_call", "code_interpreter_call", "image_generation_call":
					blocked = true
				}
			}
		}
		blocked = blocked || providerBoundState(item, depth+1)
		return !blocked
	})
	return blocked
}
