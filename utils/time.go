package utils

import (
	"fmt"
	"strings"
	"time"
)

const TimeFormat = "2006-01-02 15:04:05"

func ConvertStr2Time(timestamp, format string) (t time.Time, err error) {
	if !strings.Contains(format, "%Y") || !strings.Contains(format, "%m") || !strings.Contains(format, "%d") {
		return
	}
	if !strings.Contains(format, "%H") {
		timestamp += "00"
		format += "%H"
	}

	if !strings.Contains(format, "%M") {
		timestamp += "00"
		format += "%M"
	}

	if !strings.Contains(format, "%S") {
		timestamp += "00"
		format += "%S"
	}
	t, err = time.ParseInLocation(TimeFormat, timestamp, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time %q with format %q: %w", timestamp, format, err)
	}
	return t, nil
}
