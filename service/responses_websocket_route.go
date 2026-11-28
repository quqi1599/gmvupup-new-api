package service

import (
	"errors"
	"slices"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func ResponsesSocketBinding(c *gin.Context) *model.ResponsesSocketBinding {
	if c == nil {
		return nil
	}
	v, _ := c.Get(string(constant.ContextKeyResponsesWebSocket))
	b, _ := v.(*model.ResponsesSocketBinding)
	return b
}

func RequiredRelayEndpoint(c *gin.Context) constant.EndpointType {
	if ResponsesSocketBinding(c) != nil {
		return constant.EndpointTypeResponsesWebSocket
	}
	return types.PathToRequiredEndpointType(c.Request.URL.Path)
}

func ResponsesSocketGroups(c *gin.Context) []string {
	group := common.GetContextKeyString(c, constant.ContextKeyUsingGroup)
	if group == "auto" {
		return GetUserAutoGroup(common.GetContextKeyString(c, constant.ContextKeyUserGroup))
	}
	return []string{group}
}

func BoundResponsesChannel(c *gin.Context, b *model.ResponsesSocketBinding, modelName string) (*model.Channel, string, error) {
	if b.TokenID != c.GetInt("token_id") || b.UserID != c.GetInt("id") || b.Model != modelName || !slices.Contains(ResponsesSocketGroups(c), b.Group) || isTokenChannelExcluded(c, b.ChannelID) || !model.IsChannelEnabledForGroupModel(b.Group, modelName, b.ChannelID) {
		return nil, "", errors.New("Responses WebSocket route is no longer eligible for this request")
	}
	ch, err := model.CacheGetChannel(b.ChannelID)
	if err != nil {
		return nil, "", err
	}
	if ch == nil || ch.Status != common.ChannelStatusEnabled || !ch.SupportsEndpointType(constant.EndpointTypeResponsesWebSocket) {
		return nil, "", errors.New("Responses WebSocket channel is no longer enabled")
	}
	if common.GetContextKeyString(c, constant.ContextKeyUsingGroup) == "auto" {
		common.SetContextKey(c, constant.ContextKeyAutoGroup, b.Group)
	}
	return ch, b.Group, nil
}
