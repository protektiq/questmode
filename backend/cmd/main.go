package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"questmode/backend/internal/claude"
	"questmode/backend/internal/handlers"
	questmath "questmode/backend/internal/math"
	"questmode/backend/internal/migrate"
	"questmode/backend/internal/spelling"
)

const (
	defaultPort      = "8080"
	minConnStringLen = 8
	maxConnStringLen = 2048
	maxAnswerLen     = 256
	maxSeedWords     = 500
	maxGenreLen      = 64
)

// AppState holds shared infrastructure for HTTP handlers.
type AppState struct {
	DB    *pgxpool.Pool
	Redis *redis.Client
}

type spellingCheckRequest struct {
	WordID string `json:"word_id"`
	Answer string `json:"answer"`
}

type spellingSeedRequest struct {
	Words []string `json:"words"`
}

type mathCheckRequest struct {
	SessionID string `json:"session_id"`
	ProblemID string `json:"problem_id"`
	Answer    int    `json:"answer"`
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
	storyHandler := handlers.NewStoryHandler(pool, rdb, buildClaudeClient())
	router := gin.Default()
	registerRoutes(router, state, storyHandler)

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
	err := godotenv.Load()
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
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

func registerRoutes(router *gin.Engine, state *AppState, storyHandler *handlers.StoryHandler) {
	api := router.Group("/api")

	api.GET("/health", func(c *gin.Context) {
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

	storyGroup := api.Group("/story")
	storyHandler.RegisterStoryRoutes(storyGroup)

	spellingGroup := api.Group("/spelling")
	spellingGroup.GET("/word", func(c *gin.Context) {
		learnerID, err := resolveLatestLearnerID(c.Request.Context(), state.DB)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "no session found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve learner session"})
			return
		}

		word, err := spelling.GetActiveWord(c.Request.Context(), learnerID, state.DB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load active spelling word"})
			return
		}
		if word.ID == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "no active spelling word"})
			return
		}
		c.JSON(http.StatusOK, word)
	})

	spellingGroup.POST("/check", func(c *gin.Context) {
		var req spellingCheckRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		req.WordID = strings.TrimSpace(req.WordID)
		req.Answer = strings.TrimSpace(req.Answer)
		if req.WordID == "" || req.Answer == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "word_id and answer are required"})
			return
		}
		if len(req.Answer) > maxAnswerLen {
			c.JSON(http.StatusBadRequest, gin.H{"error": "answer exceeds max length"})
			return
		}

		learnerID, err := resolveLatestLearnerID(c.Request.Context(), state.DB)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "no session found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve learner session"})
			return
		}

		result, err := spelling.CheckAnswer(c.Request.Context(), learnerID, req.WordID, req.Answer, state.DB)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	spellingGroup.POST("/seed", func(c *gin.Context) {
		adminKey := strings.TrimSpace(os.Getenv("ADMIN_KEY"))
		if adminKey == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "admin key is not configured"})
			return
		}
		if strings.TrimSpace(c.GetHeader("X-Admin-Key")) != adminKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var req spellingSeedRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		if len(req.Words) > maxSeedWords {
			c.JSON(http.StatusBadRequest, gin.H{"error": "word list exceeds max size"})
			return
		}

		learnerID, err := resolveLatestLearnerID(c.Request.Context(), state.DB)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "no session found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve learner session"})
			return
		}

		if err := spelling.SeedWordList(c.Request.Context(), learnerID, req.Words, state.DB); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"seeded": true})
	})

	mathGroup := api.Group("/math")
	mathGroup.GET("/problem", func(c *gin.Context) {
		difficulty := strings.TrimSpace(c.Query("difficulty"))
		genre := strings.TrimSpace(c.Query("genre"))
		frustratedRaw := strings.TrimSpace(c.Query("frustrated"))
		frustrated := false

		if genre != "" && len(genre) > maxGenreLen {
			c.JSON(http.StatusBadRequest, gin.H{"error": "genre exceeds max length"})
			return
		}

		if difficulty != "" {
			switch strings.ToLower(difficulty) {
			case string(questmath.DiffEasy), string(questmath.DiffMedium), string(questmath.DiffHard):
			default:
				c.JSON(http.StatusBadRequest, gin.H{"error": "difficulty must be easy, medium, or hard"})
				return
			}
		}

		if frustratedRaw != "" {
			parsed, err := strconv.ParseBool(frustratedRaw)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "frustrated must be a boolean"})
				return
			}
			frustrated = parsed
		}

		problem := questmath.GetProblem(questmath.Difficulty(difficulty), genre, frustrated)
		if problem.ID == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "no problem available"})
			return
		}
		c.JSON(http.StatusOK, questmath.PublicProblem(problem))
	})

	mathGroup.POST("/check", func(c *gin.Context) {
		var req mathCheckRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		req.SessionID = strings.TrimSpace(req.SessionID)
		req.ProblemID = strings.TrimSpace(req.ProblemID)
		if req.SessionID == "" || req.ProblemID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id and problem_id are required"})
			return
		}

		learnerID, err := resolveLatestLearnerID(c.Request.Context(), state.DB)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "no session found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resolve learner session"})
			return
		}

		result, err := questmath.CheckAnswer(
			c.Request.Context(),
			learnerID,
			req.SessionID,
			req.ProblemID,
			req.Answer,
			state.DB,
		)
		if err != nil {
			switch err.Error() {
			case "learner_id, session_id, and problem_id are required",
				"learner_id must be a valid uuid",
				"session_id must be a valid uuid",
				"db is required",
				"problem not found":
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if strings.Contains(err.Error(), "exceeds max length") || strings.Contains(err.Error(), "answer must be between") {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check math answer"})
			return
		}
		c.JSON(http.StatusOK, result)
	})
}

func resolveLatestLearnerID(ctx context.Context, db *pgxpool.Pool) (string, error) {
	var learnerID string
	err := db.QueryRow(ctx, `
		SELECT learner_id::text
		FROM quest_sessions
		ORDER BY started_at DESC
		LIMIT 1
	`).Scan(&learnerID)
	if err != nil {
		return "", err
	}
	return learnerID, nil
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

func buildClaudeClient() *claude.ClaudeClient {
	apiKey := strings.TrimSpace(os.Getenv("CLAUDE_API_KEY"))
	if apiKey == "" {
		return nil
	}
	client, err := claude.NewClaudeClient(apiKey)
	if err != nil {
		log.Printf("claude client disabled: %v", err)
		return nil
	}
	return client
}
