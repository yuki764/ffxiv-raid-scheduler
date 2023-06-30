package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type duty struct {
	Date  time.Time
	Title string
	Desc  string
}

// https://discord.com/developers/docs/resources/guild-scheduled-event
type event struct {
	Id                 string `json:"id,omitempty"`
	Name               string `json:"name"`
	Description        string `json:"description,omitempty"`
	ScheduledStartTime string `json:"scheduled_start_time"`
	EntityType         int    `json:"entity_type"`
	GuildId            string `json:"guild_id,omitempty"`
	ChannelId          string `json:"channel_id"`
	PrivacyLevel       int    `json:"privacy_level"`
}

const (
	DiscordISO8601 = "2006-01-02T15:04:05-07:00"
)

func main() {
	scheduledEventsUrl := "https://discordapp.com/api/guilds/" + os.Getenv("DISCORD_GUILD_ID") + "/scheduled-events"
	discodeAuthHeader := "Bot " + os.Getenv("DISCORD_BOT_TOKEN")
	discordEventChannelId := os.Getenv("DISCORD_EVENT_CHANNEL_ID")
	discordNotificationChannelId := os.Getenv("DISCORD_NOTIFICATION_CHANNEL_ID")

	spreadsheetId := os.Getenv("SPREADSHEET_ID")

	ctx := context.Background()

	existingEvents := []event{}

	req, err := http.NewRequest("GET", scheduledEventsUrl, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Authorization", discodeAuthHeader)
	req.Header.Add("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&existingEvents); err != nil {
		panic(err)
	}

	activeTimestamps := []string{}

	thisAndNextMonthSchedules, todayTime := getThisAndNextMonthSchedules(ctx, spreadsheetId)

append:
	for _, sch := range thisAndNextMonthSchedules {
		fmt.Printf("%#v\n", sch)
		timestamp := sch.Date.Format(DiscordISO8601)
		activeTimestamps = append(activeTimestamps, timestamp)

		for _, evt := range existingEvents {
			if timestamp == evt.ScheduledStartTime {
				fmt.Println("exists:", timestamp)
				if timestamp == todayTime.Format(DiscordISO8601) {
					if err := notifyEventToDiscordChannel(discodeAuthHeader, discordNotificationChannelId, evt); err != nil {
						panic(err)
					}
				}
				continue append
			}
		}

		fmt.Println("append:", timestamp)

		b := new(bytes.Buffer)
		err := json.NewEncoder(b).Encode(event{
			Name:               sch.Title,
			Description:        sch.Desc,
			ScheduledStartTime: timestamp,
			EntityType:         2, // means "VOICE"
			ChannelId:          discordEventChannelId,
			PrivacyLevel:       2, // means "GUILD_ONLY"
		})
		if err != nil {
			panic(err)
		}
		req, err := http.NewRequest("POST", scheduledEventsUrl, b)
		if err != nil {
			panic(err)
		}
		req.Header.Add("Authorization", discodeAuthHeader)
		req.Header.Add("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			panic(err)
		}

		fmt.Println("append:", resp.Status)

		// wait in order to avoid rate limiting
		time.Sleep(3 * time.Second)

		if timestamp == todayTime.Format(DiscordISO8601) {
			evt := event{}
			defer resp.Body.Close()
			if err := json.NewDecoder(resp.Body).Decode(&evt); err != nil {
				panic(err)
			}

			if err := notifyEventToDiscordChannel(discodeAuthHeader, discordNotificationChannelId, evt); err != nil {
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

		fmt.Println("cleanup:", existingEvt.Id)

		// if an existing event is not active (fetched from Spreadsheet currently), deleting event
		req, err := http.NewRequest("DELETE", scheduledEventsUrl+"/"+existingEvt.Id, nil)
		if err != nil {
			panic(err)
		}
		req.Header.Add("Authorization", discodeAuthHeader)
		req.Header.Add("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			panic(err)
		}

		fmt.Println("cleanup:", resp.Status)

		// wait in order to avoid rate limiting
		time.Sleep(3 * time.Second)
	}
}

func notifyEventToDiscordChannel(authHeader string, notificationChannelId string, evt event) error {
	discordMsg := &struct {
		Content string `json:"content"`
	}{
		Content: fmt.Sprintf("今日は%s日クポ。 %s\nhttps://discord.com/events/%s/%s\n", evt.Name, evt.Description, evt.GuildId, evt.Id),
	}
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(discordMsg)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://discordapp.com/api/channels/"+notificationChannelId+"/messages", b)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", authHeader)
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	fmt.Println("notify:", resp.Status)

	return nil
}
