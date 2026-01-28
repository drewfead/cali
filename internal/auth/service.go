package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

// GetServiceAccountClient creates an authenticated HTTP client using a service account
func GetServiceAccountClient(ctx context.Context, keyPath string) (*http.Client, error) {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read service account key: %w", err)
	}

	// Verify this is a service account credential
	credType, err := DetectCredentialType(data)
	if err != nil {
		return nil, err
	}
	if credType != CredentialTypeServiceAccount {
		return nil, fmt.Errorf("expected service account credentials, got %s", credType)
	}

	// Create JWT config from service account JSON
	config, err := google.JWTConfigFromJSON(data, calendar.CalendarScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse service account key: %w", err)
	}

	// Return authenticated HTTP client
	// Service accounts don't need token refresh - they generate tokens on demand
	return config.Client(ctx), nil
}
