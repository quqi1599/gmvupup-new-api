package model

import (
	"crypto/sha256"
	"errors"
	"slices"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/types"
)

// This binding exists only for the lifetime of one authenticated client socket.
type ResponsesSocketBinding struct {
	ChannelID, KeyIndex, TokenID, UserID int
	Model, Group                         string
	KeyHash                              [32]byte
}

func (b *ResponsesSocketBinding) ChannelKey(ch *Channel) (string, int, *types.NewAPIError) {
	if b == nil || b.ChannelID == 0 {
		return ch.GetNextEnabledKey()
	}
	lock := GetChannelPollingLock(ch.Id)
	lock.Lock()
	defer lock.Unlock()
	if ch.Id == b.ChannelID {
		key := ch.Key
		active := !ch.ChannelInfo.IsMultiKey
		if ch.ChannelInfo.IsMultiKey {
			keys := ch.GetKeys()
			if b.KeyIndex >= 0 && b.KeyIndex < len(keys) {
				key = keys[b.KeyIndex]
				status, found := ch.ChannelInfo.MultiKeyStatusList[b.KeyIndex]
				active = !found || status == common.ChannelStatusEnabled
			}
		}
		if active && sha256.Sum256([]byte(key)) == b.KeyHash {
			return key, b.KeyIndex, nil
		}
	}
	return "", 0, types.NewErrorWithStatusCode(errors.New("Responses WebSocket credential is no longer available"), types.ErrorCodeGetChannelFailed, 409, types.ErrOptionWithSkipRetry())
}

// The handshake has no model body. Advertise WS only for groups with at least
// one explicitly enabled channel; per-turn selection checks the actual model.
func HasResponsesSocketChannel(groups []string, channelID int, models []string) (bool, error) {
	if len(groups) == 0 {
		return false, nil
	}
	abilities := DB.Model(&Ability{}).Select("channel_id").Where(commonGroupCol+" IN ? AND enabled = ?", groups, true)
	if models != nil {
		if len(models) == 0 {
			return false, nil
		}
		abilities = abilities.Where("model IN ?", models)
	}
	query := DB.Model(&Channel{}).Select("id", "type", "setting").Where("status = ? AND id IN (?)", common.ChannelStatusEnabled, abilities)
	if channelID > 0 {
		query = query.Where("id = ?", channelID)
	}
	var channels []Channel
	if err := query.Find(&channels).Error; err != nil {
		return false, err
	}
	return slices.ContainsFunc(channels, func(ch Channel) bool { return ch.SupportsEndpointType(constant.EndpointTypeResponsesWebSocket) }), nil
}
