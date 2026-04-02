package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the Muscle layer
type Config struct {
	App    AppConfig
	Redis  RedisConfig
	Neo4j  Neo4jConfig
	Kafka  KafkaConfig
	Engine EngineConfig
	LLM    LLMConfig
	JWT    JWTConfig
	Admin  AdminConfig
}

// AppConfig holds application-level settings
type AppConfig struct {
	Port      string
	WSPort    string
	LogLevel  string
	Transport string
}

// RedisConfig holds Redis connection settings
type RedisConfig struct {
	Host         string
	Port         int
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolTimeout  time.Duration
}

// Neo4jConfig holds Neo4j connection settings
type Neo4jConfig struct {
	URI            string
	Username       string
	Password       string
	Database       string
	MaxConnPool    int
	MaxConnLife    time.Duration
	RequestTimeout time.Duration
}

// KafkaConfig holds Kafka consumer settings
type KafkaConfig struct {
	Brokers        []string
	GroupID        string
	Topic          string
	MinBytes       int
	MaxBytes       int
	MaxWait        time.Duration
	StartOffset    string
	CommitInterval time.Duration
}

// EngineConfig holds rule engine settings
type EngineConfig struct {
	WorkerPoolSize  int
	EventBufferSize int
	QueryTimeout    time.Duration
	RedisScanCount  int
	EnableMetrics   bool
}

// LLMConfig holds vLLM LLM connection settings
type LLMConfig struct {
	Host        string
	Port        int
	ModelName   string
	Temperature float64
	TopP        float64
	MaxTokens   int
	Timeout     time.Duration
	MaxRetries  int
	RetryDelay  time.Duration
}

// JWTConfig holds JWT authentication settings
type JWTConfig struct {
	SecretKey          string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	Issuer             string
}

// AdminConfig holds admin user credentials
type AdminConfig struct {
	Username     string
	PasswordHash string
}

// ErrInvalidCredentials is returned when admin credentials are invalid
var ErrInvalidCredentials = fmt.Errorf("invalid credentials")

// LLMAddr returns the LLM API address
func (l *LLMConfig) LLMAddr() string {
	return fmt.Sprintf("http://%s:%d", l.Host, l.Port)
}

// ServerAddr returns the API server address
func (c *Config) ServerAddr() string {
	if c.App.Port == "" {
		return ":3000"
	}
	return ":" + c.App.Port
}

// WebSocketAddr returns the WebSocket server address
func (c *Config) WebSocketAddr() string {
	if c.App.WSPort == "" {
		return ":3001"
	}
	return ":" + c.App.WSPort
}

// RedisAddr returns the Redis address
func (r *RedisConfig) RedisAddr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// KafkaBrokers returns the Kafka brokers as comma-separated string
func (k *KafkaConfig) KafkaBrokers() string {
	return fmt.Sprintf("%s", k.Brokers)
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	cfg := &Config{
		App: AppConfig{
			Port:     "3000",
			WSPort:   "3001",
			LogLevel: "info",
		},
		Redis: RedisConfig{
			Host:         "localhost",
			Port:         6379,
			Password:     "",
			DB:           0,
			PoolSize:     100,
			MinIdleConns: 10,
			MaxRetries:   3,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
			PoolTimeout:  3 * time.Second,
		},
		Neo4j: Neo4jConfig{
			URI:            "bolt://localhost:7687",
			Username:       "neo4j",
			Password:       "password",
			Database:       "neo4j",
			MaxConnPool:    100,
			MaxConnLife:    30 * time.Minute,
			RequestTimeout: 10 * time.Second,
		},
		Kafka: KafkaConfig{
			Brokers:        []string{"localhost:9092"},
			GroupID:        "muscle-rule-engine",
			Topic:          "match-events",
			MinBytes:       1e3,
			MaxBytes:       10e6,
			MaxWait:        1 * time.Second,
			StartOffset:    "earliest",
			CommitInterval: 1 * time.Second,
		},
		Engine: EngineConfig{
			WorkerPoolSize:  10,
			EventBufferSize: 1000,
			QueryTimeout:    5 * time.Second,
			RedisScanCount:  100,
			EnableMetrics:   true,
		},
		LLM: LLMConfig{
			Host:        "localhost",
			Port:        8000,
			ModelName:   "llama-3-8b",
			Temperature: 0.1,
			TopP:        0.9,
			MaxTokens:   2048,
			Timeout:     30 * time.Second,
			MaxRetries:  3,
			RetryDelay:  1 * time.Second,
		},
		JWT: JWTConfig{
			SecretKey:          "muscle-gamification-jwt-secret-change-in-production",
			AccessTokenExpiry:  15 * time.Minute,
			RefreshTokenExpiry: 7 * 24 * time.Hour,
			Issuer:             "muscle-gamification",
		},
		Admin: AdminConfig{
			Username:     "admin",
			PasswordHash: "$2a$10$placeholder", // Default hash - override with env
		},
	}
	return cfg
}

