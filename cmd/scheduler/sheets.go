package main

import (
	"context"
	"fmt"
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

func getThisAndNextMonthSchedules(ctx context.Context, spreadsheetId string, tz *time.Location) ([]duty, time.Time) {
	srv, err := sheets.NewService(ctx)
	if err != nil {
		panic(err)
	}

	now := time.Now().In(tz)
	thisYear := now.Year()
	thisMonth := now.Month()
	thisDay := now.Day()

	targetMonths := []struct {
		Year  int
		Month int
	}{}

	spreadsheet, err := srv.Spreadsheets.Get(spreadsheetId).Do()
	if err != nil {
		panic(err)
	}

	// check existence of month's name sheets
	for _, cands := range []time.Time{
		time.Date(thisYear, thisMonth, 1, 1, 1, 1, 1, tz),                          // this month
		time.Date(thisYear, thisMonth, 1, 1, 1, 1, 1, tz).Add(32 * 24 * time.Hour), // next month
	} {
		for _, sheet := range spreadsheet.Sheets {
			if sheet.Properties.Title == fmt.Sprint(int(cands.Month()))+"月" {
				targetMonths = append(targetMonths, struct {
					Year  int
					Month int
				}{
					Year:  cands.Year(),
					Month: int(cands.Month()),
				})
			}
		}
	}

	var activeSch []duty

	for _, month := range targetMonths {
		table, err := srv.Spreadsheets.Values.Get(spreadsheetId, fmt.Sprint(month.Month)+"月!A4:M100").Do()
		if err != nil {
			panic(err)
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
						panic(err)
					}
					sch.StartTime = time.Date(month.Year, time.Month(month.Month), day, 21, 50, 0, 0, tz).UTC()
					sch.EndTime = time.Date(month.Year, time.Month(month.Month), day, 23, 50, 0, 0, tz).UTC()
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

	return activeSch, time.Date(thisYear, thisMonth, thisDay, 21, 50, 0, 0, tz).UTC()
}
