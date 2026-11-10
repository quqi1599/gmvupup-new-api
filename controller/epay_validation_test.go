package controller

import (
	"fmt"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupEpayOrderTest(t *testing.T) *gorm.DB {
	t.Helper()
	confirmPaymentComplianceForTest(t)
	oldDB, oldLogDB := model.DB, model.LOG_DB
	oldAddress, oldID, oldKey, oldMethods := operation_setting.PayAddress, operation_setting.EpayId, operation_setting.EpayKey, operation_setting.PayMethods
	oldRedis := common.RedisEnabled
	t.Cleanup(func() {
		model.DB, model.LOG_DB = oldDB, oldLogDB
		operation_setting.PayAddress, operation_setting.EpayId, operation_setting.EpayKey, operation_setting.PayMethods = oldAddress, oldID, oldKey, oldMethods
		common.RedisEnabled = oldRedis
	})
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.TopUp{}, &model.Log{}, &model.SubscriptionOrder{}))
	model.DB, model.LOG_DB = db, db
	oldMainType, oldLogType, oldMaster := common.MainDatabaseType(), common.LogDatabaseType(), common.IsMasterNode
	t.Cleanup(func() {
		common.SetMainDatabaseType(oldMainType)
		common.SetLogDatabaseType(oldLogType)
		common.IsMasterNode = oldMaster
	})
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	common.IsMasterNode = false
	t.Setenv("LOG_SQL_DSN", "")
	require.NoError(t, model.InitLogDB())
	common.RedisEnabled = false
	operation_setting.PayAddress, operation_setting.EpayId, operation_setting.EpayKey = "https://zpayz.cn", "merchant", "test-key"
	operation_setting.PayMethods = []map[string]string{{"type": "alipay"}, {"type": "wxpay"}}
	require.NoError(t, db.Create(&model.User{Id: 42, Username: "zpay-test", Status: common.UserStatusEnabled}).Error)
	require.NoError(t, db.Create(&model.TopUp{UserId: 42, Amount: 2, Money: 9.99, TradeNo: "12345678901234567890123456789012", PaymentMethod: "alipay", PaymentProvider: model.PaymentProviderEpay, Status: common.TopUpStatusPending}).Error)
	return db
}

