// Package simplefin provides an HTTP client for the SimpleFIN protocol.
// It fetches account and balance data from a SimpleFIN Bridge access URL.
package simplefin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Org represents the financial institution that owns an account.
type Org struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// Account represents a single financial account returned by the SimpleFIN protocol.
type Account struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Currency    string `json:"currency"`
	Balance     string `json:"balance"`
	BalanceDate int64  `json:"balance-date"`
	Org         Org    `json:"org"`
}

// AccountSet is the top-level response from the SimpleFIN /accounts endpoint.
type AccountSet struct {
	Errors   []string  `json:"errors"`
	Accounts []Account `json:"accounts"`
}

// FetchAccounts requests account data from the SimpleFIN /accounts endpoint.
//
// accessURL is the full access URL (may contain embedded Basic Auth credentials,
// e.g. https://user:pass@beta.bridge.simplefin.org/simplefin).
//
// If startDate is non-nil, a start-date query parameter is added (Unix epoch).
// The balances-only=1 query parameter is always added.
//
// Basic Auth credentials embedded in accessURL are extracted and sent as an
// Authorization header; they are removed from the request URL to prevent
// credential leakage in logs.
func FetchAccounts(accessURL string, startDate *time.Time) (*AccountSet, error) {
	parsed, err := url.Parse(accessURL)
	if err != nil {
		return nil, fmt.Errorf("simplefin: invalid access URL: %w", err)
	}

	// Extract embedded credentials before building the request.
	var user, pass string
	if parsed.User != nil {
		user = parsed.User.Username()
		pass, _ = parsed.User.Password()
		// Clear credentials from URL so they don't appear in outbound request URL.
		parsed.User = nil
	}

	// Build query parameters.
	q := parsed.Query()
	q.Set("balances-only", "1")
	if startDate != nil {
		q.Set("start-date", strconv.FormatInt(startDate.Unix(), 10))
	}
	parsed.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("simplefin: failed to build request: %w", err)
	}

	// Apply Basic Auth via header (credentials never in URL).
	if user != "" {
		req.SetBasicAuth(user, pass)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("simplefin: HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("simplefin: unexpected HTTP status %d", resp.StatusCode)
	}

	var result AccountSet
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("simplefin: failed to decode response: %w", err)
	}

	return &result, nil
}
