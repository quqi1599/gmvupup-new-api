package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestChannelAffinityFallbackPriorityAndBudget(t *testing.T) {
	for _, cached := range []bool{false, true} {
		for _, exhaust := range []bool{false, true} {
			for _, tc := range []struct {
				name        string
				affinity    bool
				miss        bool
				skipRetry   bool
				retries     int
				autoGroups  string
				crossGroup  bool
				excludeHigh bool
				want        []int
			}{
				{name: "ordinary", retries: 2, want: []int{901, 902, 903}},
				{name: "affinity miss", affinity: true, miss: true, retries: 2, want: []int{901, 902, 903}},
				{name: "affinity hit", affinity: true, retries: 2, want: []int{902, 901, 902}},
				{name: "zero budget", affinity: true, want: []int{902}},
				{name: "one retry", affinity: true, retries: 1, want: []int{902, 901}},
				{name: "skip retry", affinity: true, skipRetry: true, retries: 2, want: []int{902}},
				{name: "auto same group", affinity: true, retries: 1, autoGroups: `["default","vip"]`, want: []int{902, 901}},
				{name: "cross group", affinity: true, retries: 1, autoGroups: `["default","vip"]`, crossGroup: true, want: []int{902, 901, 904, 905}},
				{name: "cross empty group", affinity: true, retries: 1, autoGroups: `["default","empty","vip"]`, crossGroup: true, want: []int{902, 901, 904, 905}},
				{name: "token exclusion", affinity: true, retries: 1, excludeHigh: true, want: []int{902, 902}},
			} {
				t.Run(fmt.Sprintf("cache=%t/exhaust=%t/%s", cached, exhaust, tc.name), func(t *testing.T) {
					c := setupAffinityPriorityChannels(t, cached)
					require.NoError(t, model.DB.Model(&model.Channel{}).Where("id >= ?", 904).Update("group", "vip").Error)
					require.NoError(t, model.DB.Model(&model.Ability{}).Where("channel_id >= ?", 904).Update("group", "vip").Error)
					model.InitChannelCache()
					oldAuto, oldGroups, oldRetries := setting.AutoGroups2JsonString(), setting.UserUsableGroups2JSONString(), common.RetryTimes
					t.Cleanup(func() {
						common.RetryTimes = oldRetries
						require.NoError(t, setting.UpdateAutoGroupsByJsonString(oldAuto))
						require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(oldGroups))
					})
					common.RetryTimes = tc.retries
					common.SetContextKey(c, constant.ContextKeyUserGroup, "default")
					group := "default"
					if tc.autoGroups != "" {
						require.NoError(t, setting.UpdateAutoGroupsByJsonString(tc.autoGroups))
						require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"default","vip":"vip","empty":"empty"}`))
						common.SetContextKey(c, constant.ContextKeyTokenCrossGroupRetry, tc.crossGroup)
						group = "auto"
					}
					if tc.excludeHigh {
						common.SetContextKey(c, constant.ContextKeyTokenExcludedChannels, []int{901})
					}
					var channel *model.Channel
					if tc.affinity {
						// Populate the matching-rule metadata without initializing the
						// process-wide affinity cache or capturing a test Redis client.
						setChannelAffinityContext(c, channelAffinityMeta{RuleName: t.Name(), SkipRetry: tc.skipRetry})
						if !tc.miss {
							channel = &model.Channel{Id: 902}
							MarkChannelAffinityUsed(c, "default", channel.Id)
							common.SetContextKey(c, constant.ContextKeyAutoGroupIndex, 0)
						}
					}
					// Distribution and the relay loop use separate retry structures.
					if channel == nil {
						var err error
						channel, _, err = CacheGetRandomSatisfiedChannel(&RetryParam{Ctx: c, TokenGroup: group, ModelName: "gpt-5.6-sol", ExhaustCandidates: exhaust})
						require.NoError(t, err)
					}
					param := &RetryParam{Ctx: c, TokenGroup: group, ModelName: "gpt-5.6-sol", ExhaustCandidates: exhaust}
					want := append([]int(nil), tc.want...)
					if exhaust && tc.name == "affinity hit" {
						want = []int{902, 901, 903}
					}
					if exhaust && tc.excludeHigh {
						want = []int{902, 903}
					}
					var attempts []int
					for ; param.GetRetry() <= common.RetryTimes; param.IncreaseRetry() {
						if len(attempts) > 0 {
							var err error
							channel, _, err = CacheGetRandomSatisfiedChannel(param)
							require.NoError(t, err)
						}
						if channel == nil {
							break
						}
						attempts = append(attempts, channel.Id)
						require.LessOrEqual(t, len(attempts), len(want), "affinity must not increase retry budget")
						if ShouldSkipRetryAfterChannelAffinityFailure(c) {
							break
						}
						if exhaust {
							param.ExcludedChannelIds = append(param.ExcludedChannelIds, channel.Id)
						}
					}
					require.Equal(t, want, attempts)
				})
			}
		}
	}
}

func setupAffinityPriorityChannels(t *testing.T, cached bool) *gin.Context {
	t.Helper()
	oldDB, oldCache, oldRedis := model.DB, common.MemoryCacheEnabled, common.RedisEnabled
	oldMaster, oldSQLitePath := common.IsMasterNode, common.SQLitePath
	oldMainType, oldLogType := common.MainDatabaseType(), common.LogDatabaseType()
	t.Cleanup(func() {
		model.DB, common.MemoryCacheEnabled, common.RedisEnabled = oldDB, oldCache, oldRedis
		common.IsMasterNode, common.SQLitePath = oldMaster, oldSQLitePath
		common.SetMainDatabaseType(oldMainType)
		common.SetLogDatabaseType(oldLogType)
		if oldCache && oldDB != nil {
			model.InitChannelCache()
		}
	})
	common.MemoryCacheEnabled, common.RedisEnabled = cached, false
	t.Setenv("SQL_DSN", "local")
	common.IsMasterNode = false
	common.SQLitePath = filepath.Join(t.TempDir(), "affinity-priority.db")
	require.NoError(t, model.InitDB())
	sqlDB, err := model.DB.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, model.DB.AutoMigrate(&model.Channel{}, &model.Ability{}))
	for index, priority := range []int64{100, 90, 80, 100, 90, 80} {
		channel := model.Channel{Id: 901 + index, Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled,
			Group: "default", Models: "gpt-5.6-sol", Priority: common.GetPointer(priority), Weight: common.GetPointer(uint(1))}
		require.NoError(t, model.DB.Create(&channel).Error)
		require.NoError(t, channel.AddAbilities(nil))
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	return c
}