func TestEpayNotifyChecksPaidAmountAndMerchant(t *testing.T) {
	db := setupEpayOrderTest(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/notify", EpayNotify)
	for _, tc := range []struct{ name, money, pid, want string }{
		{"underpaid", "0.01", "merchant", "fail"},
		{"overpaid", "10.00", "merchant", "fail"},
		{"wrong merchant", "9.99", "other", "fail"},
		{"valid", "9.99", "merchant", "success"},
		{"duplicate", "9.99", "merchant", "success"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := epay.GenerateParams(map[string]string{"pid": tc.pid, "money": tc.money, "type": "alipay", "trade_no": "zpay-1", "out_trade_no": "12345678901234567890123456789012", "trade_status": "TRADE_SUCCESS", "sign_type": "MD5"}, "test-key")
			query := url.Values{}
			for k, v := range p {
				query.Set(k, v)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/notify?"+query.Encode(), nil))
			require.Equal(t, tc.want, w.Body.String())
			var user model.User
			require.NoError(t, db.First(&user, 42).Error)
			if tc.want == "fail" {
				require.Zero(t, user.Quota)
				require.Equal(t, common.TopUpStatusPending, model.GetTopUpByTradeNo("12345678901234567890123456789012").Status)
			} else {
				require.Equal(t, int(2*common.QuotaPerUnit), user.Quota)
			}
		})
	}
}

func TestEpayPaymentAmountValidation(t *testing.T) {
	oldID := operation_setting.EpayId
	operation_setting.EpayId = "merchant"
	t.Cleanup(func() { operation_setting.EpayId = oldID })
	for _, tc := range []struct {
		money    string
		expected float64
		valid    bool
	}{
		{"9.99", 9.99, true}, {"10", 10, true}, {"10.0", 10, true},
		{"1.24", 1.239, true}, {"0", 0, false}, {"-1.00", 1, false},
		{"1e2", 100, false}, {"1.000", 1, false}, {"NaN", 1, false}, {"", 1, false},
		{"9.98", 9.99, false}, {"99999999999999999999999", 1, false},
	} {
		t.Run(fmt.Sprintf("%s_%f", tc.money, tc.expected), func(t *testing.T) {
			assert.Equal(t, tc.valid, validEpayOrderPayment(map[string]string{"pid": "merchant", "money": tc.money, "trade_no": "zpay-1"}, tc.expected))
		})
	}
	assert.False(t, validEpayOrderPayment(map[string]string{"pid": "merchant", "money": "1.00"}, 1))
}

func TestEpayTradeNumbersMeetZpayLimit(t *testing.T) {
	ids := map[string]bool{}
	pattern := regexp.MustCompile(`^[0-9]{32}$`)
	for range 100 {
		id, err := newEpayTradeNo()
		require.NoError(t, err)
		assert.Regexp(t, pattern, id)
		assert.False(t, ids[id])
		ids[id] = true
	}
}

func TestSubscriptionEpayRejectsIncorrectAmountBeforeFulfillment(t *testing.T) {
	db := setupEpayOrderTest(t)
	require.NoError(t, db.Create(&model.SubscriptionOrder{UserId: 42, PlanId: 1, Money: 19.99, TradeNo: "sub-1", PaymentMethod: "alipay", PaymentProvider: model.PaymentProviderEpay, Status: common.TopUpStatusPending}).Error)
	for _, endpoint := range []struct {
		name     string
		handler  gin.HandlerFunc
		redirect bool
	}{
		{"notify", SubscriptionEpayNotify, false}, {"return", SubscriptionEpayReturn, true},
	} {
		for _, method := range []string{http.MethodGet, http.MethodPost} {
			t.Run(endpoint.name+method, func(t *testing.T) {
				p := epay.GenerateParams(map[string]string{"pid": "merchant", "money": "0.01", "type": "alipay", "trade_no": "zpay-sub-1", "out_trade_no": "sub-1", "trade_status": "TRADE_SUCCESS", "sign_type": "MD5"}, "test-key")
				values := url.Values{}
				for k, v := range p {
					values.Set(k, v)
				}
				req := httptest.NewRequest(method, "/callback?"+values.Encode(), nil)
				if method == http.MethodPost {
					req = httptest.NewRequest(method, "/callback", strings.NewReader(values.Encode()))
					req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				}
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				ctx.Request = req
				endpoint.handler(ctx)
				ctx.Writer.WriteHeaderNow()
				if endpoint.redirect {
					assert.Equal(t, http.StatusFound, w.Code)
					assert.Contains(t, w.Header().Get("Location"), "pay=fail")
				} else {
					assert.Equal(t, "fail", w.Body.String())
				}
				assert.Equal(t, common.TopUpStatusPending, model.GetSubscriptionOrderByTradeNo("sub-1").Status)
			})
		}
	}
}

func TestEpayCheckoutUsesZpayContract(t *testing.T) {
	db := setupEpayOrderTest(t)
	oldPrice, oldMin, oldAddress, oldCallback, oldName := operation_setting.Price, operation_setting.MinTopUp, system_setting.ServerAddress, operation_setting.CustomCallbackAddress, common.SystemName
	t.Cleanup(func() {
		operation_setting.Price, operation_setting.MinTopUp, system_setting.ServerAddress, operation_setting.CustomCallbackAddress, common.SystemName = oldPrice, oldMin, oldAddress, oldCallback, oldName
	})
	operation_setting.Price, operation_setting.MinTopUp = 1, 1
	system_setting.ServerAddress, operation_setting.CustomCallbackAddress, common.SystemName = "https://gmvupup.top", "", "GMVUPUP"
	for _, method := range []string{"alipay", "wxpay"} {
		t.Run(method, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.Set("id", 42)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/pay", strings.NewReader(fmt.Sprintf(`{"amount":10,"payment_method":%q}`, method)))
			ctx.Request.Header.Set("Content-Type", "application/json")
			RequestEpay(ctx)
			var response struct {
				Message string
				URL     string
				Data    map[string]string
			}
			require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
			require.Equal(t, "success", response.Message)
			assert.Equal(t, "https://zpayz.cn/submit.php", response.URL)
			assert.Equal(t, "GMVUPUP API credits", response.Data["name"])
			assert.Equal(t, "https://gmvupup.top/wallet", response.Data["return_url"])
			assert.Equal(t, "https://gmvupup.top/api/user/epay/notify", response.Data["notify_url"])
			assert.Regexp(t, `^[0-9]{32}$`, response.Data["out_trade_no"])
			assert.Equal(t, method, response.Data["type"])
			verified, err := GetEpayClient().Verify(response.Data)
			require.NoError(t, err)
			assert.True(t, verified.VerifyStatus)
			var order model.TopUp
			require.NoError(t, db.Where("trade_no = ?", response.Data["out_trade_no"]).First(&order).Error)
			assert.Equal(t, fmt.Sprintf("%.2f", order.Money), response.Data["money"])
		})
	}
}
