package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port               string
	LogLevel           string
	AuthServiceURLs    []string
	ComputeServiceURLs []string
	JWTSecret          string
	APIKeys            map[string]string // Key -> Role
	RateLimitRefill    float64           // tokens per second
	RateLimitBurst     int
	CircuitFailureRate float64
	CircuitCooldown    time.Duration
	EnableCaching      bool
	CacheTTL           time.Duration
	RequestTimeout     time.Duration
	MaxRetryAttempts   int
}

func LoadConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "INFO"
	}

	authURLs := getSliceEnv("AUTH_SERVICE_URL", []string{"http://localhost:8081"})
	computeURLs := getSliceEnv("COMPUTE_SERVICE_URL", []string{"http://localhost:8082"})

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "nexuscore-enterprise-cryptographic-master-key-2026"
	}

	// Parse configured API Keys: "key1:Admin,key2:Operator"
	apiKeys := make(map[string]string)
	apiKeysStr := os.Getenv("GATEWAY_API_KEYS")
	if apiKeysStr == "" {
		apiKeysStr = "nexus-admin-api-key:Admin,nexus-operator-api-key:Operator,nexus-read-api-key:Guest"
	}
	for _, pair := range strings.Split(apiKeysStr, ",") {
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) == 2 {
			apiKeys[parts[0]] = parts[1]
		}
	}

	refill, _ := strconv.ParseFloat(getEnv("RATE_LIMIT_REFILL", "10.0"), 64)
	burst, _ := strconv.Atoi(getEnv("RATE_LIMIT_BURST", "100"))
	failRate, _ := strconv.ParseFloat(getEnv("CIRCUIT_FAILURE_RATE", "0.5"), 64)
	cooldown, _ := time.ParseDuration(getEnv("CIRCUIT_COOLDOWN", "5s"))
	enableCache := getEnv("ENABLE_CACHING", "true") == "true"
	cacheTTL, _ := time.ParseDuration(getEnv("CACHE_TTL", "30s"))
	reqTimeout, _ := time.ParseDuration(getEnv("REQUEST_TIMEOUT", "10s"))
	retries, _ := strconv.Atoi(getEnv("MAX_RETRY_ATTEMPTS", "3"))

	return &Config{
		Port:               port,
		LogLevel:           logLevel,
		AuthServiceURLs:    authURLs,
		ComputeServiceURLs: computeURLs,
		JWTSecret:          jwtSecret,
		APIKeys:            apiKeys,
		RateLimitRefill:    refill,
		RateLimitBurst:     burst,
		CircuitFailureRate: failRate,
		CircuitCooldown:    cooldown,
		EnableCaching:      enableCache,
		CacheTTL:           cacheTTL,
		RequestTimeout:     reqTimeout,
		MaxRetryAttempts:   retries,
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getSliceEnv(key string, fallback []string) []string {
	if val := os.Getenv(key); val != "" {
		parts := strings.Split(val, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return parts
	}
	return fallback
}
