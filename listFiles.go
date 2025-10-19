package main

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/viper"
)

func listFiles(s3c *s3.Client, maxNumber int) ([]string, error) {
	n := int32(maxNumber)
	b := viper.GetString("bucket")
	resp, err := s3c.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket:  &b,
		MaxKeys: &n,
	})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(resp.Contents))
	for _, key := range resp.Contents {
		lfs := logFilenamePattern.FindStringSubmatch(*key.Key)
		if len(lfs) > 0 {
			out = append(out, lfs[0])
		}
	}
	if len(out) == 0 {
		return nil, errNoFilesInBucket
	}
	return out, nil
}
