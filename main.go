package main

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/drewfead/cali/internal/auth"
	"github.com/drewfead/cali/internal/calendar"
	"github.com/drewfead/cali/internal/config"
	"github.com/drewfead/cali/proto"
	protocli "github.com/drewfead/proto-cli"
	protobuf "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//go:embed event.template.ics
var eventTemplateICS string

//go:embed list-events-response.template.ics
var listEventsResponseTemplateICS string

//go:embed get-event-response.template.ics
var getEventResponseTemplateICS string

type calendarService struct {
	proto.UnimplementedCalendarServiceServer
	calendarClient *calendar.Client // Google Calendar API client (initialized lazily)
	ctx            context.Context
	cfg            *proto.CaliConfig
}

// newCalendarService creates a calendar service with lazy initialization.
// Authentication happens only when a method is first called.
func newCalendarService(cfg *proto.CaliConfig) *calendarService {
	return &calendarService{
		cfg: cfg,
	}
}

// ensureInitialized lazily initializes the calendar client on first use
func (s *calendarService) ensureInitialized(ctx context.Context) error {
	// Already initialized
	if s.calendarClient != nil {
		return nil
	}

	// Initialize Google Calendar integration
	if err := initializeGoogleCalendar(ctx, s, s.cfg); err != nil {
		return fmt.Errorf("Google Calendar integration failed: %w\n\nGoogle Calendar credentials are required. See config.example.yaml.\n\nOption 1: Service Account (for automation/cron)\nOption 2: OAuth Client (for interactive use)\n\nSee AUTHENTICATION.md for detailed setup instructions", err)
	}

	return nil
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
		slog.Info("using service account authentication", "mode", "automated")
	} else {
		slog.Info("using OAuth user authentication", "mode", "interactive")
	}

	// Create Calendar API client with optional endpoint override
	var calendarClient *calendar.Client
	if cfg.ApiEndpoint != "" {
		calendarClient, err = calendar.NewClient(ctx, httpClient, cfg.ApiEndpoint)
	} else {
		calendarClient, err = calendar.NewClient(ctx, httpClient)
	}
	if err != nil {
		return fmt.Errorf("failed to create calendar client: %w", err)
	}

	svc.calendarClient = calendarClient
	return nil
}

func (s *calendarService) AddEvent(ctx context.Context, req *proto.AddEventRequest) (*proto.AddEventResponse, error) {
	// Lazily initialize calendar client on first use
	if err := s.ensureInitialized(ctx); err != nil {
		return &proto.AddEventResponse{
			Success: false,
			Message: "Google Calendar not configured - see AUTHENTICATION.md",
		}, err
	}

	// Log calendar ID for debugging
	calendarIDForLog := "primary"
	if req.CalendarId != nil && *req.CalendarId != "" {
		calendarIDForLog = *req.CalendarId
	}
	slog.Debug("creating event",
		"calendar_id", calendarIDForLog,
		"calendar_id_ptr", req.CalendarId,
		"summary", req.Summary,
		"location", req.Location)

	// Create event via Google Calendar API
	event, err := s.calendarClient.CreateEvent(ctx, req)
	if err != nil {
		slog.Error("failed to create event", "error", err, "calendar_id", calendarIDForLog)
		return &proto.AddEventResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create event in Google Calendar: %v", err),
		}, err
	}

	// Validate that the event was actually created
	if event == nil || event.Id == "" {
		slog.Error("created event has no ID", "calendar_id", calendarIDForLog)
		return &proto.AddEventResponse{
			Success: false,
			Message: "Event creation succeeded but returned event has no ID",
		}, fmt.Errorf("created event is missing ID")
	}

	slog.Info("event created successfully", "event_id", event.Id, "calendar_id", calendarIDForLog)

	// Use calendar_id from request, default to "primary"
	calendarID := "primary"
	if req.CalendarId != nil && *req.CalendarId != "" {
		calendarID = *req.CalendarId
	}

	return &proto.AddEventResponse{
		EventId:    event.Id,
		Success:    true,
		Message:    fmt.Sprintf("Event '%s' added successfully to Google Calendar", req.Summary),
		HtmlLink:   event.HtmlLink,
		CalendarId: calendarID,
	}, nil
}

func (s *calendarService) UpdateEvent(ctx context.Context, req *proto.UpdateEventRequest) (*proto.UpdateEventResponse, error) {
	// Lazily initialize calendar client on first use
	if err := s.ensureInitialized(ctx); err != nil {
		return &proto.UpdateEventResponse{
			Success: false,
			Message: "Google Calendar not configured - see AUTHENTICATION.md",
		}, err
	}

	// Update event via Google Calendar API
	event, err := s.calendarClient.UpdateEvent(ctx, req)
	if err != nil {
		return &proto.UpdateEventResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to update event in Google Calendar: %v", err),
		}, err
	}

	// Use calendar_id from request, default to "primary"
	calendarID := "primary"
	if req.CalendarId != nil && *req.CalendarId != "" {
		calendarID = *req.CalendarId
	}

	return &proto.UpdateEventResponse{
		EventId:    event.Id,
		Success:    true,
		Message:    fmt.Sprintf("Event '%s' updated successfully in Google Calendar", event.Summary),
		HtmlLink:   event.HtmlLink,
		CalendarId: calendarID,
	}, nil
}

