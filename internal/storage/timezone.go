package storage

import (
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ParseTimezoneOffset converts WHOOP API timezone offset strings (e.g. "-0500",
// "+02:00", "Z", "") into a pgtype.Interval. Returns an invalid interval for
// malformed inputs rather than an error, since timezone data is non-critical.
func ParseTimezoneOffset(offsetStr string) pgtype.Interval {
	if offsetStr == "" || offsetStr == "Z" {
		return pgtype.Interval{Microseconds: 0, Valid: true}
	}

	offsetStr = strings.ReplaceAll(offsetStr, ":", "")
	if len(offsetStr) != 5 {
		return pgtype.Interval{Valid: false}
	}

	sign := int64(1)
	if offsetStr[0] == '-' {
		sign = -1
	} else if offsetStr[0] != '+' {
		return pgtype.Interval{Valid: false}
	}

	hours, err := strconv.ParseInt(offsetStr[1:3], 10, 64)
	if err != nil {
		return pgtype.Interval{Valid: false}
	}

	mins, err := strconv.ParseInt(offsetStr[3:5], 10, 64)
	if err != nil {
		return pgtype.Interval{Valid: false}
	}

	duration := time.Duration(hours)*time.Hour + time.Duration(mins)*time.Minute
	totalMicros := sign * duration.Microseconds()
	return pgtype.Interval{Microseconds: totalMicros, Valid: true}
}
