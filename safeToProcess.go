package main

import (
	"regexp"
	"time"
)

func isSafeToProcess(filename string) bool {
	namePattern := regexp.MustCompile(`([0-9]{4}-[0-9]{2}-[0-9]{2})T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3}-.+\.log`)
	lfs := namePattern.FindStringSubmatch(filename)
	if len(lfs) < 2 {
		return false
	}

	d, err := time.Parse("2006-01-02", lfs[1])
	if err != nil {
		return false
	}

	// Reference time: midnight of the day 2 hours ago in UTC
	ref := time.Now().UTC().Add(-2 * time.Hour)
	today := time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, time.UTC)

	return d.Before(today)
}
