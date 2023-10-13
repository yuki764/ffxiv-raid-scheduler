package main

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/sheets/v4"
)

type duty struct {
	Title     string
	Desc      string
	StartTime time.Time
	EndTime   time.Time
}

type spreadsheet struct {
	id string
}

func (s spreadsheet) getThisAndNextMonthSchedules(ctx context.Context, tz *time.Location) ([]duty, time.Time, error) {
	srv, err := sheets.NewService(ctx)
	if err != nil {
		return nil, time.Now(), err
	}

	now := time.Now().In(tz)
	thisYear := now.Year()
	thisMonth := now.Month()
	thisDay := now.Day()

	targetMonths := []struct {
		Year  int
		Month time.Month
	}{}

	spreadsheet, err := srv.Spreadsheets.Get(s.id).Do()
	if err != nil {
		return nil, time.Now(), err
	}

	existThisMonthSheet := false
	// check existence of month's name sheets
	for _, cands := range []time.Time{
		time.Date(thisYear, thisMonth, 1, 1, 1, 1, 1, tz),                          // this month
		time.Date(thisYear, thisMonth, 1, 1, 1, 1, 1, tz).Add(32 * 24 * time.Hour), // next month
	} {
		for _, sheet := range spreadsheet.Sheets {
			if sheet.Properties.Title == fmt.Sprintf("%d月", cands.Month()) {
				targetMonths = append(targetMonths, struct {
					Year  int
					Month time.Month
				}{
					Year:  cands.Year(),
					Month: cands.Month(),
				})

				if thisMonth == cands.Month() {
					existThisMonthSheet = true
				}
			}
		}
	}

	if !existThisMonthSheet {
		slog.Default().Warn("The sheet of this month doesn't exist.")
	}

	var activeSch []duty

	for _, mo := range targetMonths {
		table, err := srv.Spreadsheets.Values.Get(s.id, fmt.Sprintf("%d月!A4:M100", mo.Month)).Do()
		if err != nil {
			return nil, time.Now(), err
		}

		for _, row := range table.Values {
			active := false
			var sch duty
			day := 0

			for i, c := range row {
				switch i {
				case 0:
					if c.(string) == "" {
						sch.Title = "活動"
					} else {
						sch.Title = c.(string)
					}
				case 1:
					day, err = strconv.Atoi(strings.Split(c.(string), "/")[1])
					if err != nil {
						return nil, time.Now(), err
					}
					sch.StartTime = time.Date(mo.Year, mo.Month, day, 21, 50, 0, 0, tz).UTC()
					sch.EndTime = time.Date(mo.Year, mo.Month, day, 23, 50, 0, 0, tz).UTC()
				case 3:
					if c == "有り" {
						active = true
					}
				case 12:
					if notice := c.(string); notice == "" {
						sch.Desc = ""
					} else {
						sch.Desc = "特記事項: " + notice
					}
				}
			}

			if active && sch.StartTime.After(now) {
				activeSch = append(activeSch, sch)
			}
		}
	}

	return activeSch, time.Date(thisYear, thisMonth, thisDay, 21, 50, 0, 0, tz).UTC(), nil
}
