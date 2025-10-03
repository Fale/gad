package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/viper"
)

func importLogFileContent(s3c *s3.Client, filename string) error {
	// Open the remote file
	namePattern := regexp.MustCompile(`([0-9]{4})-([0-9]{2})-([0-9]{2})T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3}-.+\.log`)
	lfs := namePattern.FindStringSubmatch(filename)
	if len(lfs) < 4 {
		return fmt.Errorf("file not matching expected pattern")
	}
	b := viper.GetString("bucket")
	resp, err := s3c.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: &b,
		Key:    &filename,
	})
	if err != nil {
		return fmt.Errorf("unable to open item %q, %v", filename, err)
	}
	defer resp.Body.Close()

	// Open files cache
	files := map[string]*os.File{}
	closeAll := func() {
		for _, f := range files {
			if f == nil {
				continue
			}
			if err := f.Sync(); err != nil {
				fmt.Printf("syncing file %s: %v", f.Name(), err)
			}
			if err := f.Close(); err != nil {
				fmt.Printf("closing file %s: %v", f.Name(), err)
			}
		}
	}
	defer closeAll()

	// Scanner for the log lines
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)

	// Regexp for dates identification
	dateCol := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

	for sc.Scan() {
		line := sc.Text()
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		first := fields[0]
		if len(first) >= 10 {
			first = first[:10]
		}
		if !dateCol.MatchString(first) {
			fmt.Printf("line not sortable, skipping: %s", line)
			continue
		}

		yyyy, mm := first[:4], first[5:7]
		dir := path.Join(viper.GetString("logs-folder"), yyyy, mm)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("folder creation %q: %w", dir, err)
		}
		fp := path.Join(dir, first+".log")

		f := files[first]
		if f == nil {
			var err error
			f, err = os.OpenFile(fp, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err != nil {
				return fmt.Errorf("opening file %q: %w", fp, err)
			}
			files[first] = f
		}
		if _, err := f.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("writing line to %q: %w", fp, err)
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("reading object %q: %w", filename, err)
	}
	return nil
}
