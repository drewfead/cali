package main

import (
	"context"
	"testing"

	"github.com/drewfead/cali/pkg/googlecaltest"
	"github.com/drewfead/cali/proto"
	protocli "github.com/drewfead/proto-cli"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ptr returns a pointer to the given value (helper for proto optional fields)
func ptr[T any](v T) *T {
	return &v
}

// loadTestConfig loads the configuration for integration tests
func loadTestConfig(t *testing.T) *proto.CaliConfig {
	t.Helper()

	cfg := &proto.CaliConfig{}
	configLoader := protocli.NewConfigLoader(
		protocli.SingleCommandMode,
		protocli.FileConfig(protocli.DefaultConfigPaths("cali")...),
		protocli.EnvPrefix("CALI"),
	)

	if err := configLoader.LoadServiceConfig(nil, "cali", cfg); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	return cfg
}

// TestIntegration_GoogleCalendarAPI tests the Google Calendar integration using a mock server.
// This test runs without requiring real Google Calendar credentials.
func TestIntegration_GoogleCalendarAPI(t *testing.T) {
	// Create mock Google Calendar API server
	mockServer := googlecaltest.NewServer()
	defer mockServer.Close()

	ctx := context.Background()

	// Load configuration and override API endpoint to point at mock
	cfg := loadTestConfig(t)
	cfg.ApiEndpoint = mockServer.URL

	// Initialize service
	svc := newCalendarService(cfg)

	// Force initialization - should succeed with mock server
	if err := svc.ensureInitialized(ctx); err != nil {
		t.Fatalf("failed to initialize with mock server: %v", err)
	}

	tests := []struct {
		name       string
		request    *proto.AddEventRequest
		wantErr    bool
		skipReason string
	}{
		{
			name: "create event with default times",
			request: &proto.AddEventRequest{
				Summary:     "Integration Test Event",
				Description: ptr("This event was created by an automated integration test"),
				Location:    ptr("Test Location"),
			},
			wantErr: false,
		},
		{
			name: "create event with specific calendar ID",
			request: &proto.AddEventRequest{
				Summary:     "Integration Test Event - Custom Calendar",
				Description: ptr("Testing with explicit calendar ID"),
				Location:    ptr("Virtual"),
				CalendarId:  ptr("primary"),
			},
			wantErr: false,
		},
		{
			name: "create event with explicit times",
			request: &proto.AddEventRequest{
				Summary:     "Integration Test Event - With Times",
				Description: ptr("Testing with start and end times"),
				StartTime:   timestamppb.Now(),
				EndTime:     timestamppb.Now(),
			},
			wantErr: false,
		},
		{
			name: "create event on shared calendar",
			request: &proto.AddEventRequest{
				Summary:     "Integration Test - Shared Calendar",
				Description: ptr("Testing service account with shared calendar"),
				Location:    ptr("Automated"),
				// Update this calendar ID to match your test calendar
				CalendarId: ptr("77375caf1a9297541a0f25d2ce7cae6b7ac6b455232feb324c2610db6b6d7450@group.calendar.google.com"),
			},
			wantErr:    false,
			skipReason: "requires calendar ID to be shared with service account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}

			// Reset mock server state for each test
			defer mockServer.Reset()

			// Call AddEvent
			resp, err := svc.AddEvent(ctx, tt.request)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("AddEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If we expected an error, we're done
			if tt.wantErr {
				return
			}

			// Verify response
			if resp == nil {
				t.Fatal("AddEvent() returned nil response")
			}

			if !resp.Success {
				t.Errorf("AddEvent() success = false, message = %s", resp.Message)
			}

			if resp.EventId == "" {
				t.Error("AddEvent() returned empty event ID")
			}

			if resp.HtmlLink == "" {
				t.Error("AddEvent() returned empty HTML link")
			}

			// Log success details
			t.Logf("✓ Event created successfully")
			t.Logf("  Event ID: %s", resp.EventId)
			t.Logf("  Calendar: %s", resp.CalendarId)
			t.Logf("  View at: %s", resp.HtmlLink)
		})
	}
}

// TestIntegration_ServiceAccountAuth tests service account authentication specifically.
// This test verifies that service account credentials are loaded correctly.
func TestIntegration_ServiceAccountAuth(t *testing.T) {
	t.Skip("requires service account credentials - see AUTHENTICATION.md for setup")

	ctx := context.Background()

	// Load configuration
	cfg := loadTestConfig(t)

	// Check if service account is configured
	if cfg.Auth == nil || cfg.Auth.ServiceAccount == nil {
		t.Skip("service account not configured in config")
	}

	svc := newCalendarService(cfg)

	// Force initialization
	if err := svc.ensureInitialized(ctx); err != nil {
		t.Fatal("expected calendar client to be initialized with service account: " + err.Error())
	}

	// Try creating a test event
	resp, err := svc.AddEvent(ctx, &proto.AddEventRequest{
		Summary:     "Service Account Test Event",
		Description: ptr("Testing service account authentication"),
		Location:    ptr("Automated Test"),
	})
	if err != nil {
		t.Fatalf("AddEvent() with service account failed: %v", err)
	}

	if !resp.Success {
		t.Errorf("AddEvent() success = false, message = %s", resp.Message)
	}

	t.Logf("✓ Service account authentication working")
	t.Logf("  Event created: %s", resp.HtmlLink)
}

// TestIntegration_OAuthAuth tests OAuth user authentication specifically.
// This test verifies that OAuth credentials are loaded correctly.
func TestIntegration_OAuthAuth(t *testing.T) {
	t.Skip("requires OAuth credentials - see AUTHENTICATION.md for setup")

	ctx := context.Background()

	// Load configuration
	cfg := loadTestConfig(t)

	// Check if OAuth client is configured
	if cfg.Auth == nil || cfg.Auth.OauthClient == nil {
		t.Skip("OAuth client not configured in config")
	}

	// Temporarily remove service account from config to force OAuth
	originalServiceAccount := cfg.Auth.ServiceAccount
	cfg.Auth.ServiceAccount = nil
	defer func() {
		cfg.Auth.ServiceAccount = originalServiceAccount
	}()

	svc := newCalendarService(cfg)

	// Force initialization
	if err := svc.ensureInitialized(ctx); err != nil {
		t.Fatal("expected calendar client to be initialized with OAuth: " + err.Error())
	}

	// Try creating a test event
	resp, err := svc.AddEvent(ctx, &proto.AddEventRequest{
		Summary:     "OAuth Test Event",
		Description: ptr("Testing OAuth user authentication"),
		Location:    ptr("Interactive Test"),
	})
	if err != nil {
		t.Fatalf("AddEvent() with OAuth failed: %v", err)
	}

	if !resp.Success {
		t.Errorf("AddEvent() success = false, message = %s", resp.Message)
	}

	t.Logf("✓ OAuth authentication working")
	t.Logf("  Event created: %s", resp.HtmlLink)
}
