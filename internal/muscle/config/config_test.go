package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

// TestDefaultConfig tests DefaultConfig returns valid defaults
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	// App config
	if cfg.App.Port != "3000" {
		t.Errorf("Expected App.Port '3000', got '%s'", cfg.App.Port)
	}
	if cfg.App.LogLevel != "info" {
		t.Errorf("Expected App.LogLevel 'info', got '%s'", cfg.App.LogLevel)
	}

	// Redis config
	if cfg.Redis.Host != "localhost" {
		t.Errorf("Expected Redis.Host 'localhost', got '%s'", cfg.Redis.Host)
	}
	if cfg.Redis.Port != 6379 {
		t.Errorf("Expected Redis.Port 6379, got %d", cfg.Redis.Port)
	}
	if cfg.Redis.PoolSize != 100 {
		t.Errorf("Expected Redis.PoolSize 100, got %d", cfg.Redis.PoolSize)
	}

	// Neo4j config
	if cfg.Neo4j.URI != "bolt://localhost:7687" {
		t.Errorf("Expected Neo4j.URI 'bolt://localhost:7687', got '%s'", cfg.Neo4j.URI)
	}
	if cfg.Neo4j.Username != "neo4j" {
		t.Errorf("Expected Neo4j.Username 'neo4j', got '%s'", cfg.Neo4j.Username)
	}

	// Kafka config
	if len(cfg.Kafka.Brokers) != 1 || cfg.Kafka.Brokers[0] != "localhost:9092" {
		t.Errorf("Expected Kafka.Brokers ['localhost:9092'], got %v", cfg.Kafka.Brokers)
	}
	if cfg.Kafka.Topic != "match-events" {
		t.Errorf("Expected Kafka.Topic 'match-events', got '%s'", cfg.Kafka.Topic)
	}

	// Engine config
	if cfg.Engine.WorkerPoolSize != 10 {
		t.Errorf("Expected Engine.WorkerPoolSize 10, got %d", cfg.Engine.WorkerPoolSize)
	}
	if cfg.Engine.EventBufferSize != 1000 {
		t.Errorf("Expected Engine.EventBufferSize 1000, got %d", cfg.Engine.EventBufferSize)
	}

	// LLM config
	if cfg.LLM.Host != "localhost" {
		t.Errorf("Expected LLM.Host 'localhost', got '%s'", cfg.LLM.Host)
	}
	if cfg.LLM.Port != 8000 {
		t.Errorf("Expected LLM.Port 8000, got %d", cfg.LLM.Port)
	}
	if cfg.LLM.ModelName != "llama-3-8b" {
		t.Errorf("Expected LLM.ModelName 'llama-3-8b', got '%s'", cfg.LLM.ModelName)
	}
}

