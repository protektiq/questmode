package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"questmode/backend/internal/migrate"
)

const (
	defaultPort     = "8080"
	minConnStringLen = 8
	maxConnStringLen = 2048
)

// AppState holds shared infrastructure for HTTP handlers.
type AppState struct {
	DB    *pgxpool.Pool
	Redis *redis.Client
}

func main() {
	if err := loadDotenvIfDev(); err != nil {
		panic(err)
	}

	ctx := context.Background()

	databaseURL, err := requireEnv("DATABASE_URL")
	if err != nil {
		panic(err)
	}
	redisURL, err := requireEnv("REDIS_URL")
	if err != nil {
		panic(err)
	}

	pool, err := openPostgres(ctx, databaseURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	rdb, err := openRedis(ctx, redisURL)
	if err != nil {
		panic(err)
	}
	defer func() { _ = rdb.Close() }()

	if err := migrate.Run(ctx, pool); err != nil {
		panic(err)
	}

	state := &AppState{DB: pool, Redis: rdb}
	router := gin.Default()
	registerRoutes(router, state)

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	if err := router.Run(":" + port); err != nil {
		panic(err)
	}
}

func loadDotenvIfDev() error {
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return nil
	}
	if strings.EqualFold(os.Getenv("ENV"), "production") {
		return nil
	}
	return godotenv.Load()
}

func requireEnv(key string) (string, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return "", fmt.Errorf("required environment variable %q is not set", key)
	}
	if len(v) < minConnStringLen || len(v) > maxConnStringLen {
		return "", fmt.Errorf("environment variable %q has invalid length", key)
	}
	return v, nil
}

func openPostgres(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres pool: %w", err)
	}
	pctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	return pool, nil
}

func openRedis(ctx context.Context, redisURL string) (*redis.Client, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse REDIS_URL: %w", err)
	}
	client := redis.NewClient(opt)
	pctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := client.Ping(pctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return client, nil
}

func registerRoutes(router *gin.Engine, state *AppState) {
	router.GET("/api/health", func(c *gin.Context) {
		dbSt, rdsSt, overall := probeDeps(c.Request.Context(), state)
		code := http.StatusOK
		if overall != "ok" {
			code = http.StatusServiceUnavailable
		}
		c.JSON(code, gin.H{
			"status": overall,
			"db":     dbSt,
			"redis":  rdsSt,
		})
	})
}

func probeDeps(ctx context.Context, state *AppState) (dbStatus, redisStatus, overall string) {
	dbStatus = "error"
	redisStatus = "error"

	pctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := state.DB.Ping(pctx); err == nil {
		dbStatus = "ok"
	}

	rctx, rcancel := context.WithTimeout(ctx, 2*time.Second)
	defer rcancel()
	if err := state.Redis.Ping(rctx).Err(); err == nil {
		redisStatus = "ok"
	}

	overall = "ok"
	if dbStatus != "ok" || redisStatus != "ok" {
		overall = "degraded"
	}
	return dbStatus, redisStatus, overall
}
