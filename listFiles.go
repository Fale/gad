package main

import (
	"context"
	"errors"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/viper"
)

func listFiles(s3c *s3.Client, maxNumber int) ([]string, error) {
	logFiles := regexp.MustCompile(`([0-9]{4}-[0-9]{2}-[0-9]{2})T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3}-.+\.log`)
	n := int32(maxNumber)
	b := viper.GetString("bucket")
	resp, err := s3c.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket:  &b,
		MaxKeys: &n,
	})
	if err != nil {
		return []string{}, err
	}
	if len(resp.Contents) == 0 {
		return []string{}, errors.New("no files in the bucket")
	}
	out := []string{}
	for _, key := range resp.Contents {
		lfs := logFiles.FindStringSubmatch(*key.Key)
		if len(lfs) > 0 {
			out = append(out, lfs[0])
		}
	}
	return out, nil
}
