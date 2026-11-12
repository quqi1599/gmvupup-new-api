package controller

import (
	"bytes"
	"fmt"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// Released order schema, before the additive RMB fields were introduced.
type topUpBeforeRMB struct {
	Id              int
	UserId          int `gorm:"index"`
	Amount          int64
	Money           float64
	TradeNo         string `gorm:"unique;type:varchar(255);index"`
	PaymentMethod   string `gorm:"type:varchar(50)"`
	PaymentProvider string `gorm:"type:varchar(50);default:''"`
	CreateTime      int64
	CompleteTime    int64
	Status          string
}

func TestCNYTopUpDatabaseCompatibility(t *testing.T) {
	for _, engine := range []string{"sqlite", "mysql", "postgres"} {
		for _, migration := range []string{"fresh", "upgrade"} {
			t.Run(engine+"/"+migration, func(t *testing.T) {
				var dialector gorm.Dialector
				var databaseType common.DatabaseType
				switch engine {
				case "sqlite":
					dialector = sqlite.Open(filepath.Join(t.TempDir(), "topup.db"))
					databaseType = common.DatabaseTypeSQLite
				case "mysql", "postgres":
					env := "TEST_TOPUP_MYSQL_DSN"
					if engine == "postgres" {
						env = "TEST_TOPUP_POSTGRES_DSN"
					}
					dsn := os.Getenv(env)
					if dsn == "" {
						if os.Getenv("REQUIRE_TOPUP_DB_MATRIX") == "1" {
							t.Fatal(env + " is required for the release database matrix")
						}
						t.Skip(env + " is not configured")
					}
					if engine == "mysql" {
						dialector = mysql.Open(dsn)
						databaseType = common.DatabaseTypeMySQL
					} else {
						dialector = postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true})
						databaseType = common.DatabaseTypePostgreSQL
					}
				}
				var statements bytes.Buffer
				prefix := fmt.Sprintf("rmb_%d_", time.Now().UnixNano())
				db, err := gorm.Open(dialector, &gorm.Config{
					NamingStrategy: schema.NamingStrategy{TablePrefix: prefix},
					Logger:         gormlogger.New(log.New(&statements, "", 0), gormlogger.Config{LogLevel: gormlogger.Info}),
				})
				require.NoError(t, err)
				sqlDB, err := db.DB()
				require.NoError(t, err)
				if engine == "sqlite" {
					sqlDB.SetMaxOpenConns(1)
				} else {
					sqlDB.SetMaxOpenConns(4)
				}
				oldDB, oldLogDB, oldRedis := model.DB, model.LOG_DB, common.RedisEnabled
				oldMain, oldLog := common.MainDatabaseType(), common.LogDatabaseType()
				oldMaster, oldUnit := common.IsMasterNode, common.QuotaPerUnit
				t.Cleanup(func() {
					_ = db.Migrator().DropTable(&model.Log{}, &model.TopUp{}, &model.User{})
					_ = sqlDB.Close()
					model.DB, model.LOG_DB, common.RedisEnabled = oldDB, oldLogDB, oldRedis
					common.SetMainDatabaseType(oldMain)
					common.SetLogDatabaseType(oldLog)
					common.IsMasterNode, common.QuotaPerUnit = oldMaster, oldUnit
				})
				model.DB, model.LOG_DB = db, db
				common.RedisEnabled, common.IsMasterNode, common.QuotaPerUnit = false, false, 500000
				common.SetMainDatabaseType(databaseType)
				common.SetLogDatabaseType(databaseType)
				versionSQL := "SELECT version()"
				if engine == "sqlite" {
					versionSQL = "SELECT sqlite_version()"
				}
				var version string
				require.NoError(t, db.Raw(versionSQL).Scan(&version).Error)
				t.Logf("engine=%s version=%s migration=%s", engine, version, migration)
				require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}))
				require.NoError(t, db.Create(&model.User{Id: 77, Username: "matrix-" + prefix, Quota: 123456, Status: common.UserStatusEnabled}).Error)
				if migration == "upgrade" {
					table := prefix + "top_ups"
					require.NoError(t, db.Table(table).AutoMigrate(&topUpBeforeRMB{}))
					require.NoError(t, db.Table(table).Create(&topUpBeforeRMB{UserId: 77, Amount: 1, Money: 7.3, TradeNo: "legacy-matrix", PaymentMethod: "alipay", PaymentProvider: model.PaymentProviderEpay, Status: common.TopUpStatusPending}).Error)
				}
				require.NoError(t, db.AutoMigrate(&model.TopUp{}))
				statements.Reset()
				require.NoError(t, db.AutoMigrate(&model.TopUp{}))
				repeated := strings.ToUpper(statements.String())
				assert.NotContains(t, repeated, "ALTER TABLE")
				assert.NotContains(t, repeated, "CREATE TABLE")
				var account model.User
				require.NoError(t, db.First(&account, 77).Error)
				assert.Equal(t, 123456, account.Quota, "migration preserves an existing wallet balance")
				if migration == "upgrade" {
					legacy := model.GetTopUpByTradeNo("legacy-matrix")
					require.NotNil(t, legacy)
					assert.Equal(t, int64(1), legacy.Amount)
					assert.Equal(t, 7.3, legacy.Money)
					assert.Zero(t, legacy.AmountCNY)
					assert.Zero(t, legacy.CreditedQuota)
					assert.Equal(t, common.TopUpStatusPending, legacy.Status)
				}
				order := model.TopUp{UserId: 77, AmountCNY: 1, CreditedQuota: 68493, Money: 1, TradeNo: "rmb-matrix", PaymentMethod: "alipay", PaymentProvider: model.PaymentProviderEpay, Status: common.TopUpStatusPending}
				require.NoError(t, db.Create(&order).Error)
				duplicate := order
				duplicate.Id = 0
				require.Error(t, db.Create(&duplicate).Error, "order number uniqueness survives migration")
				start := make(chan struct{})
				results := make(chan error, 2)
				for range 2 {
					go func() {
						<-start
						_, settleErr := model.RechargeEpay(order.TradeNo, "alipay", "")
						results <- settleErr
					}()
				}
				close(start)
				require.NoError(t, <-results)
				require.NoError(t, <-results)
				require.NoError(t, db.First(&account, 77).Error)
				assert.Equal(t, 123456+68493, account.Quota, "concurrent callbacks credit exactly once")
				assert.Equal(t, common.TopUpStatusSuccess, model.GetTopUpByTradeNo(order.TradeNo).Status)
				var count int64
				require.NoError(t, db.Model(&model.Log{}).Where("user_id = ? AND type = ?", 77, model.LogTypeTopup).Count(&count).Error)
				assert.Equal(t, int64(1), count)
				if migration == "upgrade" {
					_, err := model.RechargeEpay("legacy-matrix", "alipay", "")
					require.NoError(t, err)
					require.NoError(t, db.First(&account, 77).Error)
					assert.Equal(t, 123456+68493+500000, account.Quota)
				}
				require.NoError(t, db.Model(&model.User{}).Where("id = ?", 77).Update("quota", common.MaxWalletQuota-10).Error)
				full := model.TopUp{UserId: 77, AmountCNY: 1, CreditedQuota: 68493, Money: 1, TradeNo: "rmb-full", PaymentProvider: model.PaymentProviderEpay, Status: common.TopUpStatusPending}
				require.NoError(t, db.Create(&full).Error)
				_, err = model.RechargeEpay(full.TradeNo, "alipay", "")
				require.ErrorIs(t, err, model.ErrTopUpQuotaLimitExceeded)
				assert.Equal(t, common.TopUpStatusPending, model.GetTopUpByTradeNo(full.TradeNo).Status)
				require.NoError(t, db.First(&account, 77).Error)
				assert.Equal(t, common.MaxWalletQuota-10, account.Quota)
			})
		}
	}
}

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

