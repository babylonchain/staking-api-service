package utils

import "time"

func ParseTimestampToIsoFormat(epochtime int64) string {
	// Convert the int64 epoch time to a time.Time object
	t := time.Unix(epochtime, 0)
	// Convert the time.Time object to a string in ISO8601 format
	return t.Format(time.RFC3339)
}
