package utils

import "time"

func ParseTimestampToIsoFormat(timestamp string) (string, error) {
	// Parse the timestamp into time.Time
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return "", err
	}

	// Convert the time.Time into ISO8601 format
	return t.Format(time.RFC3339), nil
}
