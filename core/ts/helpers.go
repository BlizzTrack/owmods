package ts

import "time"

func CurrentTimeToUnix() (time.Time, int64) {
	now := time.Now()
	return now, now.UnixNano() / 1000000
}
