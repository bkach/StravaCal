package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	stravaAPIBase  = "https://www.strava.com/api/v3"
	stravaTokenURL = "https://www.strava.com/oauth/token"
)

// getClubID returns the club ID from environment variable
func getClubID() (string, error) {
	clubID := os.Getenv("STRAVA_CLUB_ID")
	if clubID == "" {
		return "", fmt.Errorf("STRAVA_CLUB_ID environment variable is not set")
	}
	return clubID, nil
}

// loadTokens loads Strava OAuth credentials from environment variables
func loadTokens() (*TokenStore, error) {
	clientID := os.Getenv("STRAVA_CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	refreshToken := os.Getenv("REFRESH_TOKEN")

	if clientID == "" || clientSecret == "" || refreshToken == "" {
		return nil, fmt.Errorf("missing required environment variables: STRAVA_CLIENT_ID, CLIENT_SECRET, REFRESH_TOKEN")
	}

	return &TokenStore{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RefreshToken: refreshToken,
	}, nil
}

// refreshTokens refreshes the Strava OAuth access token using the refresh token
func refreshTokens(tokens *TokenStore) error {
	payload := fmt.Sprintf(
		`{"client_id":"%s","client_secret":"%s","grant_type":"refresh_token","refresh_token":"%s"}`,
		tokens.ClientID, tokens.ClientSecret, tokens.RefreshToken,
	)

	resp, err := http.Post(stravaTokenURL, "application/json", strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to refresh tokens: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	tokens.AccessToken = tokenResp.AccessToken
	tokens.RefreshToken = tokenResp.RefreshToken

	return nil
}

// makeAPIRequest makes an authenticated request to the Strava API
// Automatically handles token refresh if the access token has expired
func makeAPIRequest(tokens *TokenStore, url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		log.Println("Access token expired, refreshing...")
		if err := refreshTokens(tokens); err != nil {
			return nil, fmt.Errorf("failed to refresh tokens: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
		resp, err = client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to retry request: %w", err)
		}
	}

	return resp, nil
}

// fetchClubEvents retrieves upcoming events from Strava using the undocumented endpoint
// CRITICAL: Uses upcoming=true parameter which is essential for filtering
// Rate limit impact: ~1 request per 200 events
func fetchClubEvents(tokens *TokenStore) ([]StravaEvent, error) {
	var allEvents []StravaEvent
	page := 1
	perPage := 200 // Conservative to stay under rate limits
	clubID, err := getClubID()
	if err != nil {
		return nil, err
	}

	for {
		// UNDOCUMENTED ENDPOINT - not in official API docs but works
		url := fmt.Sprintf("%s/clubs/%s/group_events?upcoming=true&page=%d&per_page=%d", stravaAPIBase, clubID, page, perPage)

		resp, err := makeAPIRequest(tokens, url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}

		var events []StravaEvent
		if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
			return nil, fmt.Errorf("failed to decode events: %w", err)
		}

		if len(events) == 0 {
			break
		}

		allEvents = append(allEvents, events...)

		if len(events) < perPage {
			break
		}

		page++
		log.Printf("Fetched page %d, got %d events", page-1, len(events))
	}

	return allEvents, nil
}

// redactPhoneNumbers removes phone numbers from text and replaces them with "[Phone Number Redacted]".
// It handles UK mobile, landline, and international formats with optional punctuation, brackets, and spacing.
// Examples matched:
//  - 07801 252100
//  - 07341 081992
//  - 07599393367
//  - +44 7801 252100
//  - +44 (0)7801-252-100
//  - (020) 7946 0018
//  - 0207-946-0018
func redactPhoneNumbers(text string) string {
	// First, clean up any existing redactions (both old and new formats)
	text = regexp.MustCompile(`<Phone Number Redacted>`).ReplaceAllString(text, "[Phone Number Redacted]")
	text = regexp.MustCompile(`\[Phone Number Redacted\]`).ReplaceAllString(text, "[Phone Number Redacted]")

	// Define comprehensive patterns for UK phone numbers
	phonePatterns := []*regexp.Regexp{
		// UK landlines with 4-digit area codes (0xxx xxx xxxx) - MUST come before mobile
		regexp.MustCompile(`\b0[1-9]\d{2}[\s\-]*\d{3}[\s\-]*\d{4}\b`),

		// UK mobile numbers starting with 07
		regexp.MustCompile(`\b07\d{3}[\s\-]*\d{3}[\s\-]*\d{3}\b`),

		// London numbers (020) - 3-digit area code with 4+4 digits
		regexp.MustCompile(`\b020[\s\-]*\d{4}[\s\-]*\d{4}\b`),

		// UK landlines with 3-digit area codes (0xx xxxx xxxx)
		regexp.MustCompile(`\b0[1-9]\d{2}[\s\-]*\d{4}[\s\-]*\d{4}\b`),

		// UK landlines - more flexible (catch remaining patterns)
		regexp.MustCompile(`\b0[1-3]\d{2,3}[\s\-]*\d{3,4}[\s\-]*\d{3,4}\b`),

		// International format +44
		regexp.MustCompile(`\+44\s*(?:\(0\))?\s*[1-9]\d{1,3}[\s\-]*\d{3,4}[\s\-]*\d{3,4}\b`),

		// Bracketed area codes like (020) xxxx xxxx
		regexp.MustCompile(`\([0-9]{3,4}\)[\s\-]*\d{3,4}[\s\-]*\d{4}\b`),

		// Continuous digits starting with 0 (10-11 digits) - catch-all
		regexp.MustCompile(`\b0\d{9,10}\b`),
	}

	result := text
	for _, pattern := range phonePatterns {
		result = pattern.ReplaceAllString(result, "[Phone Number Redacted]")
	}

	return result
}

// convertStravaEvent transforms Strava API response to our standardized Event format
// Key transformations:
// - upcoming_occurrences[0] -> start time
// - Calculates end time (+2 hours estimate since API doesn't provide)
// - Constructs proper Strava URL for the event
// - Redacts phone numbers from description
func convertStravaEvent(se StravaEvent) (*Event, error) {
	if len(se.UpcomingOccurrences) == 0 {
		return nil, fmt.Errorf("no upcoming occurrences for event %d", se.ID)
	}

	// Use the first upcoming occurrence - Strava may have recurring events
	startTime, err := time.Parse("2006-01-02T15:04:05Z", se.UpcomingOccurrences[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse start time: %w", err)
	}

	// Estimate end time as 1 hour after start - Strava doesn't provide end_date_local
	endTime := startTime.Add(1 * time.Hour)

	// Format organizer name from first and last name
	organizer := strings.TrimSpace(se.OrganizingAthlete.FirstName + " " + se.OrganizingAthlete.LastName)

	clubID, err := getClubID()
	if err != nil {
		return nil, err
	}
	event := &Event{
		ID:          se.ID,
		Title:       se.Title,
		Start:       startTime,
		End:         endTime,
		Description: redactPhoneNumbers(se.Description),
		URL:         fmt.Sprintf("https://www.strava.com/clubs/%s/group_events/%d", clubID, se.ID),
		Location:    se.Address,
		Organizer:   organizer,
	}

	return event, nil
}
