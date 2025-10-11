# 🏃 Malvern Buzzards Running Club Calendar

Never miss a group run! All club events from Strava, automatically synced to your calendar.

---

## 📅 How to Access the Calendar

Choose the option that works best for you:

### Option 1: View Online (Easiest)

Just want to check what's coming up? **No setup required.**

👉 **[Open the Buzzard Run Schedule](https://calendar.google.com/calendar/u/0/embed?src=b46aef20694569443acfef51d9e19e413b17addeb0190f9cefd8dad63ec30e77@group.calendar.google.com&ctz=Europe/London)**

Bookmark this page for quick access anytime.

---

### Option 2: Add to Google Calendar

Get all club runs in your personal Google Calendar. **Updates automatically every 15 minutes.**

👉 **[Click here to add to your Google Calendar](https://calendar.google.com/calendar/u/0?cid=YjQ2YWVmMjA2OTQ1Njk0NDNhY2ZlZjUxZDllMTllNDEzYjE3YWRkZWIwMTkwZjljZWZkOGRhZDYzZWMzMGU3N0Bncm91cC5jYWxlbmRhci5nb29nbGUuY29t)**

That's it! Events will appear alongside your other calendar entries.

---

### Option 3: Subscribe in Apple Calendar, Outlook, or Other Apps

Using a different calendar app? You can subscribe to a live feed that stays up to date automatically.

**Important:** You need to **subscribe** (not import) for automatic updates.

#### 📋 Step 1: Copy this link

```
https://bkach.github.io/StravaCal/calendar.ics
```

#### 🍎 Step 2 (Apple Calendar): Subscribe to the feed

**On Mac:**
1. Open **Calendar**
2. In the menu bar: **File → New Calendar Subscription**
3. Paste the link and click **Subscribe**
4. (Optional) Set refresh to *Every hour* for faster updates

**On iPhone/iPad:**
1. Open **Settings → Calendar → Accounts → Add Account**
2. Tap **Other → Add Subscribed Calendar**
3. Paste the link and tap **Next**, then **Save**

#### 💼 Step 2 (Outlook): Subscribe to the feed

1. Open Outlook Calendar
2. Click **Add Calendar → Subscribe from web**
3. Paste the link and confirm

#### 🖥️ Step 2 (Other apps): Subscribe to the feed

Most calendar apps (Thunderbird, Zoho, Fastmail, etc.) have a **"Subscribe to calendar"** or **"Add calendar by URL"** option. Paste the link above when prompted.

---

⚠️ **Don't manually import the file** — if you download and import the `.ics` file, it won't update when new runs are added. Always use the subscription method.

---

## 🤔 Questions?

**How often does it update?**
Every 15 minutes. New runs on Strava appear in the calendar within 15 minutes.

**What information is included?**
Run title, time, location, organizer, and a link to the event on Strava.

**Will I get notifications?**
That depends on your calendar settings. You can set up reminders in your calendar app just like any other event.

---
---

# 🛠️ For Developers: Project Documentation

This is a Go application that automatically syncs Strava club events to Google Calendar and generates `.ics` calendar files. It runs every 15 minutes via GitHub Actions.

## ⚙️ Setup – Use This for Your Own Club

Want to generate a live calendar like this for your own Strava club? Do so by cloning this repo and following these (admittedly very sparse) instructions.

### Required Environment Variables

These are always required:
```bash
export STRAVA_CLIENT_ID="your_strava_client_id"
export STRAVA_CLUB_ID="your_strava_club_id"
export CLIENT_SECRET="your_strava_client_secret"
export REFRESH_TOKEN="your_strava_refresh_token"
```

### Optional: Google Calendar Sync

To sync to Google Calendar (optional for ICS-only generation), add:
```bash
export GOOGLE_CALENDAR_ID="your_google_calendar_id"
```

And provide service account credentials (choose one method):
```bash
# Method 1: Environment variable (raw JSON string)
export GOOGLE_SERVICE_ACCOUNT='{"type":"service_account","project_id":"...",...}'

# Method 2: File (recommended for local development)
# Place service-account.json in the project root
```

## Commands

```bash
go run .       # Full sync: Fetch from Strava → Google Calendar → ICS
go run . ics   # Generate ICS file only from cached events
go run . gcal  # Sync to Google Calendar only from cached events
go run . test  # Test with sample data from output/validation/events_raw.json
```

## GitHub Actions

Runs every 15 minutes to sync events to Google Calendar and generate ICS file. See [`.github/workflows/update-calendar.yml`](.github/workflows/update-calendar.yml) for the workflow configuration.

Set up:
1. Enable GitHub Pages (Settings → Pages → Source: GitHub Actions) - Only needed if you want to host the ICS file publicly
2. Add repository secrets (Settings → Secrets and variables → Actions):
   - `STRAVA_CLIENT_ID` - Your Strava OAuth client ID
   - `STRAVA_CLUB_ID` - Your Strava club ID
   - `CLIENT_SECRET` - Your Strava OAuth client secret
   - `REFRESH_TOKEN` - Your Strava OAuth refresh token
   - `GOOGLE_CALENDAR_ID` - Target Google Calendar ID
   - `SERVICE_ACCOUNT_B64` - Base64 encoded Google service account JSON (see workflow for how it's decoded)

## Project Structure

```
main.go     - Entry point and command handling
types.go    - Shared data structures
strava.go   - Strava API integration (OAuth, event fetching, phone number redaction)
gcal.go     - Google Calendar sync (create, update, delete events)
ics.go      - ICS calendar file generation (RFC 5545 format)
```

## Output

- `output/events/events.json` - Event data cache (all events from last 7 days)
- `output/calendar.ics` - iCalendar file for calendar apps (next 60 days)

## Features

- **Google Calendar sync**: Automatically creates, updates, and deletes events in Google Calendar
- **Automatic updates**: GitHub Actions syncs calendar every 15 minutes
- **ICS file generation**: Downloadable calendar file for any calendar app
- **Phone number redaction**: Automatically removes phone numbers from event descriptions
- **Timezone handling**: All times properly converted to Europe/London (BST/GMT)
- **Event filtering**: Syncs next 60 days, caches last 7 days of events
- **Smart sync**: Only updates changed events, removes deleted ones

## API

Uses `GET /api/v3/clubs/{id}/group_events?upcoming=true` (undocumented Strava endpoint)

**Rate limits**: 100 req/15min, 1000 req/day