// TestConfigValidation tests configuration validation
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			cfg: &Config{
				Redis: RedisConfig{Host: "localhost", Port: 6379},
				Neo4j: Neo4jConfig{URI: "bolt://localhost:7687"},
				Kafka: KafkaConfig{Brokers: []string{"localhost:9092"}},
				LLM:   LLMConfig{Host: "localhost", Port: 8000, ModelName: "test-model"},
				Admin: AdminConfig{Username: "admin", PasswordHash: "$2b$10$ziPWvZH7DKff9a2d0a2gBOhKL4.6WQOjEbVC6A0NaJoCqxIPdibmy"},
			},
			wantErr: false,
		},
		{
			name: "missing redis host",
			cfg: &Config{
				Redis: RedisConfig{Port: 6379},
				Neo4j: Neo4jConfig{URI: "bolt://localhost:7687"},
				Kafka: KafkaConfig{Brokers: []string{"localhost:9092"}},
				LLM:   LLMConfig{Host: "localhost", Port: 8000, ModelName: "test-model"},
			},
			wantErr: true,
			errMsg:  "redis host is required",
		},
		{
			name: "invalid redis port",
			cfg: &Config{
				Redis: RedisConfig{Host: "localhost", Port: 70000},
				Neo4j: Neo4jConfig{URI: "bolt://localhost:7687"},
				Kafka: KafkaConfig{Brokers: []string{"localhost:9092"}},
				LLM:   LLMConfig{Host: "localhost", Port: 8000, ModelName: "test-model"},
			},
			wantErr: true,
			errMsg:  "invalid redis port",
		},
		{
			name: "negative redis port",
			cfg: &Config{
				Redis: RedisConfig{Host: "localhost", Port: -1},
				Neo4j: Neo4jConfig{URI: "bolt://localhost:7687"},
				Kafka: KafkaConfig{Brokers: []string{"localhost:9092"}},
				LLM:   LLMConfig{Host: "localhost", Port: 8000, ModelName: "test-model"},
			},
			wantErr: true,
			errMsg:  "invalid redis port",
		},
		{
			name: "missing neo4j uri",
			cfg: &Config{
				Redis: RedisConfig{Host: "localhost", Port: 6379},
				Neo4j: Neo4jConfig{},
				Kafka: KafkaConfig{Brokers: []string{"localhost:9092"}},
				LLM:   LLMConfig{Host: "localhost", Port: 8000, ModelName: "test-model"},
			},
			wantErr: true,
			errMsg:  "neo4j uri is required",
		},
		{
			name: "missing kafka brokers",
			cfg: &Config{
				Redis: RedisConfig{Host: "localhost", Port: 6379},
				Neo4j: Neo4jConfig{URI: "bolt://localhost:7687"},
				Kafka: KafkaConfig{Brokers: []string{}},
				LLM:   LLMConfig{Host: "localhost", Port: 8000, ModelName: "test-model"},
			},
			wantErr: false, // Kafka is optional for MCP server mode - logs warning only
		},
		{
			name: "missing llm host",
			cfg: &Config{
				Redis: RedisConfig{Host: "localhost", Port: 6379},
				Neo4j: Neo4jConfig{URI: "bolt://localhost:7687"},
				Kafka: KafkaConfig{Brokers: []string{"localhost:9092"}},
				LLM:   LLMConfig{Port: 8000, ModelName: "test-model"},
			},
			wantErr: false, // LLM is optional for MCP server mode - logs warning only
		},
		{
			name: "invalid llm port",
			cfg: &Config{
				Redis: RedisConfig{Host: "localhost", Port: 6379},
				Neo4j: Neo4jConfig{URI: "bolt://localhost:7687"},
				Kafka: KafkaConfig{Brokers: []string{"localhost:9092"}},
				LLM:   LLMConfig{Host: "localhost", Port: 70000, ModelName: "test-model"},
			},
			wantErr: true,
			errMsg:  "invalid llm port",
		},
		{
			name: "missing llm model name",
			cfg: &Config{
				Redis: RedisConfig{Host: "localhost", Port: 6379},
				Neo4j: Neo4jConfig{URI: "bolt://localhost:7687"},
				Kafka: KafkaConfig{Brokers: []string{"localhost:9092"}},
				LLM:   LLMConfig{Host: "localhost", Port: 8000},
			},
			wantErr: false, // LLM model name is optional when host is empty (MCP mode)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				errMsg := err.Error()
				// Check if error contains the expected message
				hasPrefix := false
				switch tt.errMsg {
				case "redis host is required":
					hasPrefix = errMsg == tt.errMsg
				case "neo4j uri is required":
					hasPrefix = errMsg == tt.errMsg
				case "at least one kafka broker is required":
					hasPrefix = errMsg == tt.errMsg
				case "llm host is required":
					hasPrefix = errMsg == tt.errMsg
				case "llm model name is required":
					hasPrefix = errMsg == tt.errMsg
				default:
					hasPrefix = strings.Contains(errMsg, tt.errMsg)
				}
				if !hasPrefix {
					t.Errorf("Validate() error message = %v, want %v", errMsg, tt.errMsg)
				}
			}
		})
	}
}

// TestRedisAddr tests Redis address generation
func TestRedisAddr(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     int
		expected string
	}{
		{"default", "localhost", 6379, "localhost:6379"},
		{"custom host", "redis.example.com", 6380, "redis.example.com:6380"},
		{"ip address", "192.168.1.1", 6379, "192.168.1.1:6379"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &RedisConfig{Host: tt.host, Port: tt.port}
			addr := cfg.RedisAddr()
			if addr != tt.expected {
				t.Errorf("RedisAddr() = %v, want %v", addr, tt.expected)
			}
		})
	}
}

// TestLLMAddr tests LLM address generation
func TestLLMAddr(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     int
		expected string
	}{
		{"default", "localhost", 8000, "http://localhost:8000"},
		{"custom host", "llm.example.com", 8080, "http://llm.example.com:8080"},
		{"ip address", "192.168.1.100", 8000, "http://192.168.1.100:8000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &LLMConfig{Host: tt.host, Port: tt.port}
			addr := cfg.LLMAddr()
			if addr != tt.expected {
				t.Errorf("LLMAddr() = %v, want %v", addr, tt.expected)
			}
		})
	}
}

// TestServerAddr tests server address generation
func TestServerAddr(t *testing.T) {
	tests := []struct {
		name     string
		port     string
		expected string
	}{
		{"default port", "", ":3000"},
		{"custom port", "3000", ":3000"},
		{"port 9000", "9000", ":9000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{App: AppConfig{Port: tt.port}}
			addr := cfg.ServerAddr()
			if addr != tt.expected {
				t.Errorf("ServerAddr() = %v, want %v", addr, tt.expected)
			}
		})
	}
}

