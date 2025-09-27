package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/viper"
)

func main() {
	if err := initConfig(); err != nil {
		fmt.Fprintln(os.Stderr, "failed loading gad config:", err)
		os.Exit(1)
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(viper.GetString("profile")))
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed loading AWS config:", err)
		os.Exit(1)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.DisableLogOutputChecksumValidationSkipped = true
	})

	findUnsafe := false
	for {
		fmt.Printf("getting new files... ")
		list, err := listFiles(client, 500)
		if err != nil {
			log.Fatalf("unable to identify the property to manage: %v\n", err)
		}
		fmt.Printf("done\n")

		for _, i := range list {
			fmt.Printf("processing %s... ", i)

			if !isSafeToProcess(i) {
				findUnsafe = true
				fmt.Printf("skipped (unsafe)\n")
				continue
			}

			if err := importLogFileContent(client, i); err != nil {
				panic(fmt.Sprintf("failed to import file: %v", err))
			}
			// Delete file
			fmt.Printf("(deleting) ")
			b := viper.GetString("bucket")
			_, err := client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
				Bucket: &b,
				Key:    &i,
			})
			if err != nil {
				panic(fmt.Sprintf("failed to delete file: %v", err))
			}
			// Done
			fmt.Printf("done\n")
		}
		if findUnsafe {
			break
		}
	}
}
