package models

import "time"

// Request/Response models for API endpoints
type AuthCodeRequest struct {
	AuthorizationCode string                 `json:"authorization_code"`
	RedirectURI       string                 `json:"redirect_uri,omitempty"`
	WABAID            string                 `json:"waba_id,omitempty"`         // From frontend message event
	PhoneNumberID     string                 `json:"phone_number_id,omitempty"` // From frontend message event
	BusinessID        string                 `json:"business_id,omitempty"`     // From frontend message event
	ClientInfo        map[string]interface{} `json:"client_info,omitempty"`
}

type FacebookTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type FacebookBusinessAccount struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	VerificationStatus string `json:"verification_status"`
	ProfilePictureURI  string `json:"profile_picture_uri"`
}

type FacebookPhoneNumber struct {
	ID                     string `json:"id"`
	DisplayPhoneNumber     string `json:"display_phone_number"`
	VerifiedName           string `json:"verified_name"`
	QualityRating          string `json:"quality_rating"`
	Status                 string `json:"status"`
	CodeVerificationStatus string `json:"code_verification_status"`
}

// Business account storage model
type BusinessAccount struct {
	ID              string                 `json:"id"`
	WABAID          string                 `json:"waba_id"`
	BusinessName    string                 `json:"business_name"`
	PhoneNumbers    []BusinessPhoneNumber  `json:"phone_numbers"`
	AccessToken     string                 `json:"access_token,omitempty"` // Don't send in API responses
	TokenExpiresAt  time.Time              `json:"token_expires_at,omitempty"`
	WebhooksEnabled bool                   `json:"webhooks_enabled"`
	SetupComplete   bool                   `json:"setup_complete"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type BusinessPhoneNumber struct {
	ID            string `json:"id"`
	PhoneNumber   string `json:"phone_number"`
	DisplayName   string `json:"display_name"`
	Status        string `json:"status"`
	QualityRating string `json:"quality_rating"`
	IsVerified    bool   `json:"is_verified"`
}

// API Response models
type BusinessSetupResponse struct {
	Success      bool                   `json:"success"`
	Message      string                 `json:"message,omitempty"`
	Error        string                 `json:"error,omitempty"`
	BusinessInfo *BusinessAccount       `json:"business_info,omitempty"`
	SetupStatus  string                 `json:"setup_status,omitempty"`
	NextSteps    []string               `json:"next_steps,omitempty"`
	TokenInfo    map[string]interface{} `json:"token_info,omitempty"` // Token details for frontend display
}

// Webhook models
type WebhookEvent struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Time    int64  `json:"time"`
		Changes []struct {
			Value struct {
				MessagingProduct string `json:"messaging_product"`
				Metadata         struct {
					DisplayPhoneNumber string `json:"display_phone_number"`
					PhoneNumberID      string `json:"phone_number_id"`
				} `json:"metadata"`
				Messages []struct {
					ID        string `json:"id"`
					From      string `json:"from"`
					Timestamp string `json:"timestamp"`
					Type      string `json:"type"`
					Text      struct {
						Body string `json:"body"`
					} `json:"text,omitempty"`
				} `json:"messages,omitempty"`
				Statuses []struct {
					ID          string `json:"id"`
					RecipientID string `json:"recipient_id"`
					Status      string `json:"status"`
					Timestamp   string `json:"timestamp"`
				} `json:"statuses,omitempty"`
			} `json:"value"`
			Field string `json:"field"`
		} `json:"changes"`
	} `json:"entry"`
}

// WhatsApp Message Templates
type WhatsAppTemplate struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Language     string `json:"language"`
	Status       string `json:"status"`
	Category     string `json:"category"`
	QualityScore struct {
		Score string `json:"score"`
	} `json:"quality_score,omitempty"`
}

type TemplatesResponse struct {
	Success   bool               `json:"success"`
	Error     string             `json:"error,omitempty"`
	Templates []WhatsAppTemplate `json:"templates,omitempty"`
	TokenInfo map[string]any     `json:"token_info,omitempty"`
}
