package handlers

import (
	"back/config"
	"back/models"
	"encoding/json"
	"io"
	"log"
	"net/http"
)

type WebhookHandler struct {
	config *config.Config
}

func NewWebhookHandler(cfg *config.Config) *WebhookHandler {
	return &WebhookHandler{
		config: cfg,
	}
}

func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.verifyWebhook(w, r)
	case http.MethodPost:
		h.handleWebhookEvent(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *WebhookHandler) verifyWebhook(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	log.Printf("Webhook verification: mode=%s, token=%s", mode, token)

	if mode == "subscribe" && token == h.config.WebhookVerifyToken {
		log.Printf("Webhook verified successfully")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(challenge))
		return
	}

	log.Printf("Webhook verification failed")
	w.WriteHeader(http.StatusForbidden)
}

func (h *WebhookHandler) handleWebhookEvent(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read webhook body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var event models.WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf("Failed to parse webhook event: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("Received webhook event: object=%s, entries=%d", event.Object, len(event.Entry))

	// Process webhook events
	for _, entry := range event.Entry {
		for _, change := range entry.Changes {
			h.processWebhookChange(entry.ID, change)
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (h *WebhookHandler) processWebhookChange(entryID string, change struct {
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
}) {
	log.Printf("Processing change: field=%s, phone_number_id=%s",
		change.Field, change.Value.Metadata.PhoneNumberID)

	// Handle incoming messages
	for _, message := range change.Value.Messages {
		h.handleIncomingMessage(change.Value.Metadata.PhoneNumberID, message)
	}

	// Handle message statuses
	for _, status := range change.Value.Statuses {
		h.handleMessageStatus(change.Value.Metadata.PhoneNumberID, status)
	}
}

func (h *WebhookHandler) handleIncomingMessage(phoneNumberID string, message struct {
	ID        string `json:"id"`
	From      string `json:"from"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Text      struct {
		Body string `json:"body"`
	} `json:"text,omitempty"`
}) {
	log.Printf("Incoming message: id=%s, from=%s, type=%s, body=%s",
		message.ID, message.From, message.Type, message.Text.Body)

	// TODO: Implement your message handling logic
	// Examples:
	// - Save message to database
	// - Trigger automated responses
	// - Forward to customer service system
	// - Process commands or keywords
}

func (h *WebhookHandler) handleMessageStatus(phoneNumberID string, status struct {
	ID          string `json:"id"`
	RecipientID string `json:"recipient_id"`
	Status      string `json:"status"`
	Timestamp   string `json:"timestamp"`
}) {
	log.Printf("Message status update: id=%s, recipient=%s, status=%s",
		status.ID, status.RecipientID, status.Status)

	// TODO: Implement your status handling logic
	// Examples:
	// - Update delivery status in database
	// - Trigger notifications
	// - Update analytics
}
