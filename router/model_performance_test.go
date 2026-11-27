package router

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestModelPerformanceRouteRequiresAdmin(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "router.db")), &gorm.Config{})
	require.NoError(t, err)
	sql, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sql.Close() })
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.SaaSGiftAccount{}))
	require.NoError(t, db.Create(&model.User{Id: 90, Username: "fixture", Role: common.RoleAdminUser, Status: common.UserStatusEnabled}).Error)
	prior := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = prior })
	r := gin.New()
	r.Use(sessions.Sessions("synthetic-role-fixture", cookie.NewStore([]byte("local-test-only-cookie-signing-key"))))
	r.GET("/fixture/session/:role", func(c *gin.Context) {
		role, _ := strconv.Atoi(c.Param("role"))
		s := sessions.Default(c)
		s.Set("username", "fixture")
		s.Set("role", role)
		s.Set("id", 90)
		s.Set("status", common.UserStatusEnabled)
		require.NoError(t, s.Save())
		c.Status(http.StatusNoContent)
	})
	SetApiRouter(r)
	request := func(path string, cookies []*http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("New-Api-User", "90")
		for _, c := range cookies {
			req.AddCookie(c)
		}
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)
		return res
	}
	url := "/api/model-performance?minutes=60"
	require.Equal(t, http.StatusUnauthorized, request(url, nil).Code)
	ordinary := request("/fixture/session/1", nil).Result().Cookies()
	require.Len(t, ordinary, 1)
	rejected := request(url, ordinary)
	require.Contains(t, rejected.Body.String(), "\"success\":false")
	require.NotContains(t, rejected.Body.String(), "\"models\"")
	admin := request("/fixture/session/10", nil).Result().Cookies()
	require.Len(t, admin, 1)
	allowed := request(url, admin)
	require.Equal(t, http.StatusOK, allowed.Code)
	require.Equal(t, "no-store", allowed.Header().Get("Cache-Control"))
	require.Contains(t, allowed.Body.String(), "\"success\":true")
	require.Contains(t, allowed.Body.String(), "\"ready\":false")
	require.Equal(t, http.StatusBadRequest, request("/api/model-performance?minutes=9999", admin).Code)
}
