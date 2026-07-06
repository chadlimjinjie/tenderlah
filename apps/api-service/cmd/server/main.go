// @title       Midtone API
// @version     1.0.0
// @description Midtone API server (Go rewrite)
// @host        localhost:4002
// @BasePath    /
// @securityDefinitions.apikey BearerAuth
// @in          header
// @name        Authorization
package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/midtone/api-server/internal/auth"
	"github.com/midtone/api-server/internal/db"
	"github.com/midtone/api-server/internal/handlers/maintenance"
	"github.com/midtone/api-server/internal/handlers/tenders"
	"github.com/midtone/api-server/internal/handlers/users"
	"github.com/midtone/api-server/internal/llm"
	mw "github.com/midtone/api-server/internal/middleware"
	"github.com/midtone/api-server/internal/storage"

	_ "github.com/midtone/api-server/docs"
)

func main() {
	_ = godotenv.Load()

	auth.Init(
		mustEnv("AUTH_SERVICE_URL", "http://localhost:3100"),
	)
	db.Init(mustEnv("DATABASE_URL", ""))

	q := db.New(db.Pool)

	ollamaClient := llm.New(
		mustEnv("OLLAMA_BASE_URL", "https://ollama.com"),
		os.Getenv("OLLAMA_API_KEY"),
		llm.DefaultModel,
	)

	supabaseStorage := storage.New(
		mustEnv("SUPABASE_URL", ""),
		mustEnv("SUPABASE_SERVICE_ROLE_KEY", ""),
		mustEnv("SUPABASE_STORAGE_BUCKET", "catalogue"),
	)

	r := gin.Default()
	r.Use(mw.Authenticate())

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true, "service": "midtone-api", "version": "1.0.0"})
	})

	api := r.Group("/api")
	maintenance.New(q).Register(api.Group("/maintenance"))
	tenders.New(q, ollamaClient, mustEnv("OLLAMA_EMBED_BASE_URL", "http://localhost:11434")).Register(api.Group("/tenders"))
	users.New(q, supabaseStorage, mustEnv("OLLAMA_EMBED_BASE_URL", "http://localhost:11434")).Register(api.Group("/users"))

	r.GET("/api/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	port := mustEnv("PORT", "4002")
	log.Printf("Listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

func mustEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	if fallback == "" {
		log.Fatalf("required env var %s not set", key)
	}
	return fallback
}
