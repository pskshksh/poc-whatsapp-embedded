package handlers

import (
	"back/services"
	"encoding/json"
	"net/http"
)

type BusinessHandler struct {
	storage *services.StorageService
}

func NewBusinessHandler(storage *services.StorageService) *BusinessHandler {
	return &BusinessHandler{
		storage: storage,
	}
}

func (h *BusinessHandler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	accounts, err := h.storage.ListBusinessAccounts()
	if err != nil {
		http.Error(w, "Failed to fetch accounts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"accounts": accounts,
		"count":    len(accounts),
	})
}

func (h *BusinessHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	wabaID := r.URL.Query().Get("waba_id")
	if wabaID == "" {
		http.Error(w, "WABA ID is required", http.StatusBadRequest)
		return
	}

	account, err := h.storage.GetBusinessAccount(wabaID)
	if err != nil {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"account": account,
	})
}

func (h *BusinessHandler) ExportData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data, err := h.storage.ExportData()
	if err != nil {
		http.Error(w, "Failed to export data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=whatsapp_accounts.json")
	w.Write([]byte(data))
}
