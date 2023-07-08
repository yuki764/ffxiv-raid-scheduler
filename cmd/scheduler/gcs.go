package main

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/base32"
	"fmt"
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

func updateICalendar(ctx context.Context, bucket string, path string, events []icalEvent, icalTag string) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	bkt := client.Bucket(bucket)
	obj := bkt.Object(path)
	w := obj.NewWriter(ctx)
	w.ObjectAttrs.ContentType = "text/calendar"
	defer w.Close()

	tpl, err := template.New("ics").Parse(string(tplFile))
	if err != nil {
		return err
	}
	if err := tpl.Execute(w, map[string]interface{}{
		"icalTag": icalTag,
		"events":  events,
	}); err != nil {
		return err
	}

	fmt.Printf("uploaded ical to: https://storage.googleapis.com/%s/%s\n", bucket, path)

	return nil
}

func convertIcalEvent(evt event, tz *time.Location) (*icalEvent, error) {
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
		Url:         fmt.Sprintf("https://discord.com/events/%s/%s", evt.GuildId, evt.Id),
	}, nil
}
