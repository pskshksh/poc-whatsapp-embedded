package config

import (
	"log"
	"os"
)

type Config struct {
	FacebookAppID       string
	FacebookAppSecret   string
	FacebookRedirectURI string // <-- NEW: OAuth redirect used in token exchange
	ServerPort          string
	WebhookVerifyToken  string
	WebhookCallbackURL  string // WhatsApp webhook callback (keep separate from OAuth)
	AllowedOrigins      []string
}

func Load() *Config {
	cfg := &Config{
		FacebookAppID:       getEnv("FACEBOOK_APP_ID", "your_facebook_appid_here"),
		FacebookAppSecret:   getEnv("FACEBOOK_APP_SECRET", "your_app_secret_hrer"),
		FacebookRedirectURI: getEnv("FACEBOOK_REDIRECT_URI", "https://482e8d84cfc0.ngrok-free.app"), 
		ServerPort:          getEnv("SERVER_PORT", "8081"),
		WebhookVerifyToken:  getEnv("WEBHOOK_VERIFY_TOKEN", ""),
		WebhookCallbackURL:  getEnv("WEBHOOK_CALLBACK_URL", "https://482e8d84cfc0.ngrok-free.app/api/whatsapp/webhooks"),
		AllowedOrigins:      []string{getEnv("CLIENT_URL", "http://localhost:3001"), "https://482e8d84cfc0.ngrok-free.app"}, // ← Updated port
	}

	if cfg.FacebookAppID == "" || cfg.FacebookAppSecret == "" {
		log.Fatal("FACEBOOK_APP_ID and FACEBOOK_APP_SECRET are required")
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