// Load reads configuration from environment variables
// It first loads from .env file if present, then overrides with environment variables
func Load() (*Config, error) {
	// Try to load .env file (only fails if file doesn't exist, which is okay)
	_ = godotenv.Load()

	cfg := DefaultConfig()

	// App settings
	if v := os.Getenv("APP_PORT"); v != "" {
		cfg.App.Port = v
	}
	if v := os.Getenv("WS_PORT"); v != "" {
		cfg.App.WSPort = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.App.LogLevel = v
	}
	if v := os.Getenv("TRANSPORT"); v != "" {
		cfg.App.Transport = v
	}

	// Neo4j settings
	if v := os.Getenv("NEO4J_URI"); v != "" {
		cfg.Neo4j.URI = v
	}
	if v := os.Getenv("NEO4J_USERNAME"); v != "" {
		cfg.Neo4j.Username = v
	}
	if v := os.Getenv("NEO4J_PASSWORD"); v != "" {
		cfg.Neo4j.Password = v
	}
	if v := os.Getenv("NEO4J_DATABASE"); v != "" {
		cfg.Neo4j.Database = v
	}

	// Redis settings
	if v := os.Getenv("REDIS_HOST"); v != "" {
		cfg.Redis.Host = v
	}
	if v := os.Getenv("REDIS_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Redis.Port = port
		}
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}

	// Kafka settings
	if v := os.Getenv("KAFKA_BROKERS"); v != "" {
		cfg.Kafka.Brokers = strings.Split(v, ",")
	}
	if v := os.Getenv("KAFKA_TOPIC"); v != "" {
		cfg.Kafka.Topic = v
	}
	if v := os.Getenv("KAFKA_GROUP_ID"); v != "" {
		cfg.Kafka.GroupID = v
	}

	// LLM settings
	if v := os.Getenv("LLM_HOST"); v != "" {
		cfg.LLM.Host = v
	}
	if v := os.Getenv("LLM_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.LLM.Port = port
		}
	}
	if v := os.Getenv("MODEL_NAME"); v != "" {
		cfg.LLM.ModelName = v
	}
	if v := os.Getenv("LLM_TEMPERATURE"); v != "" {
		if temp, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.LLM.Temperature = temp
		}
	}
	if v := os.Getenv("LLM_MAX_TOKENS"); v != "" {
		if tokens, err := strconv.Atoi(v); err == nil {
			cfg.LLM.MaxTokens = tokens
		}
	}

	// JWT settings
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWT.SecretKey = v
	}
	if v := os.Getenv("JWT_ISSUER"); v != "" {
		cfg.JWT.Issuer = v
	}
	if v := os.Getenv("JWT_ACCESS_EXPIRY"); v != "" {
		if duration, err := time.ParseDuration(v); err == nil {
			cfg.JWT.AccessTokenExpiry = duration
		}
	}
	if v := os.Getenv("JWT_REFRESH_EXPIRY"); v != "" {
		if duration, err := time.ParseDuration(v); err == nil {
			cfg.JWT.RefreshTokenExpiry = duration
		}
	}

	// Admin credentials
	if v := os.Getenv("ADMIN_USERNAME"); v != "" {
		cfg.Admin.Username = v
	}
	if v := os.Getenv("ADMIN_PASSWORD"); v != "" {
		cfg.Admin.PasswordHash = v
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.App.Port == "" {
		c.App.Port = "3000"
	}
	if c.App.WSPort == "" {
		c.App.WSPort = "3001"
	}
	if c.Redis.Host == "" {
		return fmt.Errorf("redis host is required")
	}
	if c.Redis.Port <= 0 || c.Redis.Port > 65535 {
		return fmt.Errorf("invalid redis port: %d", c.Redis.Port)
	}
	if c.Neo4j.URI == "" {
		return fmt.Errorf("neo4j uri is required")
	}

	// Kafka and LLM are optional for MCP server mode
	// Only validate if explicitly configured
	if len(c.Kafka.Brokers) == 0 {
		log.Println("Warning: Kafka not configured, event processing will be disabled")
	}
	if c.LLM.Host != "" {
		// LLM host is set, validate port and model name if provided
		if c.LLM.Port <= 0 || c.LLM.Port > 65535 {
			return fmt.Errorf("invalid llm port: %d", c.LLM.Port)
		}
		if c.LLM.ModelName == "" {
			log.Println("Warning: LLM model name not specified, using default")
		}
	} else {
		log.Println("Warning: LLM not configured, natural language rules will be disabled")
	}

	if c.JWT.SecretKey == "" {
		c.JWT.SecretKey = "muscle-gamification-jwt-secret-change-in-production"
	}
	if c.JWT.Issuer == "" {
		c.JWT.Issuer = "muscle-gamification"
	}
	if c.Admin.Username == "" {
		c.Admin.Username = "admin"
	}
	if c.Admin.PasswordHash == "" {
		c.Admin.PasswordHash = "$2a$10$placeholder"
	}
	return nil
}

// NewContext returns a context with timeout
func (c *Config) NewContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), c.Engine.QueryTimeout)
}
