package services

import (
	"back/config"
	"back/models"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type WhatsAppService struct {
	config   *config.Config
	client   *http.Client
	facebook *FacebookService
}

func NewWhatsAppService(cfg *config.Config, facebook *FacebookService) *WhatsAppService {
	return &WhatsAppService{
		config:   cfg,
		client:   &http.Client{},
		facebook: facebook,
	}
}

// Setup webhooks for a WABA
func (w *WhatsAppService) SetupWebhooks(accessToken, wabaID string) error {
	if w.config.WebhookCallbackURL == "" {
		return fmt.Errorf("webhook callback URL not configured")
	}

	url := fmt.Sprintf("https://graph.facebook.com/v23.0/%s/subscribed_apps", wabaID)

	payload := map[string]interface{}{
		"subscribed_fields": []string{
			"messages",
			"message_deliveries",
			"message_reads",
			"message_echoes",
			"message_template_status_update",
			"account_alerts",
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook setup request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook setup failed with status %d", resp.StatusCode)
	}

	return nil
}

// Get business profile information
func (w *WhatsAppService) GetBusinessProfile(accessToken, phoneNumberID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://graph.facebook.com/v23.0/%s/whatsapp_business_profile", phoneNumberID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("business profile request failed: %w", err)
	}
	defer resp.Body.Close()

	var profile struct {
		Data []map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to decode business profile: %w", err)
	}

	if len(profile.Data) == 0 {
		return map[string]interface{}{}, nil
	}

	return profile.Data[0], nil
}

// Send a test message (for verification)
func (w *WhatsAppService) SendTestMessage(accessToken, phoneNumberID, recipientNumber, message string) error {
	url := fmt.Sprintf("https://graph.facebook.com/v23.0/%s/messages", phoneNumberID)

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                recipientNumber,
		"type":              "text",
		"text": map[string]string{
			"body": message,
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("send message request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("send message failed with status %d", resp.StatusCode)
	}

	return nil
}

// ListTemplates fetches all message templates for a WABA.
func (w *WhatsAppService) ListTemplates(accessToken, wabaID string) ([]models.WhatsAppTemplate, error) {
	base := fmt.Sprintf("https://graph.facebook.com/v23.0/%s/message_templates", wabaID)
	fields := "id,name,language,status,category,quality_score"
	url := fmt.Sprintf("%s?fields=%s&limit=100", base, fields)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("build templates request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	var out []models.WhatsAppTemplate
	for {
		resp, err := w.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("templates request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("templates failed (%s)", resp.Status)
		}

		var body struct {
			Data   []models.WhatsAppTemplate `json:"data"`
			Paging struct {
				Next string `json:"next"`
			} `json:"paging"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return nil, fmt.Errorf("decode templates: %w", err)
		}
		out = append(out, body.Data...)

		if body.Paging.Next == "" {
			break
		}
		// follow next page
		req, err = http.NewRequest("GET", body.Paging.Next, nil)
		if err != nil {
			return nil, fmt.Errorf("build next page request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	return out, nil
}
