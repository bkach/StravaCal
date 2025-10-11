package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// getCalendarService creates and returns an authenticated Google Calendar service
// using the service account JSON key from either:
// 1. GOOGLE_SERVICE_ACCOUNT environment variable (for CI/CD)
// 2. service-account.json file (for local development)
func getCalendarService() (*calendar.Service, error) {
	ctx := context.Background()

	var serviceAccountKey []byte
	var err error

	// Try to get service account key from environment variable first (for CI/CD)
	serviceAccountEnv := os.Getenv("GOOGLE_SERVICE_ACCOUNT")
	if serviceAccountEnv != "" {
		serviceAccountKey = []byte(serviceAccountEnv)
		log.Println("Using service account from GOOGLE_SERVICE_ACCOUNT environment variable")
	} else {
		// Fall back to reading from file (for local development)
		serviceAccountKey, err = os.ReadFile("service-account.json")
		if err != nil {
			return nil, fmt.Errorf("unable to read service account key (tried GOOGLE_SERVICE_ACCOUNT env var and service-account.json file): %w", err)
		}
		log.Println("Using service account from service-account.json file")
	}

	// Create credentials from service account key
	config, err := google.JWTConfigFromJSON(serviceAccountKey, calendar.CalendarScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse service account key: %w", err)
	}

	// Create calendar service
	srv, err := calendar.NewService(ctx, option.WithHTTPClient(config.Client(ctx)))
	if err != nil {
		return nil, fmt.Errorf("unable to create calendar service: %w", err)
	}

	return srv, nil
}

