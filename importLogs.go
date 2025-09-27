package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/viper"
)

func importLogFileContent(s3c *s3.Client, filename string) error {
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

	dir := path.Join(viper.GetString("logs-folder"), lfs[1], lfs[2])
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("folder creation %q: %w", dir, err)
	}
	filePath := path.Join(dir, fmt.Sprintf("%s-%s-%s.log", lfs[1], lfs[2], lfs[3]))
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening file %q: %w", filePath, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("copying stream to file: %w", err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync file: %w", err)
	}
	return nil
}
