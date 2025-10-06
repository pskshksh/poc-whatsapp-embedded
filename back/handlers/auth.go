package handlers

import (
	"back/models"
	"back/services"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type AuthHandler struct {
	facebook *services.FacebookService
	whatsapp *services.WhatsAppService
	storage  *services.StorageService
}

func NewAuthHandler(facebook *services.FacebookService, whatsapp *services.WhatsAppService, storage *services.StorageService) *AuthHandler {
	return &AuthHandler{
		facebook: facebook,
		whatsapp: whatsapp,
		storage:  storage,
	}
}

func (h *AuthHandler) HandleEmbeddedSignup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.AuthCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AuthorizationCode == "" {
		h.sendError(w, "Authorization code is required", http.StatusBadRequest)
		return
	}

	// IMPORTANT: Use the redirect_uri from the request if provided, otherwise use config default
	redirectURI := req.RedirectURI
	if redirectURI == "" {
		// For WhatsApp Embedded Signup, we need to use the configured redirect URI
		// This should match exactly what's in your Facebook app settings
		redirectURI = "https://c6c766eee161.ngrok-free.app" // Use your actual ngrok URL
	}

	// Step 1: Exchange authorization code for access token
	log.Printf("Step 1: Exchanging authorization code for access token with redirect_uri=%q", redirectURI)
	tokenResp, err := h.facebook.ExchangeToken(req.AuthorizationCode, redirectURI)
	if err != nil {
		h.sendError(w, fmt.Sprintf("Token exchange failed: %v", err), http.StatusBadRequest)
		return
	}

	// Log token details (remove in production)
	log.Printf("✅ Token exchange successful!")
	log.Printf("📄 Token details: AccessToken length=%d, TokenType=%s, ExpiresIn=%d",
		len(tokenResp.AccessToken), tokenResp.TokenType, tokenResp.ExpiresIn)
	log.Printf("🔑 Access Token (first 20 chars): %s...", tokenResp.AccessToken[:20])

	// Step 2: Get business accounts
	log.Printf("Step 2: Fetching business accounts")
	businesses, err := h.facebook.GetBusinessAccounts(tokenResp.AccessToken)
	if err != nil {
		h.sendError(w, fmt.Sprintf("Failed to fetch business accounts: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("📊 Found %d business accounts", len(businesses))
	for i, business := range businesses {
		log.Printf("  Business %d: ID=%s, Name=%s, VerificationStatus=%s",
			i+1, business.ID, business.Name, business.VerificationStatus)
	}

	if len(businesses) == 0 {
		log.Printf("⚠️ No business accounts found via standard endpoint")
		log.Printf("ℹ️ This is normal for WhatsApp Embedded Signup - WABA is created but may not appear in /me/businesses immediately")

		// For embedded signup, we should get the WABA ID from the frontend message event
		// Check if the frontend provided WABA details
		if req.WABAID != "" {
			log.Printf("✅ Using WABA ID from frontend message event: %s", req.WABAID)

			// Create a virtual business account with the provided WABA ID
			businesses = []models.FacebookBusinessAccount{
				{
					ID:                 req.WABAID,
					Name:               "WhatsApp Business Account (Embedded Signup)",
					VerificationStatus: "pending",
				},
			}
		} else {
			h.sendError(w, "No WhatsApp Business Accounts found and no WABA ID provided. Please ensure you completed the embedded signup flow and check browser console for the message event with WABA details.", http.StatusNotFound)
			return
		}
	}

	// Step 3: Process first business account
	log.Printf("Step 3: Processing business account: %s", businesses[0].ID)
	business := businesses[0]

	// Step 4: Get phone numbers for the business
	log.Printf("Step 4: Fetching phone numbers for WABA: %s", business.ID)
	phoneNumbers, err := h.facebook.GetPhoneNumbers(tokenResp.AccessToken, business.ID)
	if err != nil {
		h.sendError(w, fmt.Sprintf("Failed to fetch phone numbers: %v", err), http.StatusInternalServerError)
		return
	}

	if len(phoneNumbers) == 0 {
		h.sendError(w, "No phone numbers found for this business account", http.StatusNotFound)
		return
	}

	// Step 5: Setup webhooks
	log.Printf("Step 5: Setting up webhooks for WABA: %s", business.ID)
	webhooksEnabled := true
	if err := h.whatsapp.SetupWebhooks(tokenResp.AccessToken, business.ID); err != nil {
		log.Printf("Warning: Failed to setup webhooks: %v", err)
		webhooksEnabled = false
	}

	// Step 6: Get business profile information
	log.Printf("Step 6: Fetching business profile")
	profile, err := h.whatsapp.GetBusinessProfile(tokenResp.AccessToken, phoneNumbers[0].ID)
	if err != nil {
		log.Printf("Warning: Failed to get business profile: %v", err)
		profile = map[string]interface{}{}
	}

	// Step 7: Create business account record
	log.Printf("Step 7: Creating business account record")
	businessPhoneNumbers := make([]models.BusinessPhoneNumber, 0, len(phoneNumbers))
	for _, phone := range phoneNumbers {
		businessPhoneNumbers = append(businessPhoneNumbers, models.BusinessPhoneNumber{
			ID:            phone.ID,
			PhoneNumber:   phone.DisplayPhoneNumber,
			DisplayName:   phone.VerifiedName,
			Status:        phone.Status,
			QualityRating: phone.QualityRating,
			IsVerified:    phone.CodeVerificationStatus == "VERIFIED",
		})
	}

	account := &models.BusinessAccount{
		ID:              fmt.Sprintf("ba_%d", time.Now().Unix()),
		WABAID:          business.ID,
		BusinessName:    business.Name,
		PhoneNumbers:    businessPhoneNumbers,
		AccessToken:     tokenResp.AccessToken, // Encrypt in production
		TokenExpiresAt:  time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		WebhooksEnabled: webhooksEnabled,
		SetupComplete:   true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		Metadata: map[string]interface{}{
			"verification_status": business.VerificationStatus,
			"profile_info":        profile,
			"setup_source":        "embedded_signup",
			"redirect_uri":        redirectURI, // Store the redirect URI used
		},
	}

	// Step 8: Save to storage
	log.Printf("Step 8: Saving business account to storage")
	if err := h.storage.SaveBusinessAccount(account); err != nil {
		h.sendError(w, fmt.Sprintf("Failed to save business account: %v", err), http.StatusInternalServerError)
		return
	}

	// Step 9: Prepare next steps
	nextSteps := []string{
		"Business account is ready to send messages",
		"Configure message templates in WhatsApp Manager",
		"Test messaging functionality",
	}

	if !webhooksEnabled {
		nextSteps = append(nextSteps, "Manual webhook configuration may be required")
	}

	// Step 10: Send success response with token details
	log.Printf("Step 10: Setup completed successfully for WABA: %s", business.ID)
	response := models.BusinessSetupResponse{
		Success:      true,
		Message:      "WhatsApp Business Account setup completed successfully",
		BusinessInfo: account,
		SetupStatus:  "complete",
		NextSteps:    nextSteps,
		TokenInfo: map[string]interface{}{
			"access_token_length":  len(tokenResp.AccessToken),
			"access_token_preview": tokenResp.AccessToken,
			"full_access_token":    tokenResp.AccessToken, // Full token for debugging
			"token_type":           tokenResp.TokenType,
			"expires_in":           tokenResp.ExpiresIn,
			"token_created_at":     time.Now().Format(time.RFC3339),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(models.BusinessSetupResponse{
		Success: false,
		Error:   message,
	})
}