func TestCNYTopUpPaysAndCreditsTheEnteredYuan(t *testing.T) {
	db := setupEpayOrderTest(t)
	oldType, oldRate, oldPrice, oldMin, oldUnit := operation_setting.GetQuotaDisplayType(), operation_setting.USDExchangeRate, operation_setting.Price, operation_setting.MinTopUp, common.QuotaPerUnit
	t.Cleanup(func() {
		operation_setting.GetGeneralSetting().QuotaDisplayType = oldType
		operation_setting.USDExchangeRate, operation_setting.Price, operation_setting.MinTopUp, common.QuotaPerUnit = oldRate, oldPrice, oldMin, oldUnit
	})
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeCNY
	operation_setting.USDExchangeRate, operation_setting.Price, operation_setting.MinTopUp, common.QuotaPerUnit = 7.3, 7.3, 1, 500000
	infoWriter := httptest.NewRecorder()
	infoContext, _ := gin.CreateTestContext(infoWriter)
	GetTopUpInfo(infoContext)
	var info struct {
		Data struct {
			AmountCurrency string          `json:"amount_currency"`
			Discount       map[int]float64 `json:"discount"`
		}
	}
	require.NoError(t, common.Unmarshal(infoWriter.Body.Bytes(), &info))
	assert.Equal(t, "CNY", info.Data.AmountCurrency)
	assert.Empty(t, info.Data.Discount)
	for _, amount := range []int64{1, 10, 100} {
		t.Run(fmt.Sprint(amount), func(t *testing.T) {
			assert.InDelta(t, float64(amount), getPayMoney(amount, "default"), 0.000001)
			quoteWriter := httptest.NewRecorder()
			quoteContext, _ := gin.CreateTestContext(quoteWriter)
			quoteContext.Set("id", 42)
			quoteContext.Request = httptest.NewRequest(http.MethodPost, "/amount", strings.NewReader(fmt.Sprintf(`{"amount":%d}`, amount)))
			quoteContext.Request.Header.Set("Content-Type", "application/json")
			RequestAmount(quoteContext)
			assert.JSONEq(t, fmt.Sprintf(`{"message":"success","data":"%d.00"}`, amount), quoteWriter.Body.String())
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.Set("id", 42)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/pay", strings.NewReader(fmt.Sprintf(`{"amount":%d,"payment_method":"alipay"}`, amount)))
			ctx.Request.Header.Set("Content-Type", "application/json")
			RequestEpay(ctx)
			var response struct {
				Message string
				Data    map[string]string
			}
			require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
			require.Equal(t, "success", response.Message, w.Body.String())
			assert.Equal(t, fmt.Sprintf("%d.00", amount), response.Data["money"])
			var order model.TopUp
			require.NoError(t, db.Where("trade_no = ?", response.Data["out_trade_no"]).First(&order).Error)
			assert.Equal(t, float64(amount), order.Money)

			assert.Zero(t, order.Amount, "old binaries must reject new RMB orders instead of overcrediting")
			assert.Equal(t, amount, order.AmountCNY)
			expectedQuota, quotaErr := common.WalletQuotaFromDecimalStrict(decimal.NewFromInt(amount).Mul(decimal.NewFromInt(500000)).Div(decimal.NewFromFloat(7.3)))
			require.NoError(t, quotaErr)
			assert.Equal(t, int64(expectedQuota), order.CreditedQuota)
			var before model.User
			require.NoError(t, db.First(&before, 42).Error)
			// A changed rate after checkout must not alter the persisted wallet credit.
			operation_setting.USDExchangeRate = 1
			t.Cleanup(func() { operation_setting.USDExchangeRate = 7.3 })
			router := gin.New()
			router.GET("/notify", EpayNotify)
			for _, paid := range []string{"0.01", fmt.Sprintf("%d.00", amount), fmt.Sprintf("%d.00", amount)} {
				params := epay.GenerateParams(map[string]string{"pid": "merchant", "money": paid, "type": "alipay", "trade_no": "provider-" + order.TradeNo, "out_trade_no": order.TradeNo, "trade_status": "TRADE_SUCCESS", "sign_type": "MD5"}, "test-key")
				query := url.Values{}
				for k, v := range params {
					query.Set(k, v)
				}
				w := httptest.NewRecorder()
				router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/notify?"+query.Encode(), nil))
				var after model.User
				require.NoError(t, db.First(&after, 42).Error)
				if paid == "0.01" {
					require.Equal(t, "fail", w.Body.String())
					assert.Equal(t, before.Quota, after.Quota)
				} else {
					require.Equal(t, "success", w.Body.String())
					assert.Equal(t, before.Quota+expectedQuota, after.Quota)
					var entry model.Log
					require.NoError(t, db.Where("user_id = ? AND type = ?", 42, model.LogTypeTopup).Last(&entry).Error)
					assert.Contains(t, entry.Content, fmt.Sprintf("¥%d.00", amount))
				}
			}
		})
	}
}

