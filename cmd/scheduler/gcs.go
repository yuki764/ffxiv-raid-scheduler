package main

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/base32"
	"strings"
	"time"

	"text/template"

	"cloud.google.com/go/storage"
)

type icalEvent struct {
	Uid         string
	Summary     string
	Description string
	StartTime   string
	EndTime     string
	Timezone    string
	Url         string
}

const (
	IcalDateFormat = "20060102T150405"
)

//go:embed templates/schedule.ics.go.tpl
var tplFile []byte

type ical struct {
	tag       string
	gcsBucket string
	gcsPath   string
}

func (i ical) updateGcsObject(ctx context.Context, events []icalEvent) (string, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", err
	}
	w := client.Bucket(i.gcsBucket).Object(i.gcsPath).NewWriter(ctx)
	w.ObjectAttrs.ContentType = "text/calendar"

	tpl, err := template.New("ics").Parse(string(tplFile))
	if err != nil {
		return "", err
	}

	defer w.Close()
	if err := tpl.Execute(w, map[string]interface{}{
		"icalTag": i.tag,
		"events":  events,
	}); err != nil {
		return "", err
	}

	return "https://storage.googleapis.com/" + i.gcsBucket + "/" + i.gcsPath, nil
}

func convertIcalEvent(evt discordEvent, tz *time.Location) (*icalEvent, error) {
	start, err := time.Parse(DiscordISO8601, evt.ScheduledStartTime)
	if err != nil {
		return nil, err
	}
	end, err := time.Parse(DiscordISO8601, evt.ScheduledEndTime)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256([]byte("ffxiv-raid-scheduler" + evt.Name + evt.ScheduledStartTime))
	uid := strings.ToLower(base32.HexEncoding.WithPadding(base32.NoPadding).EncodeToString(hash[:]))

	return &icalEvent{
		Uid:         uid,
		Summary:     evt.Name,
		Description: evt.Description,
		StartTime:   start.In(tz).Format(IcalDateFormat),
		EndTime:     end.In(tz).Format(IcalDateFormat),
		Timezone:    tz.String(),
		Url:         "https://discord.com/events/" + evt.GuildId + "/" + evt.Id,
	}, nil
}
