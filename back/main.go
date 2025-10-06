package main

import (
	"back/config"
	"back/handlers"
	"back/services"
	"fmt"
	"log"
	"net/http"
)

func enableCORS(next http.HandlerFunc, allowedOrigins []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}

		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize services
	facebookService := services.NewFacebookService(cfg)
	storageService := services.NewStorageService()
	whatsappService := services.NewWhatsAppService(cfg, facebookService)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(facebookService, whatsappService, storageService)
	businessHandler := handlers.NewBusinessHandler(storageService)
	webhookHandler := handlers.NewWebhookHandler(cfg)

	// Routes with CORS
	http.HandleFunc("/health", enableCORS(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","timestamp":"%s"}`,
			fmt.Sprintf("%d", 1234567890))
	}, cfg.AllowedOrigins))

	http.HandleFunc("/api/whatsapp/setup", enableCORS(authHandler.HandleEmbeddedSignup, cfg.AllowedOrigins))
	http.HandleFunc("/api/business/accounts", enableCORS(businessHandler.ListAccounts, cfg.AllowedOrigins))
	http.HandleFunc("/api/business/account", enableCORS(businessHandler.GetAccount, cfg.AllowedOrigins))
	http.HandleFunc("/api/business/export", enableCORS(businessHandler.ExportData, cfg.AllowedOrigins))
	http.HandleFunc("/api/whatsapp/webhooks", enableCORS(webhookHandler.HandleWebhook, cfg.AllowedOrigins))

	// Start server
	fmt.Printf("🚀 WhatsApp Server starting on port %s\n", cfg.ServerPort)
	fmt.Printf("📡 Health check: http://localhost:%s/health\n", cfg.ServerPort)
	fmt.Printf("🔗 Embedded signup: http://localhost:%s/api/whatsapp/setup\n", cfg.ServerPort)
	fmt.Printf("📞 Webhook endpoint: http://localhost:%s/api/whatsapp/webhooks\n", cfg.ServerPort)

	if cfg.WebhookCallbackURL != "" {
		fmt.Printf("🔔 Webhook callback URL: %s\n", cfg.WebhookCallbackURL)
	} else {
		fmt.Printf("⚠️  Webhook callback URL not configured - webhooks will not work\n")
	}

	log.Fatal(http.ListenAndServe(":"+cfg.ServerPort, nil))
}
