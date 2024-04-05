package utils

import "time"

func ParseTimestampToIsoFormat(timestamp string) (string, error) {
	var t time.Time
	// Parse the timestamp into time.Time
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		// TODO: workaround for the timestamp format issue, we will use epoch time between services
		// Indexer is sending some other timestamp format which is not RFC3339
		layout := "2006-01-02 15:04:05 -0700 MST"
		tInMST, errMstTime := time.Parse(layout, timestamp)
		if errMstTime != nil {
			return "", err
		}
		return tInMST.Format(time.RFC3339), nil
	}

	// Convert the time.Time into ISO8601 format
	return t.Format(time.RFC3339), nil
}
