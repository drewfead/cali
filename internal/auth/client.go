package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"runtime"

	"golang.org/x/oauth2"
)

const (
	localServerPort = "8080"
	callbackPath    = "/oauth2callback"
)

// GetClient returns an authenticated HTTP client for Google Calendar API
func GetClient(ctx context.Context, config *oauth2.Config, tokenPath string) (*http.Client, error) {
	// Try to load existing token
	tok, err := LoadToken(tokenPath)
	if err == nil {
		// Token loaded successfully, return client
		return config.Client(ctx, tok), nil
	}

	// Token not found, initiate OAuth flow
	tok, err = GetTokenFromWeb(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to get token from web: %w", err)
	}

	// Save the token for future use
	if err := SaveToken(tokenPath, tok); err != nil {
		return nil, fmt.Errorf("unable to save token: %w", err)
	}

	return config.Client(ctx, tok), nil
}

// GetTokenFromWeb initiates browser-based OAuth flow
func GetTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	// Set redirect URL to local server
	config.RedirectURL = fmt.Sprintf("http://localhost:%s%s", localServerPort, callbackPath)

	// Channel to receive authorization code
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	// Create HTTP server to receive callback
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":" + localServerPort,
		Handler: mux,
	}

	// Handle OAuth callback
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no authorization code received")
			fmt.Fprintf(w, "Error: No authorization code received")
			return
		}

		codeCh <- code
		fmt.Fprintf(w, "Authorization successful! You can close this window and return to the terminal.")
	})

	// Start server in background
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("failed to start local server: %w", err)
		}
	}()

	// Generate authorization URL
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	// Open browser
	slog.Info("opening browser for authorization")
	slog.Info("if the browser doesn't open automatically, visit this URL", "url", authURL)

	if err := openBrowser(authURL); err != nil {
		slog.Warn("failed to open browser automatically", "error", err)
	}

	// Wait for authorization code or error
	var code string
	select {
	case code = <-codeCh:
		// Got authorization code
	case err := <-errCh:
		server.Shutdown(ctx)
		return nil, err
	case <-ctx.Done():
		server.Shutdown(ctx)
		return nil, ctx.Err()
	}

	// Shutdown server
	server.Shutdown(ctx)

	// Exchange authorization code for token
	tok, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("unable to exchange authorization code: %w", err)
	}

	return tok, nil
}

// openBrowser opens the specified URL in the default browser
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}
