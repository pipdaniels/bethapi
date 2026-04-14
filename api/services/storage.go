package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"bethapi/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	v2config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type StorageService struct {
	S3Client *s3.Client
	Bucket   string
	BaseURL  string
}

var Storage *StorageService

func InitStorage() {
	cfg, err := v2config.LoadDefaultConfig(context.TODO(),
		v2config.WithRegion("auto"),
		v2config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     config.AppConfig.R2AccessKey,
				SecretAccessKey: config.AppConfig.R2SecretKey,
			}, nil
		})),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	Storage = &StorageService{
		S3Client: s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(config.AppConfig.R2Endpoint)
		}),
		Bucket:  config.AppConfig.R2Bucket,
		BaseURL: config.AppConfig.R2PublicDomain,
	}
	log.Println("Storage Service (R2) Initialized")
}

func (s *StorageService) GetPresignedURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s.S3Client)
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
	}

	request, err := presignClient.PresignGetObject(ctx, input, func(o *s3.PresignOptions) {
		o.Expires = expires
	})
	if err != nil {
		return "", err
	}

	return request.URL, nil
}

func (s *StorageService) GetPublicURL(key string) string {
	return fmt.Sprintf("%s/%s", s.BaseURL, key)
}
