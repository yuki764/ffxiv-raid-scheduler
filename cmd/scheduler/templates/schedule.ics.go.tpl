BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Setuu//ffxiv-raid-scheduler-{{ .icalTag }} v0.1//JA
BEGIN:VTIMEZONE
TZID:Asia/Tokyo
BEGIN:STANDARD
DTSTART:19390101T000000
TZOFFSETFROM:+0900
TZOFFSETTO:+0900
END:STANDARD
END:VTIMEZONE
{{- range .events }}
BEGIN:VEVENT
UID:{{ .Uid }}@neigepluie.net
CATEGORIES:Raid
SUMMARY:[{{ $.icalTag }}] {{ .Summary }}
DESCRIPTION:{{ .Description }}
TRANSP:OPAQUE
DTSTART;TZID={{ .Timezone }}:{{ .StartTime }}
DTEND;TZID={{ .Timezone }}:{{ .EndTime }}
LOCATION={{ .Url }}
END:VEVENT
{{- end }}
END:VCALENDAR
