package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"back/config"
	"back/models"
)

type FacebookService struct {
	config *config.Config
	client *http.Client
}

func NewFacebookService(cfg *config.Config) *FacebookService {
	return &FacebookService{
		config: cfg,
		client: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
	}
}

// Graph API error envelope
type fbError struct {
	Error struct {
		Message      string `json:"message"`
		Type         string `json:"type"`
		Code         int    `json:"code"`
		ErrorSubcode int    `json:"error_subcode"`
		FbTraceID    string `json:"fbtrace_id"`
	} `json:"error"`
}

func readFBError(body io.Reader) string {
	b, _ := io.ReadAll(body)
	var e fbError
	if json.Unmarshal(b, &e) == nil && e.Error.Message != "" {
		return fmt.Sprintf("%s (type=%s code=%d subcode=%d trace=%s)",
			e.Error.Message, e.Error.Type, e.Error.Code, e.Error.ErrorSubcode, e.Error.FbTraceID)
	}
	return string(b)
}

// Exchange authorization code for access token (Embedded Signup / OAuth)
func (f *FacebookService) ExchangeToken(authCode, redirectURI string) (*models.FacebookTokenResponse, error) {
	if f.config.FacebookAppID == "" || f.config.FacebookAppSecret == "" {
		return nil, fmt.Errorf("missing Facebook app credentials in config")
	}

	// For WhatsApp Embedded Signup, try multiple strategies for redirect_uri
	strategies := []map[string]string{
		// Strategy 1: No redirect_uri (common for embedded signup)
		{
			"client_id":     f.config.FacebookAppID,
			"client_secret": f.config.FacebookAppSecret,
			"code":          authCode,
		},
		// Strategy 2: Empty redirect_uri
		{
			"client_id":     f.config.FacebookAppID,
			"client_secret": f.config.FacebookAppSecret,
			"code":          authCode,
			"redirect_uri":  "",
		},
		// Strategy 3: Use provided redirect_uri
		{
			"client_id":     f.config.FacebookAppID,
			"client_secret": f.config.FacebookAppSecret,
			"code":          authCode,
			"redirect_uri":  redirectURI,
		},
	}

	// If redirectURI is empty, only try first two strategies
	strategiesToTry := strategies
	if redirectURI == "" {
		strategiesToTry = strategies[:2]
	}

	var lastError error
	for i, strategy := range strategiesToTry {
		fmt.Printf("Trying token exchange strategy %d: %+v\n", i+1, strategy)

		form := url.Values{}
		for key, value := range strategy {
			form.Set(key, value)
		}

		req, err := http.NewRequest(http.MethodPost, "https://graph.facebook.com/v19.0/oauth/access_token", strings.NewReader(form.Encode()))
		if err != nil {
			lastError = fmt.Errorf("build token exchange request: %w", err)
			continue
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")

		resp, err := f.client.Do(req)
		if err != nil {
			lastError = fmt.Errorf("token exchange request failed: %w", err)
			continue
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == http.StatusOK {
			// Success!
			fmt.Printf("✅ Token exchange successful with strategy %d\n", i+1)
			var tokenResp models.FacebookTokenResponse
			if err := json.Unmarshal(body, &tokenResp); err != nil {
				return nil, fmt.Errorf("failed to parse token response: %w; raw=%s", err, string(body))
			}
			return &tokenResp, nil
		}

		// Parse error for logging
		var fe fbError
		_ = json.Unmarshal(body, &fe)
		if fe.Error.Message != "" {
			lastError = fmt.Errorf("strategy %d failed: %s (type=%s code=%d subcode=%d)",
				i+1, fe.Error.Message, fe.Error.Type, fe.Error.Code, fe.Error.ErrorSubcode)
			fmt.Printf("❌ Strategy %d failed: %s\n", i+1, fe.Error.Message)
		} else {
			lastError = fmt.Errorf("strategy %d failed (%s): %s", i+1, resp.Status, string(body))
			fmt.Printf("❌ Strategy %d failed: %s\n", i+1, resp.Status)
		}
	}

	return nil, fmt.Errorf("all token exchange strategies failed, last error: %w", lastError)
}

// Get business accounts associated with access token
func (f *FacebookService) GetBusinessAccounts(accessToken string) ([]models.FacebookBusinessAccount, error) {
	u, _ := url.Parse("https://graph.facebook.com/v19.0/me/businesses")
	q := u.Query()
	q.Set("fields", "id,name,verification_status,profile_picture_uri")
	q.Set("access_token", accessToken)
	u.RawQuery = q.Encode()

	fmt.Printf("🔍 Fetching business accounts from: %s\n", u.String())

	resp, err := f.client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("business accounts request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("📄 Business accounts response status: %d\n", resp.StatusCode)
	fmt.Printf("📄 Business accounts response body: %s\n", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("business accounts failed (%s): %s", resp.Status, readFBError(strings.NewReader(string(body))))
	}

	var response struct {
		Data []models.FacebookBusinessAccount `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode business accounts response: %w", err)
	}

	fmt.Printf("✅ Successfully parsed %d business accounts\n", len(response.Data))

	// If no business accounts found, try alternative WhatsApp-specific endpoint
	if len(response.Data) == 0 {
		fmt.Printf("🔄 No standard business accounts found, trying WhatsApp-specific approach...\n")
		return f.getWhatsAppBusinessAccounts(accessToken)
	}

	return response.Data, nil
}

// Alternative method to get WhatsApp Business Accounts directly
func (f *FacebookService) getWhatsAppBusinessAccounts(accessToken string) ([]models.FacebookBusinessAccount, error) {
	// Try to get WhatsApp Business Accounts directly
	u, _ := url.Parse("https://graph.facebook.com/v19.0/me")
	q := u.Query()
	q.Set("fields", "id,name")
	q.Set("access_token", accessToken)
	u.RawQuery = q.Encode()

	fmt.Printf("🔍 Trying user info endpoint: %s\n", u.String())

	resp, err := f.client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("user info request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("📄 User info response status: %d\n", resp.StatusCode)
	fmt.Printf("📄 User info response body: %s\n", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info failed (%s): %s", resp.Status, string(body))
	}

	// For embedded signup, we might need to create a virtual business account
	// since the WABA might be created but not yet show up in standard business endpoints
	var userInfo struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	fmt.Printf("ℹ️ User ID: %s, Name: %s\n", userInfo.ID, userInfo.Name)

	// Return empty for now - this indicates we need to handle the embedded signup differently
	// The frontend message event should contain the WABA ID that was created
	return []models.FacebookBusinessAccount{}, nil
}

// Get phone numbers for a specific WABA
func (f *FacebookService) GetPhoneNumbers(accessToken, wabaID string) ([]models.FacebookPhoneNumber, error) {
	u, _ := url.Parse(fmt.Sprintf("https://graph.facebook.com/v19.0/%s/phone_numbers", url.PathEscape(wabaID)))
	q := u.Query()
	q.Set("fields", "id,display_phone_number,verified_name,quality_rating,status,code_verification_status")
	q.Set("access_token", accessToken)
	u.RawQuery = q.Encode()

	resp, err := f.client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("phone numbers request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("phone numbers failed (%s): %s", resp.Status, readFBError(resp.Body))
	}

	var response struct {
		Data []models.FacebookPhoneNumber `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode phone numbers response: %w", err)
	}
	return response.Data, nil
}

// Validate access token (simple check)
func (f *FacebookService) ValidateToken(accessToken string) (bool, error) {
	u, _ := url.Parse("https://graph.facebook.com/v19.0/me")
	q := u.Query()
	q.Set("access_token", accessToken)
	u.RawQuery = q.Encode()

	resp, err := f.client.Get(u.String())
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	return false, fmt.Errorf("token invalid (%s): %s", resp.Status, readFBError(resp.Body))
}
