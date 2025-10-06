package services

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
	"back/models"
)

// In-memory storage (replace with database in production)
type StorageService struct {
	businesses map[string]*models.BusinessAccount
	mutex      sync.RWMutex
}

func NewStorageService() *StorageService {
	return &StorageService{
		businesses: make(map[string]*models.BusinessAccount),
		mutex:      sync.RWMutex{},
	}
}

func (s *StorageService) SaveBusinessAccount(account *models.BusinessAccount) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	account.UpdatedAt = time.Now()
	if account.CreatedAt.IsZero() {
		account.CreatedAt = time.Now()
	}

	s.businesses[account.WABAID] = account
	return nil
}

func (s *StorageService) GetBusinessAccount(wabaID string) (*models.BusinessAccount, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	account, exists := s.businesses[wabaID]
	if !exists {
		return nil, fmt.Errorf("business account not found")
	}

	return account, nil
}

func (s *StorageService) ListBusinessAccounts() ([]*models.BusinessAccount, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	accounts := make([]*models.BusinessAccount, 0, len(s.businesses))
	for _, account := range s.businesses {
		accounts = append(accounts, account)
	}

	return accounts, nil
}

func (s *StorageService) DeleteBusinessAccount(wabaID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.businesses, wabaID)
	return nil
}

// Utility method to export data (for backup/migration)
func (s *StorageService) ExportData() (string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	data, err := json.MarshalIndent(s.businesses, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to export data: %w", err)
	}

	return string(data), nil
}
