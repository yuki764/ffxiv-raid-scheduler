package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/sheets/v4"
)

func getThisAndNextMonthSchedules(ctx context.Context, spreadsheetId string) ([]duty, time.Time) {
	srv, err := sheets.NewService(ctx)
	if err != nil {
		panic(err)
	}

	tz, err := time.LoadLocation(os.Getenv("TZ"))
	if err != nil {
		panic(err)
	}

	now := time.Now().In(tz)
	thisYear := now.Year()
	thisMonth := now.Month()
	thisDay := now.Day()

	var activeSch []duty

	thisMonthSch, err := srv.Spreadsheets.Values.Get(spreadsheetId, fmt.Sprint(int(thisMonth))+"月!A4:M100").Do()
	if err != nil {
		panic(err)
	}

	for _, row := range thisMonthSch.Values {
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
				sch.Date = time.Date(thisYear, thisMonth, day, 21, 50, 0, 0, tz).UTC()
			case 3:
				if c == "有り" {
					active = true
				}
			case 12:
				sch.Desc = c.(string)
			}
		}

		if active && day >= thisDay {
			activeSch = append(activeSch, sch)
		}
	}

	nextMonth := (thisMonth + 1) % 12
	nextMonthSch, err := srv.Spreadsheets.Values.Get(spreadsheetId, fmt.Sprint(int(nextMonth))+"月!A4:M100").Do()
	if err != nil {
		panic(err)
	}
	for _, row := range nextMonthSch.Values {
		active := false
		var sch duty

		for i, c := range row {
			switch i {
			case 0:
				if c.(string) == "" {
					sch.Title = "活動"
				} else {
					sch.Title = c.(string)
				}
			case 1:
				day, err := strconv.Atoi(strings.Split(c.(string), "/")[1])
				if err != nil {
					panic(err)
				}
				sch.Date = time.Date(thisYear+((int(thisMonth)+1)/12), nextMonth, day, 21, 50, 0, 0, tz).UTC()
			case 3:
				if c == "有り" {
					active = true
				}
			case 12:
				if c.(string) != "" {
					sch.Desc = "特記事項: " + c.(string)
				}
			}
		}

		if active {
			activeSch = append(activeSch, sch)
		}
	}

	return activeSch, time.Date(thisYear, thisMonth, thisDay, 21, 50, 0, 0, tz).UTC()
}
