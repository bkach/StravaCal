// Strava to Google Calendar Sync
//
// This program fetches upcoming club events from Strava using the undocumented
// group_events API endpoint and syncs them with a Google Calendar.
//
// CRITICAL: Uses partner API endpoint not in official documentation:
// GET /clubs/{id}/group_events?upcoming=true
//
// Features:
// - Fetches events from Strava club (configurable via STRAVA_CLUB_ID)
// - Syncs events to Google Calendar (creates, updates, deletes)
// - Generates HTML schedule for web display
// - Generates ICS calendar file
// - Backs up events to JSON file
//
// Required Environment Variables:
// - STRAVA_CLIENT_ID: Strava OAuth client ID
// - STRAVA_CLUB_ID: Strava club ID to fetch events from
// - CLIENT_SECRET: Strava OAuth client secret
// - REFRESH_TOKEN: Strava OAuth refresh token
// - GOOGLE_CALENDAR_ID: Target Google Calendar ID
// - GOOGLE_SERVICE_ACCOUNT: Google service account JSON (base64 encoded or JSON string)
//
// Authentication:
// - Strava: OAuth2 with refresh token
// - Google Calendar: Service account (service-account.json)
//
// Successfully validated October 2025 with real events
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"time"
)

const (
	eventsFile   = "output/events/events.json"
	calendarFile = "output/calendar.ics"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "test":
			testWithSampleData()
			return
		case "ics":
			generateICSOnly()
			return
		case "gcal":
			syncGoogleCalendarOnly()
			return
		}
	}

	// Default: Full sync - fetch from Strava, sync to Google Calendar, generate ICS
	log.Println("Starting Strava to Google Calendar Sync...")

	// Load Strava tokens
	tokens, err := loadTokens()
	if err != nil {
		log.Fatalf("Failed to load tokens: %v", err)
	}

	// Fetch events from Strava
	log.Println("Fetching club events from Strava API...")
	stravaEvents, err := fetchClubEvents(tokens)
	if err != nil {
		log.Printf("Failed to fetch events from API: %v", err)
		log.Println("API might be temporarily unavailable.")
		return
	}

	log.Printf("Fetched %d events from Strava", len(stravaEvents))

	// Convert Strava events to our format
	var convertedEvents []Event
	for _, se := range stravaEvents {
		event, err := convertStravaEvent(se)
		if err != nil {
			log.Printf("Failed to convert event %d: %v", se.ID, err)
			continue
		}
		convertedEvents = append(convertedEvents, *event)
	}

	// Filter and sort events
	log.Println("Filtering and sorting events...")
	finalEvents := filterAndSortEvents(convertedEvents)

	// Save events to JSON for backup
	log.Printf("Saving %d events to %s...", len(finalEvents), eventsFile)
	if err := saveEvents(finalEvents); err != nil {
		log.Fatalf("Failed to save events: %v", err)
	}

	// Get Google Calendar ID from environment
	calendarID := os.Getenv("GOOGLE_CALENDAR_ID")
	if calendarID == "" {
		log.Println("Warning: GOOGLE_CALENDAR_ID not set, skipping Google Calendar sync")
	} else {
		// Authenticate with Google Calendar
		log.Println("Authenticating with Google Calendar...")
		calendarService, err := getCalendarService()
		if err != nil {
			log.Fatalf("Failed to authenticate with Google Calendar: %v", err)
		}

		// Filter events for next 60 days (same as ICS generation)
		now := time.Now()
		sixtyDaysFromNow := now.AddDate(0, 0, 60)

		var eventsToSync []Event
		for _, event := range finalEvents {
			if event.Start.After(now) && event.Start.Before(sixtyDaysFromNow) {
				eventsToSync = append(eventsToSync, event)
			}
		}

		// Sync events with Google Calendar
		log.Printf("Syncing %d events with Google Calendar...", len(eventsToSync))
		if err := syncStravaEvents(eventsToSync, calendarService, calendarID); err != nil {
			log.Fatalf("Failed to sync events with Google Calendar: %v", err)
		}

		log.Println("✓ Google Calendar sync completed successfully!")
	}

	// Generate ICS file
	log.Println("Generating ICS file...")
	generateICSFromCache()

	log.Println("✓ All tasks completed successfully!")
}

// generateICSFromCache generates ICS file from cached events
func generateICSFromCache() {
	// Load events from JSON
	events, err := loadExistingEvents()
	if err != nil {
		log.Fatalf("Failed to load existing events: %v", err)
	}

	// Filter for events in the next 60 days
	now := time.Now()
	sixtyDaysFromNow := now.AddDate(0, 0, 60)

	var filteredEvents []Event
	for _, event := range events {
		if event.Start.After(now) && event.Start.Before(sixtyDaysFromNow) {
			filteredEvents = append(filteredEvents, event)
		}
	}

	// Sort chronologically
	sort.Slice(filteredEvents, func(i, j int) bool {
		return filteredEvents[i].Start.Before(filteredEvents[j].Start)
	})

	// Generate and save ICS file
	icsContent := generateICS(filteredEvents)
	if err := os.WriteFile(calendarFile, []byte(icsContent), 0644); err != nil {
		log.Fatalf("Error saving ICS file: %v", err)
	}

	log.Printf("Generated %s with %d events from next 60 days", calendarFile, len(filteredEvents))
}

