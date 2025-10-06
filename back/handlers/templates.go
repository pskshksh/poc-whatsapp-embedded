package handlers

import (
	"back/models"
	"back/services"
	"encoding/json"
	"net/http"
	"time"
)

type TemplatesHandler struct {
	facebook *services.FacebookService
	whatsapp *services.WhatsAppService
}

func NewTemplatesHandler(fb *services.FacebookService, wa *services.WhatsAppService) *TemplatesHandler {
	return &TemplatesHandler{facebook: fb, whatsapp: wa}
}

// POST /api/whatsapp/templates
// Body: { authorization_code, redirect_uri?, waba_id (required) }
func (h *TemplatesHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.AuthCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeTemplatesError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.AuthorizationCode == "" {
		writeTemplatesError(w, "Authorization code is required", http.StatusBadRequest)
		return
	}
	if req.WABAID == "" {
		writeTemplatesError(w, "WABA ID is required", http.StatusBadRequest)
		return
	}

	// 1) Exchange code -> token
	redirectURI := req.RedirectURI // may be empty (supports embedded signup)
	tokenResp, err := h.facebook.ExchangeToken(req.AuthorizationCode, redirectURI)
	if err != nil {
		writeTemplatesError(w, "Token exchange failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 2) Fetch templates (only)
	templates, err := h.whatsapp.ListTemplates(tokenResp.AccessToken, req.WABAID)
	if err != nil {
		writeTemplatesError(w, "Failed to fetch templates: "+err.Error(), http.StatusBadGateway)
		return
	}

	// 3) Return lean payload (token preview + templates only)
	resp := models.TemplatesResponse{
		Success:   true,
		Templates: templates,
		TokenInfo: map[string]any{
			"token_type":           tokenResp.TokenType,
			"access_token_length":  len(tokenResp.AccessToken),
			"access_token_preview": tokenResp.AccessToken,
			"full_access_token":    tokenResp.AccessToken, // keep if you want to show it
			"expires_in":           tokenResp.ExpiresIn,
			"token_created_at":     time.Now().Format(time.RFC3339),
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func writeTemplatesError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(models.TemplatesResponse{
		Success: false,
		Error:   msg,
	})
}