// syncStravaEvents synchronizes Strava events with Google Calendar
// - Creates new events that don't exist
// - Updates existing events that have changed
// - Deletes events that no longer exist on Strava
func syncStravaEvents(events []Event, srv *calendar.Service, calendarID string) error {
	ctx := context.Background()

	// Get current time for sync timestamp in Europe/London timezone
	london, _ := time.LoadLocation("Europe/London")
	now := time.Now().In(london)
	syncTime := now.Format("Mon, 2 Jan @ 3:04 PM")

	// Build a map of Strava event IDs for efficient lookup
	stravaEventMap := make(map[int64]Event)
	for _, event := range events {
		stravaEventMap[event.ID] = event
	}

	// Get all existing events from Google Calendar
	// We'll fetch events from 1 week ago to 90 days in the future
	timeMin := time.Now().AddDate(0, 0, -7).Format(time.RFC3339)
	timeMax := time.Now().AddDate(0, 0, 90).Format(time.RFC3339)

	existingEvents, err := srv.Events.List(calendarID).
		Context(ctx).
		TimeMin(timeMin).
		TimeMax(timeMax).
		SingleEvents(true).
		Do()

	if err != nil {
		return fmt.Errorf("unable to retrieve existing calendar events: %w", err)
	}

	// Track which Strava events we've seen in Google Calendar
	processedStravaIDs := make(map[int64]bool)

	// Process existing Google Calendar events
	for _, gcalEvent := range existingEvents.Items {
		// Extract Strava ID from iCalUID (format: <id>@strava.com)
		var stravaID int64
		if gcalEvent.ICalUID != "" {
			n, err := fmt.Sscanf(gcalEvent.ICalUID, "%d@strava.com", &stravaID)
			if err != nil || n != 1 || stravaID == 0 {
				// Not a Strava event or failed to parse, skip
				log.Printf("[DEBUG] Skipping non-Strava event: %s (UID: %s)", gcalEvent.Summary, gcalEvent.ICalUID)
				continue
			}
		} else {
			continue
		}

		// Check if this Strava event still exists
		stravaEvent, exists := stravaEventMap[stravaID]
		if !exists {
			// Event no longer exists on Strava, delete it
			err := srv.Events.Delete(calendarID, gcalEvent.Id).Context(ctx).Do()
			if err != nil {
				log.Printf("[ERROR] Failed to delete event %d: %v", stravaID, err)
			} else {
				log.Printf("[SYNC] Deleted: %s (no longer on Strava)", gcalEvent.Summary)
			}
			continue
		}

		// Mark this Strava event as processed
		processedStravaIDs[stravaID] = true

		// Check if the event needs updating
		needsUpdate := false
		if gcalEvent.Summary != stravaEvent.Title {
			needsUpdate = true
		}

		// Convert times to Europe/London timezone for comparison
		london, _ := time.LoadLocation("Europe/London")
		stravaStartLocal := stravaEvent.Start.In(london)
		stravaEndLocal := stravaEvent.End.In(london)

		gcalStartTime, _ := time.Parse(time.RFC3339, gcalEvent.Start.DateTime)
		gcalEndTime, _ := time.Parse(time.RFC3339, gcalEvent.End.DateTime)

		if !gcalStartTime.Equal(stravaStartLocal) || !gcalEndTime.Equal(stravaEndLocal) {
			needsUpdate = true
		}

		// Check if description has changed
		clubID, err := getClubID()
		if err != nil {
			return err
		}
		newDesc := fmt.Sprintf("Leader: %s\n\nLocation: %s\n\n%s\n\nView on Strava: %s\n\nSynced from Strava Club %s on %s",
			stravaEvent.Organizer,
			stravaEvent.Location,
			stravaEvent.Description,
			stravaEvent.URL,
			clubID,
			syncTime)

		// Normalize whitespace for comparison
		if strings.TrimSpace(gcalEvent.Description) != strings.TrimSpace(newDesc) {
			needsUpdate = true
		}

		if needsUpdate {
			// Update the event
			updatedEvent := createGoogleCalendarEvent(stravaEvent, syncTime, london)
			_, err := srv.Events.Update(calendarID, gcalEvent.Id, updatedEvent).Context(ctx).Do()
			if err != nil {
				log.Printf("[ERROR] Failed to update event %d: %v", stravaID, err)
			} else {
				log.Printf("[SYNC] Updated: %s (%s)", stravaEvent.Title, stravaStartLocal.Format("Mon 2 Jan"))
			}
		}
	}

	// Create new events that don't exist in Google Calendar
	for _, stravaEvent := range events {
		if !processedStravaIDs[stravaEvent.ID] {
			// This is a new event, create it
			newEvent := createGoogleCalendarEvent(stravaEvent, syncTime, london)
			_, err := srv.Events.Insert(calendarID, newEvent).Context(ctx).Do()
			if err != nil {
				// Check if it's a duplicate error (409)
				if strings.Contains(err.Error(), "409") || strings.Contains(err.Error(), "duplicate") {
					log.Printf("[SYNC] Event %d already exists (skipped duplicate): %s", stravaEvent.ID, stravaEvent.Title)
				} else {
					log.Printf("[ERROR] Failed to create event %d: %v", stravaEvent.ID, err)
				}
			} else {
				startLocal := stravaEvent.Start.In(london)
				log.Printf("[SYNC] Created: %s (%s)", stravaEvent.Title, startLocal.Format("Mon 2 Jan"))
			}
		}
	}

	return nil
}

// createGoogleCalendarEvent creates a Google Calendar event object from a Strava event
func createGoogleCalendarEvent(event Event, syncTime string, location *time.Location) *calendar.Event {
	startLocal := event.Start.In(location)
	endLocal := event.End.In(location)

	// Create description with all event details
	clubID, err := getClubID()
	if err != nil {
		log.Printf("[ERROR] Failed to get club ID: %v", err)
		clubID = "unknown"
	}
	description := fmt.Sprintf("Leader: %s\n\nLocation: %s\n\n%s\n\nView on Strava: %s\n\nSynced from Strava Club %s on %s",
		event.Organizer,
		event.Location,
		event.Description,
		event.URL,
		clubID,
		syncTime)

	return &calendar.Event{
		Summary:     event.Title,
		Location:    event.Location,
		Description: description,
		Start: &calendar.EventDateTime{
			DateTime: startLocal.Format(time.RFC3339),
			TimeZone: "Europe/London",
		},
		End: &calendar.EventDateTime{
			DateTime: endLocal.Format(time.RFC3339),
			TimeZone: "Europe/London",
		},
		ICalUID: fmt.Sprintf("%d@strava.com", event.ID),
		Source: &calendar.EventSource{
			Title: "Strava",
			Url:   event.URL,
		},
	}
}