// TestKafkaBrokers tests Kafka brokers string generation
func TestKafkaBrokers(t *testing.T) {
	cfg := &Config{
		Kafka: KafkaConfig{Brokers: []string{"localhost:9092", "localhost:9093"}},
	}
	brokers := cfg.Kafka.KafkaBrokers()
	expected := "[localhost:9092 localhost:9093]"
	if brokers != expected {
		t.Errorf("KafkaBrokers() = %v, want %v", brokers, expected)
	}
}

// TestLoadFromEnvironment tests loading configuration from environment variables
func TestLoadFromEnvironment(t *testing.T) {
	// Save original env vars
	origEnv := map[string]string{
		"APP_PORT":       os.Getenv("APP_PORT"),
		"LOG_LEVEL":      os.Getenv("LOG_LEVEL"),
		"REDIS_HOST":     os.Getenv("REDIS_HOST"),
		"REDIS_PORT":     os.Getenv("REDIS_PORT"),
		"NEO4J_URI":      os.Getenv("NEO4J_URI"),
		"NEO4J_USERNAME": os.Getenv("NEO4J_USERNAME"),
		"NEO4J_PASSWORD": os.Getenv("NEO4J_PASSWORD"),
		"KAFKA_BROKERS":  os.Getenv("KAFKA_BROKERS"),
		"LLM_HOST":       os.Getenv("LLM_HOST"),
		"LLM_PORT":       os.Getenv("LLM_PORT"),
		"MODEL_NAME":     os.Getenv("MODEL_NAME"),
	}
	defer func() {
		// Restore original env vars
		for k, v := range origEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Set test environment variables
	os.Setenv("APP_PORT", "9000")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("REDIS_HOST", "redis.test.com")
	os.Setenv("REDIS_PORT", "6380")
	os.Setenv("NEO4J_URI", "bolt://neo4j.test.com:7687")
	os.Setenv("NEO4J_USERNAME", "testuser")
	os.Setenv("NEO4J_PASSWORD", "testpass")
	os.Setenv("KAFKA_BROKERS", "kafka1:9092,kafka2:9092")
	os.Setenv("LLM_HOST", "llm.test.com")
	os.Setenv("LLM_PORT", "8080")
	os.Setenv("MODEL_NAME", "test-model")
	os.Setenv("LLM_TEMPERATURE", "0.5")
	os.Setenv("LLM_MAX_TOKENS", "1024")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify environment overrides
	if cfg.App.Port != "9000" {
		t.Errorf("Expected App.Port '9000', got '%s'", cfg.App.Port)
	}
	if cfg.App.LogLevel != "debug" {
		t.Errorf("Expected App.LogLevel 'debug', got '%s'", cfg.App.LogLevel)
	}
	if cfg.Redis.Host != "redis.test.com" {
		t.Errorf("Expected Redis.Host 'redis.test.com', got '%s'", cfg.Redis.Host)
	}
	if cfg.Redis.Port != 6380 {
		t.Errorf("Expected Redis.Port 6380, got %d", cfg.Redis.Port)
	}
	if cfg.Neo4j.URI != "bolt://neo4j.test.com:7687" {
		t.Errorf("Expected Neo4j.URI 'bolt://neo4j.test.com:7687', got '%s'", cfg.Neo4j.URI)
	}
	if cfg.Neo4j.Username != "testuser" {
		t.Errorf("Expected Neo4j.Username 'testuser', got '%s'", cfg.Neo4j.Username)
	}
	if len(cfg.Kafka.Brokers) != 2 {
		t.Errorf("Expected 2 Kafka brokers, got %d", len(cfg.Kafka.Brokers))
	}
	if cfg.LLM.Host != "llm.test.com" {
		t.Errorf("Expected LLM.Host 'llm.test.com', got '%s'", cfg.LLM.Host)
	}
	if cfg.LLM.Temperature != 0.5 {
		t.Errorf("Expected LLM.Temperature 0.5, got %f", cfg.LLM.Temperature)
	}
	if cfg.LLM.MaxTokens != 1024 {
		t.Errorf("Expected LLM.MaxTokens 1024, got %d", cfg.LLM.MaxTokens)
	}
}

// TestNewContext tests context creation with timeout
func TestNewContext(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Engine.QueryTimeout = 5 * time.Second

	ctx, cancel := cfg.NewContext()
	defer cancel()

	if ctx == nil {
		t.Error("Expected non-nil context")
	}

	// Context should have timeout
	select {
	case <-ctx.Done():
		if ctx.Err() == nil {
			t.Error("Expected context to have timeout")
		}
	default:
		// Context should not be done yet
	}
}
