package storage

import (
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

// ParseTimezoneOffset converts string offsets like "-0500", "+02:00", or "Z" into a pgtype.Interval
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

	totalMicros := sign * ((hours * 3600) + (mins * 60)) * 1000000
	return pgtype.Interval{Microseconds: totalMicros, Valid: true}
}
