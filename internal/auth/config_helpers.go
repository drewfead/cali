package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/drewfead/cali/proto"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

// GetClientFromConfig creates an authenticated HTTP client from typed config
func GetClientFromConfig(ctx context.Context, cfg *proto.AuthConfig, tokenPath string) (*http.Client, error) {
	// Try service account first
	if cfg.ServiceAccount != nil && cfg.ServiceAccount.ClientEmail != "" {
		return GetServiceAccountClientFromConfig(ctx, cfg.ServiceAccount)
	}

	// Fall back to OAuth
	if cfg.OauthClient != nil && cfg.OauthClient.ClientId != "" {
		return GetOAuthClientFromConfig(ctx, cfg.OauthClient, tokenPath)
	}

	return nil, fmt.Errorf("no credentials configured (need service_account or oauth_client)")
}

// GetServiceAccountClientFromConfig creates a service account client from typed config
func GetServiceAccountClientFromConfig(ctx context.Context, creds *proto.ServiceAccountCredentials) (*http.Client, error) {
	// Convert proto message to JSON that google.JWTConfigFromJSON expects
	jsonData, err := serviceAccountToJSON(creds)
	if err != nil {
		return nil, fmt.Errorf("failed to convert service account config to JSON: %w", err)
	}

	// Create JWT config from the JSON
	config, err := google.JWTConfigFromJSON(jsonData, calendar.CalendarScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse service account config: %w", err)
	}

	return config.Client(ctx), nil
}

// GetOAuthClientFromConfig creates an OAuth client from typed config
func GetOAuthClientFromConfig(ctx context.Context, creds *proto.OAuthClientCredentials, tokenPath string) (*http.Client, error) {
	// Convert proto message to JSON that google.ConfigFromJSON expects
	jsonData, err := oauthClientToJSON(creds)
	if err != nil {
		return nil, fmt.Errorf("failed to convert OAuth config to JSON: %w", err)
	}

	// Parse OAuth config
	config, err := google.ConfigFromJSON(jsonData, calendar.CalendarScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse OAuth config: %w", err)
	}

	// Try to load existing token
	tok, err := LoadToken(tokenPath)
	if err == nil {
		// Token loaded successfully
		return config.Client(ctx, tok), nil
	}

	// Token not found, initiate OAuth flow
	tok, err = GetTokenFromWeb(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to get token from web: %w", err)
	}

	// Save the token
	if err := SaveToken(tokenPath, tok); err != nil {
		return nil, fmt.Errorf("unable to save token: %w", err)
	}

	return config.Client(ctx, tok), nil
}

// serviceAccountToJSON converts ServiceAccountCredentials proto to JSON
func serviceAccountToJSON(creds *proto.ServiceAccountCredentials) ([]byte, error) {
	// Create a map matching Google's expected JSON structure
	data := map[string]interface{}{
		"type":                        creds.Type,
		"project_id":                  creds.ProjectId,
		"private_key_id":              creds.PrivateKeyId,
		"private_key":                 creds.PrivateKey,
		"client_email":                creds.ClientEmail,
		"client_id":                   creds.ClientId,
		"auth_uri":                    creds.AuthUri,
		"token_uri":                   creds.TokenUri,
		"auth_provider_x509_cert_url": creds.AuthProviderX509CertUrl,
		"client_x509_cert_url":        creds.ClientX509CertUrl,
	}

	return json.Marshal(data)
}

// oauthClientToJSON converts OAuthClientCredentials proto to JSON
func oauthClientToJSON(creds *proto.OAuthClientCredentials) ([]byte, error) {
	// Create a map matching Google's expected JSON structure for desktop apps
	installed := map[string]interface{}{
		"client_id":                   creds.ClientId,
		"project_id":                  creds.ProjectId,
		"auth_uri":                    creds.AuthUri,
		"token_uri":                   creds.TokenUri,
		"auth_provider_x509_cert_url": creds.AuthProviderX509CertUrl,
		"client_secret":               creds.ClientSecret,
		"redirect_uris":               creds.RedirectUris,
	}

	data := map[string]interface{}{
		"installed": installed,
	}

	return json.Marshal(data)
}
