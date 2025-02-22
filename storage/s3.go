package storage

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type (
	S3UploadInput struct {
		Bucket   string
		Key      string
		File     io.Reader
		Size     int64
		MimeType string
		ACL      string
	}
)

func (st *Storage) S3Upload(ctx context.Context, input *S3UploadInput) error {
	// increase timeout
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	if input.MimeType == "" {
		input.MimeType = "application/octet-stream"
	}
	payload := &s3.PutObjectInput{
		Bucket:        aws.String(input.Bucket),
		Key:           aws.String(input.Key),
		Body:          input.File,
		ContentLength: &input.Size,
		ContentType:   aws.String(input.MimeType),
	}
	payload.ACL = types.ObjectCannedACLPrivate

	if input.ACL == ACLPublicRead {
		payload.ACL = types.ObjectCannedACLPublicRead
	}
	_, err := st.s3Client.PutObject(ctx, payload)
	if err != nil {
		slog.Error("[goutils.s3] failed to upload object", "error", err, "bucket", input.Bucket, "key", input.Key)
		return err
	}

	return nil
}

func (st *Storage) S3Delete(ctx context.Context, bucket, key string) error {
	if _, err := st.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}); err != nil {
		slog.Error("[goutils.s3] failed to delete object", "error", err, "bucket", bucket, "key", key)
		return err
	}
	return nil
}

func (st *Storage) S3DeleteByPrefix(ctx context.Context, bucket, prefix string) error {
	resp, err := st.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		slog.Error("[goutils.s3] failed to list objects", "error", err, "bucket", bucket, "prefix", prefix)
		return err
	}

	for _, obj := range resp.Contents {
		_, err := st.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    obj.Key,
		})
		if err != nil {
			slog.Error("[goutils.s3] failed to delete object", "error", err, "bucket", bucket, "key", *obj.Key)
			continue
		}
	}

	return nil
}

func (st *Storage) S3GetAsReader(ctx context.Context, bucket, key string) (io.Reader, error) {
	// download from r2 as a reader
	resp, err := st.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		slog.Error("[goutils.s3] failed to get object", "error", err, "bucket", bucket, "key", key)
		return nil, err
	}
	defer resp.Body.Close()

	// copy to a io.Reader
	reader := &bytes.Buffer{}
	io.Copy(reader, resp.Body)
	if err != nil {
		slog.Error("[goutils.s3] failed to copy object to reader", "error", err, "bucket", bucket, "key", key)
		return nil, err
	}

	return reader, nil
}
