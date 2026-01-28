package main

import (
	"context"
	"fmt"
	"os"
	"time"

	protocli "github.com/drewfead/proto-cli"
	"github.com/drewfead/cali/internal/auth"
	"github.com/drewfead/cali/internal/calendar"
	"github.com/drewfead/cali/internal/config"
	"github.com/drewfead/cali/proto"
	protobuf "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type calendarService struct {
	proto.UnimplementedCalendarServiceServer
	calendarClient *calendar.Client // Google Calendar API client
}

func newCalendarService(ctx context.Context, cfg *proto.CaliConfig) *calendarService {
	svc := &calendarService{}

	// Initialize Google Calendar integration (required)
	if err := initializeGoogleCalendar(ctx, svc, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Google Calendar integration failed: %v\n\n", err)
		fmt.Fprintf(os.Stderr, "Google Calendar credentials are required. See config.example.yaml.\n\n")
		fmt.Fprintf(os.Stderr, "Option 1: Service Account (for automation/cron)\n")
		fmt.Fprintf(os.Stderr, "Option 2: OAuth Client (for interactive use)\n\n")
		fmt.Fprintf(os.Stderr, "See AUTHENTICATION.md for detailed setup instructions.\n")
		os.Exit(1)
	}

	return svc
}

func initializeGoogleCalendar(ctx context.Context, svc *calendarService, cfg *proto.CaliConfig) error {
	// Ensure config directory exists
	if err := config.EnsureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Check if we have auth config
	if cfg.Auth == nil {
		return fmt.Errorf("no auth configuration found")
	}

	// Determine token path (use config or default)
	tokenPath := cfg.Auth.OauthTokenPath
	if tokenPath == "" {
		defaultPath, _ := config.GetTokenPath()
		tokenPath = defaultPath
	}

	// Get authenticated HTTP client from typed config
	httpClient, err := auth.GetClientFromConfig(ctx, cfg.Auth, tokenPath)
	if err != nil {
		return fmt.Errorf("failed to get authenticated client: %w", err)
	}

	// Determine auth mode for logging
	if cfg.Auth.ServiceAccount != nil && cfg.Auth.ServiceAccount.ClientEmail != "" {
		fmt.Fprintf(os.Stderr, "Using service account authentication (automated mode)\n")
	} else {
		fmt.Fprintf(os.Stderr, "Using OAuth user authentication (interactive mode)\n")
	}

	// Create Calendar API client
	calendarClient, err := calendar.NewClient(ctx, httpClient)
	if err != nil {
		return fmt.Errorf("failed to create calendar client: %w", err)
	}

	svc.calendarClient = calendarClient
	return nil
}


func (s *calendarService) AddEvent(ctx context.Context, req *proto.AddEventRequest) (*proto.AddEventResponse, error) {
	// Calendar client is required (no fallback)
	if s.calendarClient == nil {
		return &proto.AddEventResponse{
			Success: false,
			Message: "Google Calendar not configured - see AUTHENTICATION.md",
		}, fmt.Errorf("calendar client not initialized")
	}

	// Create event via Google Calendar API
	event, err := s.calendarClient.CreateEvent(ctx, req)
	if err != nil {
		return &proto.AddEventResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create event in Google Calendar: %v", err),
		}, err
	}

	// Use calendar_id from request, default to "primary"
	calendarID := req.CalendarId
	if calendarID == "" {
		calendarID = "primary"
	}

	return &proto.AddEventResponse{
		EventId:    event.Id,
		Success:    true,
		Message:    fmt.Sprintf("Event '%s' added successfully to Google Calendar", req.Title),
		HtmlLink:   event.HtmlLink,
		CalendarId: calendarID,
	}, nil
}

func main() {
	ctx := context.Background()

	// Load typed configuration
	cfg := &proto.CaliConfig{}
	configLoader := protocli.NewConfigLoader(
		protocli.SingleCommandMode,
		protocli.FileConfig(protocli.DefaultConfigPaths("cali")...),
		protocli.EnvPrefix("CALI"),
	)

	// Load config (this will merge files + env vars + flags)
	if err := configLoader.LoadServiceConfig(nil, "cali", cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		fmt.Fprintf(os.Stderr, "See config.example.yaml for configuration format.\n")
		os.Exit(1)
	}

	// Initialize service with config
	impl := newCalendarService(ctx, cfg)

	// Create timestamp deserializer for all timestamp fields
	timestampDeserializer := func(ctx context.Context, flags protocli.FlagContainer) (protobuf.Message, error) {
		timeStr := flags.String()
		// If no timestamp provided, return empty timestamp (mapper will apply defaults)
		if timeStr == "" {
			return &timestamppb.Timestamp{}, nil
		}
		t, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			return nil, fmt.Errorf("invalid timestamp format (expected RFC3339): %w", err)
		}
		return timestamppb.New(t), nil
	}

	// Generate CLI from service
	serviceCLI := proto.CalendarServiceServiceCommand(ctx, impl,
		protocli.WithOutputFormats(
			protocli.JSON(),
			protocli.YAML(),
		),
		protocli.WithFlagDeserializer("google.protobuf.Timestamp", timestampDeserializer),
	)

	// Create root command with config support
	rootCmd := protocli.RootCommand("cali",
		protocli.WithService(serviceCLI),
		protocli.WithEnvPrefix("CALI"),
	)

	if err := rootCmd.Run(ctx, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