// generateICSOnly generates only the ICS file from cached events
func generateICSOnly() {
	log.Println("Generating ICS file from cached events...")

	// Load events from JSON
	events, err := loadExistingEvents()
	if err != nil {
		log.Fatalf("Failed to load existing events: %v", err)
	}

	// Filter for events in the next 60 days
	now := time.Now()
	sixtyDaysFromNow := now.AddDate(0, 0, 60)

	var filteredEvents []Event
	for _, event := range events {
		if event.Start.After(now) && event.Start.Before(sixtyDaysFromNow) {
			filteredEvents = append(filteredEvents, event)
		}
	}

	// Sort chronologically
	sort.Slice(filteredEvents, func(i, j int) bool {
		return filteredEvents[i].Start.Before(filteredEvents[j].Start)
	})

	// Ensure output directory exists
	if err := os.MkdirAll("output/schedules", 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Generate and save ICS file
	icsContent := generateICS(filteredEvents)
	if err := os.WriteFile(calendarFile, []byte(icsContent), 0644); err != nil {
		log.Fatalf("Error saving ICS file: %v", err)
	}

	log.Printf("Generated %s with %d events", calendarFile, len(filteredEvents))
}

// syncGoogleCalendarOnly syncs cached events to Google Calendar only
func syncGoogleCalendarOnly() {
	log.Println("Syncing cached events to Google Calendar...")

	// Load events from JSON
	events, err := loadExistingEvents()
	if err != nil {
		log.Fatalf("Failed to load existing events: %v", err)
	}

	// Get Google Calendar ID from environment
	calendarID := os.Getenv("GOOGLE_CALENDAR_ID")
	if calendarID == "" {
		log.Fatalf("GOOGLE_CALENDAR_ID environment variable is not set")
	}

	// Authenticate with Google Calendar
	log.Println("Authenticating with Google Calendar...")
	calendarService, err := getCalendarService()
	if err != nil {
		log.Fatalf("Failed to authenticate with Google Calendar: %v", err)
	}

	// Filter events for next 60 days
	now := time.Now()
	sixtyDaysFromNow := now.AddDate(0, 0, 60)

	var eventsToSync []Event
	for _, event := range events {
		if event.Start.After(now) && event.Start.Before(sixtyDaysFromNow) {
			eventsToSync = append(eventsToSync, event)
		}
	}

	// Sync events with Google Calendar
	log.Printf("Syncing %d events with Google Calendar...", len(eventsToSync))
	if err := syncStravaEvents(eventsToSync, calendarService, calendarID); err != nil {
		log.Fatalf("Failed to sync events with Google Calendar: %v", err)
	}

	log.Println("✓ Google Calendar sync completed successfully!")
}

// testWithSampleData tests the application with sample data from events_raw.json
func testWithSampleData() {
	log.Println("Testing with sample data from events_raw.json...")

	data, err := os.ReadFile("output/validation/events_raw.json")
	if err != nil {
		log.Fatalf("Failed to read sample events file: %v", err)
	}

	var stravaEvents []StravaEvent
	if err := json.Unmarshal(data, &stravaEvents); err != nil {
		log.Fatalf("Failed to parse sample events: %v", err)
	}

	log.Printf("Loaded %d sample events", len(stravaEvents))

	var convertedEvents []Event
	for _, se := range stravaEvents {
		event, err := convertStravaEvent(se)
		if err != nil {
			log.Printf("Failed to convert event %d: %v", se.ID, err)
			continue
		}
		convertedEvents = append(convertedEvents, *event)
	}

	log.Printf("Converted %d events", len(convertedEvents))

	log.Println("Filtering and sorting events...")
	finalEvents := filterAndSortEvents(convertedEvents)

	log.Printf("Saving %d events to %s...", len(finalEvents), eventsFile)
	if err := saveEvents(finalEvents); err != nil {
		log.Fatalf("Failed to save events: %v", err)
	}

	log.Printf("Successfully saved %d events to %s", len(finalEvents), eventsFile)

	for i, event := range finalEvents {
		if i < 5 {
			fmt.Printf("Event %d: %s - %s (%s)\n", event.ID, event.Title,
				event.Start.Format("2006-01-02 15:04"), event.Location)
		}
	}
	if len(finalEvents) > 5 {
		fmt.Printf("... and %d more events\n", len(finalEvents)-5)
	}
}

// filterEvents filters events to only include those from 7 days ago onwards
func filterEvents(events []Event) []Event {
	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -7)

	var filtered []Event
	for _, event := range events {
		if event.Start.After(sevenDaysAgo) {
			filtered = append(filtered, event)
		}
	}

	return filtered
}

// filterAndSortEvents filters and sorts events by start time (newest first)
func filterAndSortEvents(events []Event) []Event {
	filtered := filterEvents(events)

	// Sort events by start time in reverse chronological order (newest first)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Start.After(filtered[j].Start)
	})

	return filtered
}

// loadExistingEvents loads events from the JSON cache file
func loadExistingEvents() ([]Event, error) {
	if _, err := os.Stat(eventsFile); os.IsNotExist(err) {
		return []Event{}, nil
	}

	data, err := os.ReadFile(eventsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read events file: %w", err)
	}

	var events []Event
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("failed to parse events: %w", err)
	}

	// Apply phone number redaction to loaded events
	for i := range events {
		events[i].Description = redactPhoneNumbers(events[i].Description)
	}

	return events, nil
}

// saveEvents saves events to the JSON cache file
func saveEvents(events []Event) error {
	// Ensure output directory exists
	if err := os.MkdirAll("output/events", 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	if err := os.WriteFile(eventsFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write events file: %w", err)
	}

	return nil
}