func TestCNYTopUpRejectsInvalidRateAndWalletOverflow(t *testing.T) {
	db := setupEpayOrderTest(t)
	oldType, oldRate, oldUnit := operation_setting.GetQuotaDisplayType(), operation_setting.USDExchangeRate, common.QuotaPerUnit
	t.Cleanup(func() {
		operation_setting.GetGeneralSetting().QuotaDisplayType = oldType
		operation_setting.USDExchangeRate = oldRate
		common.QuotaPerUnit = oldUnit
	})
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeCNY
	common.QuotaPerUnit = 500000
	for _, rate := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		operation_setting.USDExchangeRate = rate
		_, err := validateEpayTopUpQuota(42, 1)
		require.Error(t, err)
	}
	operation_setting.USDExchangeRate = 7.3
	for _, amount := range []int64{0, -1, math.MaxInt64} {
		_, err := validateEpayTopUpQuota(42, amount)
		require.Error(t, err)
	}
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", 42).Update("quota", common.MaxWalletQuota-10).Error)
	_, err := validateEpayTopUpQuota(42, 1)
	require.ErrorIs(t, err, model.ErrTopUpQuotaLimitExceeded)
}

func TestCNYManualCompletionUsesSnapshotAndLegacyOrdersKeepOriginalValue(t *testing.T) {
	db := setupEpayOrderTest(t)
	oldUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500000
	t.Cleanup(func() { common.QuotaPerUnit = oldUnit })
	order := &model.TopUp{UserId: 42, Amount: 0, AmountCNY: 1, CreditedQuota: 68493, Money: 1, TradeNo: "rmb-manual", PaymentMethod: "alipay", PaymentProvider: model.PaymentProviderEpay, Status: common.TopUpStatusPending}
	require.NoError(t, db.Create(order).Error)
	require.NoError(t, model.ManualCompleteTopUp(order.TradeNo, ""))
	var account model.User
	require.NoError(t, db.First(&account, 42).Error)
	assert.Equal(t, 68493, account.Quota)
	var logCount int64
	require.NoError(t, db.Model(&model.Log{}).Where("user_id = ? AND type = ?", 42, model.LogTypeTopup).Count(&logCount).Error)
	require.Equal(t, int64(1), logCount)
	require.NoError(t, model.ManualCompleteTopUp(order.TradeNo, ""))
	require.NoError(t, db.Model(&model.Log{}).Where("user_id = ? AND type = ?", 42, model.LogTypeTopup).Count(&logCount).Error)
	assert.Equal(t, int64(1), logCount, "duplicate manual completion must not record another credit")
	done, err := model.RechargeEpay(order.TradeNo, "alipay", "")
	require.NoError(t, err)
	assert.True(t, done)
	require.NoError(t, db.First(&account, 42).Error)
	assert.Equal(t, 68493, account.Quota)
	// The existing test order is a legacy two-USD credit, regardless of CNY UI.
	_, err = model.RechargeEpay("12345678901234567890123456789012", "alipay", "")
	require.NoError(t, err)
	require.NoError(t, db.First(&account, 42).Error)
	assert.Equal(t, 1068493, account.Quota)
	// Missing snapshots cannot be settled as a legacy order.
	invalid := &model.TopUp{UserId: 42, AmountCNY: 1, Money: 1, TradeNo: "rmb-missing-credit", PaymentProvider: model.PaymentProviderEpay, Status: common.TopUpStatusPending}
	require.NoError(t, db.Create(invalid).Error)
	_, err = model.RechargeEpay(invalid.TradeNo, "alipay", "")
	require.ErrorIs(t, err, model.ErrInvalidTopUpQuota)
	assert.Equal(t, common.TopUpStatusPending, model.GetTopUpByTradeNo(invalid.TradeNo).Status)
	// A full wallet leaves the order pending, with no partial successful state.
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", 42).Update("quota", common.MaxWalletQuota-10).Error)
	full := &model.TopUp{UserId: 42, AmountCNY: 1, CreditedQuota: 68493, Money: 1, TradeNo: "rmb-wallet-full", PaymentProvider: model.PaymentProviderEpay, Status: common.TopUpStatusPending}
	require.NoError(t, db.Create(full).Error)
	_, err = model.RechargeEpay(full.TradeNo, "alipay", "")
	require.ErrorIs(t, err, model.ErrTopUpQuotaLimitExceeded)
	assert.Equal(t, common.TopUpStatusPending, model.GetTopUpByTradeNo(full.TradeNo).Status)
}
