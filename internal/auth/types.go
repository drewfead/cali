package auth

import (
	"encoding/json"
	"fmt"
)

// CredentialType represents the type of authentication credentials
type CredentialType int

const (
	CredentialTypeUnknown CredentialType = iota
	CredentialTypeOAuthClient
	CredentialTypeServiceAccount
)

// DetectCredentialType examines the JSON structure to determine credential type
func DetectCredentialType(data []byte) (CredentialType, error) {
	var check map[string]interface{}
	if err := json.Unmarshal(data, &check); err != nil {
		return CredentialTypeUnknown, fmt.Errorf("failed to parse credential file: %w", err)
	}

	// Service account has "type": "service_account"
	if typ, ok := check["type"].(string); ok && typ == "service_account" {
		return CredentialTypeServiceAccount, nil
	}

	// OAuth client has "installed" or "web" key
	if _, ok := check["installed"]; ok {
		return CredentialTypeOAuthClient, nil
	}
	if _, ok := check["web"]; ok {
		return CredentialTypeOAuthClient, nil
	}

	return CredentialTypeUnknown, fmt.Errorf("unknown credential type")
}

func (t CredentialType) String() string {
	switch t {
	case CredentialTypeOAuthClient:
		return "OAuth Client"
	case CredentialTypeServiceAccount:
		return "Service Account"
	default:
		return "Unknown"
	}
}
