package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	StorageProviderR2    = "r2"
	StorageProviderS3    = "s3"
	StorageProviderLocal = "local"
)

const (
	ACLPrivate    = "private"
	ACLPublicRead = "public-read"
)

type (
	Storage struct {
		cfg      Config
		s3Client *s3.Client
		bucket   string
	}
	R2Config struct {
		AccountID       string
		AccessKey       string
		AccessKeySecret string
		Bucket          string
	}
	S3Config struct {
		Region          string
		AccessKey       string
		AccessKeySecret string
		Bucket          string
	}
	Config struct {
		Provider string

		R2 R2Config
		S3 S3Config

		LocalPath string
	}

	WriteAsReaderInput struct {
		Filepath string
		Filename string
		File     io.ReadSeeker
		Size     int64
		MimeType string
		ACL      string
	}
	WriteInput struct {
		Filepath string
		Filename string
		Content  string
		MimeType string
		ACL      string
	}
)

func New(
	cfg Config,
) *Storage {
	ctx := context.TODO()
	var awscfg aws.Config
	var s3Client *s3.Client
	var err error
	bucket := ""
	if cfg.Provider == StorageProviderR2 {
		awscfg, err = awsconfig.LoadDefaultConfig(
			ctx,
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.R2.AccessKey, cfg.R2.AccessKeySecret, "")),
			awsconfig.WithRegion("auto"),
		)
		if err != nil {
			panic(err)
		}

		s3Client = s3.NewFromConfig(awscfg, func(o *s3.Options) {
			o.UsePathStyle = true
			o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.R2.AccountID))
		})
		bucket = cfg.R2.Bucket

	} else if cfg.Provider == StorageProviderS3 {
		awscfg, err = awsconfig.LoadDefaultConfig(
			ctx,
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.S3.AccessKey, cfg.S3.AccessKeySecret, "")),
			awsconfig.WithRegion(cfg.S3.Region),
		)
		if err != nil {
			panic(err)
		}

		s3Client = s3.NewFromConfig(awscfg, func(o *s3.Options) {
			o.UsePathStyle = true
			o.BaseEndpoint = aws.String(fmt.Sprintf("https://s3.%s.amazonaws.com", cfg.S3.Region))
		})
		bucket = cfg.S3.Bucket
	}

	return &Storage{
		cfg:      cfg,
		s3Client: s3Client,
		bucket:   bucket,
	}
}

func (st *Storage) WriteAsReader(ctx context.Context, input *WriteAsReaderInput) error {
	if st.cfg.Provider == StorageProviderR2 || st.cfg.Provider == StorageProviderS3 {
		key := path.Join(input.Filepath, input.Filename)
		if err := st.S3Upload(ctx, &S3UploadInput{
			Bucket: st.bucket, Key: key,
			File:     input.File,
			Size:     input.Size,
			MimeType: input.MimeType,
			ACL:      input.ACL,
		}); err != nil {
			return err
		}

	} else if st.cfg.Provider == StorageProviderLocal {
		fullpath := path.Join(st.cfg.LocalPath, input.Filepath, input.Filename)
		if err := st.CheckPath(ctx, path.Join(st.cfg.LocalPath, input.Filepath)); err != nil {
			return err
		}
		fd, err := os.Create(fullpath)
		if err != nil {
			return err
		}
		defer fd.Close()
		_, err = io.Copy(fd, input.File)
		if err != nil {
			return err
		}

	} else {
		return errors.New("invalid storage provider")
	}

	return nil
}

func (st *Storage) Write(ctx context.Context, input *WriteInput) error {
	if st.cfg.Provider == StorageProviderR2 || st.cfg.Provider == StorageProviderS3 {
		// upload to r2
		reader := strings.NewReader(input.Content)
		key := path.Join(input.Filepath, input.Filename)
		if err := st.S3Upload(ctx, &S3UploadInput{
			Bucket: st.bucket, Key: key,
			File:     reader,
			Size:     int64(len(input.Content)),
			MimeType: input.MimeType,
			ACL:      input.ACL,
		}); err != nil {
			return err
		}

	} else if st.cfg.Provider == StorageProviderLocal {
		fullpath := path.Join(st.cfg.LocalPath, input.Filepath, input.Filename)
		if err := st.CheckPath(ctx, path.Join(st.cfg.LocalPath, input.Filepath)); err != nil {
			return err
		}
		// write to local
		if err := os.WriteFile(fullpath, []byte(input.Content), 0644); err != nil {
			return err
		}

	} else {
		return errors.New("invalid storage provider")
	}

	return nil
}

func (st *Storage) Delete(ctx context.Context, filepath, filename string) error {
	if st.cfg.Provider == StorageProviderR2 || st.cfg.Provider == StorageProviderS3 {
		key := path.Join(filepath, filename)
		if err := st.S3Delete(ctx, st.bucket, key); err != nil {
			return err
		}

	} else if st.cfg.Provider == StorageProviderLocal {
		fullpath := path.Join(st.cfg.LocalPath, filepath, filename)
		if _, err := os.Stat(fullpath); err == nil {
			if err := os.Remove(fullpath); err != nil {
				return err
			}
		}

	} else {
		return errors.New("invalid storage provider")
	}

	return nil
}

func (st *Storage) DeleteByPrefix(ctx context.Context, prefix string) error {
	if st.cfg.Provider == StorageProviderR2 || st.cfg.Provider == StorageProviderS3 {
		if err := st.S3DeleteByPrefix(ctx, st.bucket, prefix); err != nil {
			return err
		}

	} else if st.cfg.Provider == StorageProviderLocal {
		fullpath := path.Join(st.cfg.LocalPath, prefix)
		if _, err := os.Stat(fullpath); err == nil {
			if err := os.RemoveAll(fullpath); err != nil {
				return err
			}
		}

	} else {
		return errors.New("invalid storage provider")
	}

	return nil
}

func (st *Storage) GetAsReader(ctx context.Context, filepath, filename string) (io.Reader, error) {
	if st.cfg.Provider == StorageProviderR2 || st.cfg.Provider == StorageProviderS3 {
		key := path.Join(filepath, filename)
		return st.S3GetAsReader(ctx, st.bucket, key)

	} else if st.cfg.Provider == StorageProviderLocal {
		fullpath := path.Join(st.cfg.LocalPath, filepath, filename)
		if _, err := os.Stat(fullpath); err == nil {
			return os.Open(fullpath)
		} else {
			return nil, err
		}

	} else {
		return nil, errors.New("invalid storage provider")
	}
}

func (st *Storage) CheckPath(ctx context.Context, filepath string) error {
	if _, err := os.Stat(filepath); err == nil {
		return nil

	} else if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath, 0755); err != nil {
			return err
		}
		return nil

	} else {
		return err
	}
}
