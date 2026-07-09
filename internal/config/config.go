package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// DatabaseURL is the PostgreSQL connection string (required).
	DatabaseURL string

	// JWTSecret is used to sign access tokens (required).
	JWTSecret string

	// PosAPIKey authorizes POS webhook requests (required).
	PosAPIKey string

	// AnalyticsAPIKey authorizes analytics-service callbacks (defaults to PosAPIKey).
	AnalyticsAPIKey string

	// ServerPort is the HTTP server listen port (default: 8080).
	ServerPort string

	// PollIntervalCvMs is the polling interval for cv_events and speech_transcripts in milliseconds (default: 500).
	PollIntervalCvMs int

	// PollIntervalTasksMs is the polling interval for completed tasks in milliseconds (default: 2000).
	PollIntervalTasksMs int

	// ConfidenceThreshold is the minimum aggregate confidence to send an alert to operators (default: 0.75).
	ConfidenceThreshold float64

	// MaxDBConns is the maximum number of connections in the pgxpool (default: 20).
	MaxDBConns int32

	// AccessTokenTTLMinutes controls access token lifetime (default: 480).
	AccessTokenTTLMinutes int

	// BootstrapAdminUsername creates the first admin user if it does not exist.
	BootstrapAdminUsername string

	// BootstrapAdminPassword is used only to create the bootstrap admin user.
	BootstrapAdminPassword string
}

// Load reads configuration from environment variables.
// Returns an error if required variables are missing.
func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}

	posAPIKey := os.Getenv("POS_API_KEY")
	if posAPIKey == "" {
		return nil, fmt.Errorf("POS_API_KEY environment variable is required")
	}

	cfg := &Config{
		DatabaseURL:            dbURL,
		JWTSecret:              jwtSecret,
		PosAPIKey:              posAPIKey,
		AnalyticsAPIKey:        getEnvOrDefault("ANALYTICS_API_KEY", posAPIKey),
		ServerPort:             getEnvOrDefault("SERVER_PORT", "8080"),
		PollIntervalCvMs:       getEnvIntOrDefault("POLL_INTERVAL_CV_MS", 500),
		PollIntervalTasksMs:    getEnvIntOrDefault("POLL_INTERVAL_TASKS_MS", 2000),
		ConfidenceThreshold:    getEnvFloatOrDefault("CONFIDENCE_THRESHOLD", 0.75),
		MaxDBConns:             int32(getEnvIntOrDefault("MAX_DB_CONNS", 20)),
		AccessTokenTTLMinutes:  getEnvIntOrDefault("ACCESS_TOKEN_TTL_MINUTES", 480),
		BootstrapAdminUsername: os.Getenv("BOOTSTRAP_ADMIN_USERNAME"),
		BootstrapAdminPassword: os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvIntOrDefault(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return parsed
}

func getEnvFloatOrDefault(key string, defaultVal float64) float64 {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	parsed, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return defaultVal
	}
	return parsed
}
