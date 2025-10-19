package main

import (
	"time"

	"github.com/spf13/viper"
)

func isSafeToProcess(filename string) bool {
	lfs := logFilenamePattern.FindStringSubmatch(filename)
	if len(lfs) < 2 {
		return false
	}

	d, err := time.Parse("2006-01-02", lfs[1])
	if err != nil {
		return false
	}

	// Reference time: midnight of the day 2 hours ago in UTC
	ref, err := time.Parse("2006-01-02", viper.GetString("day-until"))
	if err != nil {
		return false
	}
	today := time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, time.UTC)

	return d.Before(today)
}
