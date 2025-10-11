package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// generateICS creates an iCalendar (ICS) format string from a list of events
func generateICS(events []Event) string {
	var icsContent strings.Builder

	// ICS header
	icsContent.WriteString("BEGIN:VCALENDAR\r\n")
	icsContent.WriteString("VERSION:2.0\r\n")
	icsContent.WriteString("PRODID:-//StravaCal//Strava Club Events//EN\r\n")
	icsContent.WriteString("CALSCALE:GREGORIAN\r\n")
	icsContent.WriteString("METHOD:PUBLISH\r\n")
	icsContent.WriteString("X-WR-CALNAME:Malvern Buzzards Running Club\r\n")
	icsContent.WriteString("X-WR-CALDESC:Club running events from Strava\r\n")

	// Add timezone definition for Europe/London
	icsContent.WriteString("BEGIN:VTIMEZONE\r\n")
	icsContent.WriteString("TZID:Europe/London\r\n")
	icsContent.WriteString("BEGIN:DAYLIGHT\r\n")
	icsContent.WriteString("DTSTART:20070325T010000\r\n")
	icsContent.WriteString("RRULE:FREQ=YEARLY;BYMONTH=3;BYDAY=-1SU\r\n")
	icsContent.WriteString("TZOFFSETFROM:+0000\r\n")
	icsContent.WriteString("TZOFFSETTO:+0100\r\n")
	icsContent.WriteString("TZNAME:BST\r\n")
	icsContent.WriteString("END:DAYLIGHT\r\n")
	icsContent.WriteString("BEGIN:STANDARD\r\n")
	icsContent.WriteString("DTSTART:20071028T020000\r\n")
	icsContent.WriteString("RRULE:FREQ=YEARLY;BYMONTH=10;BYDAY=-1SU\r\n")
	icsContent.WriteString("TZOFFSETFROM:+0100\r\n")
	icsContent.WriteString("TZOFFSETTO:+0000\r\n")
	icsContent.WriteString("TZNAME:GMT\r\n")
	icsContent.WriteString("END:STANDARD\r\n")
	icsContent.WriteString("END:VTIMEZONE\r\n")

	// Add events
	for _, event := range events {
		icsContent.WriteString("BEGIN:VEVENT\r\n")

		// Unique ID
		icsContent.WriteString(fmt.Sprintf("UID:%d@strava.com\r\n", event.ID))

		// Date/time stamps (convert to Europe/London timezone)
		london, _ := time.LoadLocation("Europe/London")
		startLocal := event.Start.In(london).Format("20060102T150405")
		endLocal := event.End.In(london).Format("20060102T150405")
		nowUTC := time.Now().UTC().Format("20060102T150405Z")

		icsContent.WriteString(fmt.Sprintf("DTSTART;TZID=Europe/London:%s\r\n", startLocal))
		icsContent.WriteString(fmt.Sprintf("DTEND;TZID=Europe/London:%s\r\n", endLocal))
		icsContent.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", nowUTC))

		// Event details
		icsContent.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeICSText(event.Title)))

		// Description with details including sync timestamp in Europe/London timezone
		now := time.Now().In(london)
		syncTime := now.Format("Mon, 2 Jan @ 3:04 PM")
		clubID, err := getClubID()
		if err != nil {
			clubID = "unknown"
		}
		description := fmt.Sprintf("Leader: %s\n\nLocation: %s\n\n%s\n\nView on Strava: %s\n\nSynced from Strava Club %s on %s",
			event.Organizer,
			event.Location,
			event.Description,
			event.URL,
			clubID,
			syncTime)
		icsContent.WriteString(formatICSProperty("DESCRIPTION", description))

		// Add HTML version for better Google Calendar display
		htmlDescription := fmt.Sprintf("<p><strong>Leader:</strong> %s</p><p><strong>Location:</strong> %s</p><p>%s</p><p><strong>View on Strava:</strong> <a href=\"%s\">%s</a></p><p><strong>Synced from Strava Club %s on:</strong> %s</p>",
			strings.ReplaceAll(event.Organizer, "\n", "<br>"),
			strings.ReplaceAll(event.Location, "\n", "<br>"),
			strings.ReplaceAll(event.Description, "\n", "<br>"),
			event.URL,
			event.URL,
			clubID,
			syncTime)
		icsContent.WriteString(formatICSProperty("X-ALT-DESC;FMTTYPE=text/html", htmlDescription))

		// Location
		if event.Location != "" {
			icsContent.WriteString(fmt.Sprintf("LOCATION:%s\r\n", escapeICSText(event.Location)))
		}

		// URL
		icsContent.WriteString(fmt.Sprintf("URL:%s\r\n", event.URL))

		// Category
		icsContent.WriteString("CATEGORIES:Running,Club Event\r\n")

		icsContent.WriteString("END:VEVENT\r\n")
	}

	// ICS footer
	icsContent.WriteString("END:VCALENDAR\r\n")
	icsContent.WriteString("\n")

	return icsContent.String()
}

// stripHTML removes HTML tags from text for Apple Calendar compatibility
func stripHTML(input string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(input, "")

	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&apos;", "'")

	return text
}

// escapeICSText escapes special characters per RFC 5545 for Apple Calendar compatibility
func escapeICSText(s string) string {
	// Must escape in this order to avoid double-escaping
	s = strings.ReplaceAll(s, "\\", "\\\\")  // Backslash must be first
	s = strings.ReplaceAll(s, ";", "\\;")    // Semicolon
	s = strings.ReplaceAll(s, ",", "\\,")    // Comma
	s = strings.ReplaceAll(s, "\r\n", "\\n") // CRLF to literal \n
	s = strings.ReplaceAll(s, "\n", "\\n")   // LF to literal \n
	s = strings.ReplaceAll(s, "\r", "\\n")   // CR to literal \n
	return s
}

// foldLine wraps long lines per RFC 5545 (max 75 octets per line)
// Apple Calendar strictly requires this for proper display
func foldLine(text string) string {
	const maxLen = 75

	if len(text) <= maxLen {
		return text
	}

	var result strings.Builder
	for len(text) > maxLen {
		result.WriteString(text[:maxLen])
		result.WriteString("\r\n ") // Continuation: CRLF + space
		text = text[maxLen:]
	}
	result.WriteString(text)

	return result.String()
}

// formatICSProperty formats a property with proper escaping and line folding for Apple Calendar
func formatICSProperty(property, value string) string {
	// Strip HTML for Apple Calendar compatibility
	cleaned := stripHTML(value)
	// Escape special characters per RFC 5545
	escaped := escapeICSText(cleaned)
	// Combine property name and value
	line := property + ":" + escaped
	// Fold long lines (max 75 octets)
	folded := foldLine(line)
	return folded + "\r\n"
}
