package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

// https://discord.com/developers/docs/resources/guild-scheduled-event
type discordEvent struct {
	Id                 string `json:"id,omitempty"`
	Name               string `json:"name"`
	Description        string `json:"description,omitempty"`
	ScheduledStartTime string `json:"scheduled_start_time"`
	ScheduledEndTime   string `json:"scheduled_end_time"`
	EntityType         int    `json:"entity_type"`
	GuildId            string `json:"guild_id,omitempty"`
	ChannelId          string `json:"channel_id"`
	PrivacyLevel       int    `json:"privacy_level"`
}

const (
	DiscordISO8601 = "2006-01-02T15:04:05-07:00"
)

type discordHttp struct {
	authHeader            string
	guildId               string
	eventChannelId        string
	notificationChannelId string
}

func (d discordHttp) requestEventsApi(method string, suffix string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, "https://discordapp.com/api/guilds/"+d.guildId+"/scheduled-events"+suffix, body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", d.authHeader)
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	// wait in order to avoid rate limiting
	after, err := strconv.Atoi(resp.Header.Get("X-Ratelimit-Reset-After"))
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Duration(after) * time.Second)

	return resp, nil
}

func (d discordHttp) notifyEvent(evt discordEvent) error {
	msg := &struct {
		Content string `json:"content"`
	}{
		Content: fmt.Sprintf("今日は%s日クポ。 %s\nhttps://discord.com/events/%s/%s\n", evt.Name, evt.Description, evt.GuildId, evt.Id),
	}
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://discordapp.com/api/channels/"+d.notificationChannelId+"/messages", b)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", d.authHeader)
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	slog.Default().Info("Notified the event with the result "+`"`+resp.Status+`".`, "event", evt)

	return nil
}
