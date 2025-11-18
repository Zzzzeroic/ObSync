package api

import (
	"net/http"

	"obsync/internal/handlers"
	"obsync/internal/store"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, db *store.SQLiteStore) {
	v1 := r.Group("/v1")
	{
		v1.POST("/register-device", func(c *gin.Context) {
			handlers.RegisterDeviceHandler(c, db)
		})

		v1.GET("/repos/:repo/changes", func(c *gin.Context) {
			handlers.ListChangesHandler(c, db)
		})

		v1.POST("/repos/:repo/changes", func(c *gin.Context) {
			handlers.PostChangesHandler(c, db)
		})

		// 测试服务器存活用的接口
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})

		// redirect the legacy openapi path to the Swagger UI index page (temporary redirect to avoid client caching)
		v1.GET("/openapi.json", func(c *gin.Context) {
			c.Redirect(http.StatusFound, "/swagger/index.html")
		})
	}
	// Note: Swagger UI is served by gin-swagger at /swagger/*any (embedded docs)
}
