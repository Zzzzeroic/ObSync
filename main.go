package main

import (
	"log"
	"net/http"

	"obsync/docs"
	"obsync/internal/api"
	"obsync/internal/store"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化 SQLite 数据库
	db, err := store.NewSQLiteStore("obsync.db")
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}

	// set swagger info
	docs.SwaggerInfo.Title = "ObSync API"
	docs.SwaggerInfo.Version = "v0.1.0"

	r := gin.Default()

	api.RegisterRoutes(r, db)

	// swagger UI route (embedded docs package) - ensure UI loads embedded /swagger/doc.json
	// Register the wildcard route first to avoid gin routing conflicts.
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger/doc.json")))

	// ensure visiting /swagger goes to the UI index (temporary redirect)
	r.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/swagger/index.html")
	})

	if err := r.Run(); err != nil {
		log.Fatalf("server exit: %v", err)
	}
}
