package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	DB       DBConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Google   GoogleOAuthConfig
	MinIO    MinIOConfig
	Qdrant   QdrantConfig
	AI       AIConfig
	CORS     CORSConfig
}

type AppConfig struct {
	Env  string
	Port string
	Name string
}

type DBConfig struct {
	Host         string
	Port         string
	User         string
	Password     string
	Name         string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret         string
	AccessExpiry   time.Duration
	RefreshExpiry  time.Duration
}

type GoogleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type MinIOConfig struct {
	Endpoint        string
	AccessKey       string
	SecretKey       string
	UseSSL          bool
	BucketDocuments string
	BucketAudio     string
}

type QdrantConfig struct {
	Host                string
	Port                int
	APIKey              string
	CollectionDocuments string
	CollectionMemory    string
}

type AIConfig struct {
	OpenAIKey           string
	AnthropicKey        string
	Model               string
	EmbeddingModel      string
	EmbeddingDimensions int
}

type CORSConfig struct {
	Origins []string
}

func Load() (*Config, error) {
	// Walk up from CWD to find .env — works whether binary runs from project root or backend/.
	_ = godotenv.Load()          // ./env (project root or wherever binary runs)
	_ = godotenv.Load("../.env") // backend/ → project root
	_ = godotenv.Load("../../.env")

	cfg := &Config{
		App: AppConfig{
			Env:  getEnv("APP_ENV", "development"),
			Port: getEnv("APP_PORT", "8080"),
			Name: getEnv("APP_NAME", "jarvas"),
		},
		DB: DBConfig{
			Host:         getEnv("DB_HOST", "localhost"),
			Port:         getEnv("DB_PORT", "5432"),
			User:         getEnv("DB_USER", "jarvas"),
			Password:     mustEnv("DB_PASSWORD"),
			Name:         getEnv("DB_NAME", "jarvas_db"),
			SSLMode:      getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns: getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getEnvInt("DB_MAX_IDLE_CONNS", 10),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:        mustEnv("JWT_SECRET"),
			AccessExpiry:  getEnvDuration("JWT_ACCESS_EXPIRY", 15*time.Minute),
			RefreshExpiry: getEnvDuration("JWT_REFRESH_EXPIRY", 7*24*time.Hour),
		},
		Google: GoogleOAuthConfig{
			ClientID:     mustEnv("GOOGLE_CLIENT_ID"),
			ClientSecret: mustEnv("GOOGLE_CLIENT_SECRET"),
			RedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/v1/auth/google/callback"),
		},
		MinIO: MinIOConfig{
			Endpoint:        getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey:       getEnv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey:       getEnv("MINIO_SECRET_KEY", "minioadmin"),
			UseSSL:          getEnvBool("MINIO_USE_SSL", false),
			BucketDocuments: getEnv("MINIO_BUCKET_DOCUMENTS", "documents"),
			BucketAudio:     getEnv("MINIO_BUCKET_AUDIO", "audio"),
		},
		Qdrant: QdrantConfig{
			Host:                getEnv("QDRANT_HOST", "localhost"),
			Port:                getEnvInt("QDRANT_PORT", 6334),
			APIKey:              getEnv("QDRANT_API_KEY", ""),
			CollectionDocuments: getEnv("QDRANT_COLLECTION_DOCUMENTS", "documents"),
			CollectionMemory:    getEnv("QDRANT_COLLECTION_MEMORY", "memory"),
		},
		AI: AIConfig{
			OpenAIKey:           getEnv("OPENAI_API_KEY", ""),
			AnthropicKey:        getEnv("ANTHROPIC_API_KEY", ""),
			Model:               getEnv("AI_MODEL", "gpt-4o"),
			EmbeddingModel:      getEnv("AI_EMBEDDING_MODEL", "text-embedding-3-small"),
			EmbeddingDimensions: getEnvInt("AI_EMBEDDING_DIMENSIONS", 1536),
		},
		CORS: CORSConfig{
			Origins: splitEnv("CORS_ORIGINS", []string{"http://localhost:5173"}),
		},
	}

	return cfg, nil
}

func (c *DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fmt.Fprintf(os.Stderr, "FATAL: required env var %q is not set\n", key)
		os.Exit(1)
	}
	return v
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		i, err := strconv.Atoi(v)
		if err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return fallback
}

func splitEnv(key string, fallback []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var result []string
	start := 0
	for i := 0; i < len(v); i++ {
		if v[i] == ',' {
			result = append(result, v[start:i])
			start = i + 1
		}
	}
	result = append(result, v[start:])
	return result
}