func (s *calendarService) DeleteEvent(ctx context.Context, req *proto.DeleteEventRequest) (*proto.DeleteEventResponse, error) {
	// Lazily initialize calendar client on first use
	if err := s.ensureInitialized(ctx); err != nil {
		return &proto.DeleteEventResponse{
			Success: false,
			Message: "Google Calendar not configured - see AUTHENTICATION.md",
		}, err
	}

	// Delete event via Google Calendar API
	err := s.calendarClient.DeleteEvent(ctx, req)
	if err != nil {
		return &proto.DeleteEventResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to delete event from Google Calendar: %v", err),
		}, err
	}

	// Use calendar_id from request, default to "primary"
	calendarID := "primary"
	if req.CalendarId != nil && *req.CalendarId != "" {
		calendarID = *req.CalendarId
	}

	return &proto.DeleteEventResponse{
		Success:    true,
		Message:    fmt.Sprintf("Event deleted successfully from Google Calendar"),
		CalendarId: calendarID,
	}, nil
}

func (s *calendarService) GetEvent(ctx context.Context, req *proto.GetEventRequest) (*proto.GetEventResponse, error) {
	// Lazily initialize calendar client on first use
	if err := s.ensureInitialized(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize calendar client: %w", err)
	}

	// Get event via Google Calendar API
	event, err := s.calendarClient.GetEvent(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	// Validate that the event was retrieved
	if event == nil || event.Id == "" {
		slog.Error("retrieved event is invalid", "event_id", req.EventId, "calendar_id", req.CalendarId)
		return nil, fmt.Errorf("retrieved event has no ID (requested: %s)", req.EventId)
	}

	// Use calendar_id from request, default to "primary"
	calendarID := "primary"
	if req.CalendarId != nil && *req.CalendarId != "" {
		calendarID = *req.CalendarId
	}

	// Convert to proto Event
	protoEvent := calendar.MapEventToProto(event, calendarID)

	return &proto.GetEventResponse{
		Event: protoEvent,
	}, nil
}

func (s *calendarService) ListEvents(req *proto.ListEventsRequest, stream proto.CalendarService_ListEventsServer) error {
	// Lazily initialize calendar client on first use
	if err := s.ensureInitialized(stream.Context()); err != nil {
		return fmt.Errorf("failed to initialize calendar client: %w", err)
	}

	// Get response channel from calendar client
	responseChan, errChan := s.calendarClient.ListEvents(stream.Context(), req)

	// Stream responses back to client
	for {
		select {
		case response, ok := <-responseChan:
			if !ok {
				// Channel closed, check for errors
				select {
				case err := <-errChan:
					if err != nil {
						return err
					}
				default:
				}
				// Successfully completed
				return nil
			}

			// Send response (contains either an event or next_anchor)
			if err := stream.Send(response); err != nil {
				return fmt.Errorf("failed to send response: %w", err)
			}

		case err := <-errChan:
			if err != nil {
				return err
			}

		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

// ICS format helper functions
func icsTimestamp(ts *timestamppb.Timestamp) string {
	if ts == nil || !ts.IsValid() {
		return ""
	}
	// Format: YYYYMMDDTHHMMSSZ
	return ts.AsTime().UTC().Format("20060102T150405Z")
}

func icsEscape(s string) string {
	// Escape special characters per RFC 5545
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func icsNow() string {
	return time.Now().UTC().Format("20060102T150405Z")
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
		slog.Error("failed to load config", "error", err, "help", "see config.example.yaml for configuration format")
		os.Exit(1)
	}

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


	// Create ICS format for calendar events (templates loaded from embedded files)
	// Response templates use {{template "event" ...}} to reuse event template definition
	// Prepend event template to response templates so they have access to the "event" definition
	icsTemplates := map[string]string{
		"calendar.Event":              eventTemplateICS,
		"calendar.ListEventsResponse": eventTemplateICS + listEventsResponseTemplateICS,
		"calendar.GetEventResponse":   eventTemplateICS + getEventResponseTemplateICS,
	}

	// Build function map with helper functions
	icsFuncMap := template.FuncMap{
		"icsTime":   icsTimestamp,
		"icsEscape": icsEscape,
		"now":       icsNow,
		"upper":     strings.ToUpper,
	}

	icsFormat, err := protocli.TemplateFormat("ics", icsTemplates, icsFuncMap)
	if err != nil {
		slog.Error("failed to create ICS format", "error", err)
		os.Exit(1)
	}

	// Create service instance with lazy authentication
	// Authentication only happens when AddEvent is called
	svc := newCalendarService(cfg)

	// Generate CLI from service
	serviceCLI := proto.CalendarServiceCommand(ctx, svc,
		protocli.WithOutputFormats(
			protocli.JSON(),
			protocli.YAML(),
			icsFormat,
		),
		protocli.WithFlagDeserializer("google.protobuf.Timestamp", timestampDeserializer),
	)

	// Create root command with config support
	rootCmd, err := protocli.RootCommand("cali",
		protocli.Service(serviceCLI, protocli.Hoisted()),
		protocli.WithEnvPrefix("CALI"),
	)
	if err != nil {
		slog.Error("failed to create root command", "error", err)
		os.Exit(1)
	}

	if err := rootCmd.Run(ctx, os.Args); err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}
