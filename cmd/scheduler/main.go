package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"time"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		// https://cloud.google.com/logging/docs/structured-logging
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.MessageKey {
				a.Key = "message"
			}
			if a.Key == slog.LevelKey {
				a.Key = "severity"
				level := a.Value.Any().(slog.Level)
				if level == slog.LevelWarn {
					a.Value = slog.StringValue("WARNING")
				}
			}
			return a
		},
	})))

	spreadsheet := &spreadsheet{
		id: os.Getenv("SPREADSHEET_ID"),
	}

	discord := &discordHttp{
		authHeader:            "Bot " + os.Getenv("DISCORD_BOT_TOKEN"),
		guildId:               os.Getenv("DISCORD_GUILD_ID"),
		eventChannelId:        os.Getenv("DISCORD_EVENT_CHANNEL_ID"),
		notificationChannelId: os.Getenv("DISCORD_NOTIFICATION_CHANNEL_ID"),
	}

	ical := &ical{
		gcsBucket: os.Getenv("ICAL_GCS_BUCKET"),
		gcsPath:   os.Getenv("ICAL_GCS_PATH"),
		tag:       os.Getenv("ICAL_TAG"),
	}

	tz, err := time.LoadLocation(os.Getenv("TZ"))
	if err != nil {
		slog.Default().Error("Failed to load time location.", "error", err)
		panic(err)
	}

	ctx := context.Background()

	existingEvents := []discordEvent{}
	{
		resp, err := discord.requestEventsApi("GET", "", nil)
		if err != nil {
			slog.Default().Error("Failed to get events from Discord.", "error", err)
			panic(err)
		}

		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(&existingEvents); err != nil {
			slog.Default().Error("Failed to decode json of events from Discord.", "error", err)
			panic(err)
		}
	}

	activeTimestamps := []string{}

	thisAndNextMonthSchedules, todayTime, err := spreadsheet.getThisAndNextMonthSchedules(ctx, tz)
	if err != nil {
		slog.Default().Error("Failed to get schedules from Spreadsheet.", "error", err)
		panic(err)
	}

append:
	for _, sch := range thisAndNextMonthSchedules {
		slog.Default().Debug("Dump schedule struct.", "schedule", sch)
		timestamp := sch.StartTime.Format(DiscordISO8601)
		activeTimestamps = append(activeTimestamps, timestamp)

		for _, evt := range existingEvents {
			if timestamp == evt.ScheduledStartTime {
				slog.Default().Info("The event on " + timestamp + " already exists.")
				if timestamp == todayTime.Format(DiscordISO8601) {
					if err := discord.notifyEvent(evt); err != nil {
						slog.Default().Error("Failed to notify event to Discord channel.", "error", err)
						panic(err)
					}
				}
				continue append
			}
		}

		slog.Default().Info("Appending the event on " + timestamp + ".")

		b := new(bytes.Buffer)
		if err := json.NewEncoder(b).Encode(discordEvent{
			Name:               sch.Title,
			Description:        sch.Desc,
			ScheduledStartTime: timestamp,
			ScheduledEndTime:   sch.EndTime.Format(DiscordISO8601),
			EntityType:         2, // means "VOICE"
			ChannelId:          discord.eventChannelId,
			PrivacyLevel:       2, // means "GUILD_ONLY"
		}); err != nil {
			slog.Default().Error("Failed to encode Discord event JSON.", "error", err)
			panic(err)
		}

		resp, err := discord.requestEventsApi("POST", "", b)
		if err != nil {
			slog.Default().Error("Failed to post a event on "+timestamp+" to Discord.", "error", err)
			panic(err)
		}

		slog.Default().Info("Appended the event with the result " + `"` + resp.Status + `".`)

		if timestamp == todayTime.Format(DiscordISO8601) {
			evt := discordEvent{}

			defer resp.Body.Close()
			if err := json.NewDecoder(resp.Body).Decode(&evt); err != nil {
				slog.Default().Error("Failed to decode Discord event JSON.", "error", err)
				panic(err)
			}

			if err := discord.notifyEvent(evt); err != nil {
				slog.Default().Error("Failed to notify event to Discord channel.", "error", err)
				panic(err)
			}
		}
	}

cleanup:
	for _, existingEvt := range existingEvents {
		for _, timestamp := range activeTimestamps {
			if timestamp == existingEvt.ScheduledStartTime {
				continue cleanup
			}
		}

		slog.Default().Info("Cleaning up the event " + existingEvt.Id)

		// if an existing event is not active (fetched from Spreadsheet currently), deleting event
		resp, err := discord.requestEventsApi("DELETE", "/"+existingEvt.Id, nil)
		if err != nil {
			slog.Default().Error("Failed to delete the event "+existingEvt.Id+" from Discord.", "error", err)
			panic(err)
		}

		slog.Default().Info("Cleaned up the event "+existingEvt.Id+" with the result "+`"`+resp.Status+`".`, "event", existingEvt)
	}

	// export ics (iCal) file in GCS
	time.Sleep(10 * time.Second)

	events := []discordEvent{}
	{
		resp, err := discord.requestEventsApi("GET", "", nil)
		if err != nil {
			slog.Default().Error("Failed to get events from Discord.", "error", err)
			panic(err)
		}

		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
			slog.Default().Error("Failed to decode Discord events JSON.", "error", err)
			panic(err)
		}
	}

	icalEvents := []icalEvent{}
	for _, evt := range events {
		ie, err := convertIcalEvent(evt, tz)
		if err != nil {
			slog.Default().Error("Failed convert a event from Discord format to ical format.", "error", err)
			panic(err)
		}
		icalEvents = append(icalEvents, *ie)
	}

	icalUrl, err := ical.updateGcsObject(ctx, icalEvents)
	if err != nil {
		slog.Default().Error("Failed to upload a ical file to GCS.", "error", err)
		panic(err)
	}

	slog.Default().Info("Uploaded ical file to GCS: "+icalUrl, "url", icalUrl)
}
