package core

import (
	"context"
	"io"
	"time"
)

const (
	// init .
	// pending -> upload to storage, persist to db -> done
	AttachmentStatusInit = iota
	AttachmentStatusPending
	AttachmentStatusDone
	AttachmentStatusFailed
)

const (
	AttachmentChecksumMethodSHA1 = "sha1"
)

type (
	UploadAttachmentInput struct {
		OwnerID              uint64
		File                 io.ReadSeeker
		Filename             string
		Filesize             int64
		DstPrefix            string
		IsPublic             bool
		Encrypt              bool
		EncryptPublicKey     string
		DownloadURL          string
		DownloadReferrerHost string
	}

	SyncToReplicasInput struct {
		IsPublic bool
	}

	Attachment struct {
		ID               uint64 `json:"id"`
		OwnerID          uint64 `json:"owner_id"` // an abstract owner who owns this attachment
		BucketName       string `json:"-"`
		HashID           string `json:"hash_id"` // the unique hash id of the file
		Size             int64  `json:"size"`    // size in bytes
		MimeType         string `json:"mime_type"`
		Pathname         string `json:"pathname"` // the path of the file
		Filename         string `json:"filename"` // the filename of the file
		Status           int    `json:"status"`
		OriginalMimeType string `json:"original_mime_type"` // the original mime type of the file before convert

		// calculate the checksum of original file
		Checksum       string `json:"checksum"`
		ChecksumMethod string `json:"checksum_method"`

		CreatedAt *time.Time `json:"created_at"`
		UpdatedAt *time.Time `json:"updated_at"`

		ViewURL string `gorm:"-" json:"view_url"` // the url to view the file, if the file is public
	}

	AttachmentStore interface {
		// INSERT INTO @@table
		//  (owner_id, bucket_name, hash_id,
		//   size, mime_type, pathname, filename,
		//   status,
		//   original_mime_type,
		//   checksum, checksum_method,
		//   created_at, updated_at
		//  )
		// VALUES
		//  (
		//   @att.OwnerID, @att.BucketName, @att.HashID,
		//   @att.Size, @att.MimeType, @att.Pathname, @att.Filename,
		//   @att.Status,
		//   @att.OriginalMimeType,
		//   @att.Checksum, @att.ChecksumMethod,
		//   NOW(), NOW()
		//  )
		// RETURNING id;
		CreateAttachment(ctx context.Context, att *Attachment) (uint64, error)

		// SELECT * FROM @@table
		// WHERE id = @id
		// LIMIT 1;
		GetAttachment(ctx context.Context, id uint64) (*Attachment, error)

		// SELECT * FROM @@table
		// WHERE hash_id = @hashID
		// LIMIT 1;
		GetAttachmentByHashID(ctx context.Context, hashID string) (*Attachment, error)

		// SELECT * FROM @@table
		// WHERE checksum_method = @method AND checksum = @checksum
		// LIMIT 1;
		GetAttachmentByChecksum(ctx context.Context, method, checksum string) (*Attachment, error)

		// SELECT * FROM @@table
		// WHERE status = @status
		// LIMIT @limit;
		GetAttachmentsByStatus(ctx context.Context, status int, limit uint64) ([]*Attachment, error)

		// SELECT * FROM @@table
		// WHERE id > @sinceID
		// ORDER BY id ASC
		// LIMIT @limit;
		GetAttachmentsSinceID(ctx context.Context, sinceID uint64, limit uint64) ([]*Attachment, error)

		// UPDATE @@table
		//  {{set}}
		//   hash_id = @att.HashID,
		//   filename = @att.Filename,
		//   status = @att.Status,
		//   checksum = @att.Checksum,
		//  {{end}}
		// WHERE
		//  "id" = @att.ID;
		UpdateAttachment(ctx context.Context, att *Attachment) error
	}

	AttachmentService interface {
		UploadFile(ctx context.Context, input *UploadAttachmentInput) (*Attachment, error)
		GetAttachment(ctx context.Context, id uint64) (*Attachment, error)
		GetFileMimeType(file io.ReadSeeker, ext string) (string, error)
		GetAttachmentsSinceID(ctx context.Context, sinceID uint64, limit uint64) ([]*Attachment, error)
		DownloadRemoteFile(ctx context.Context, input *UploadAttachmentInput) (string, error)
		SyncToReplicas(ctx context.Context, attachment *Attachment, input *SyncToReplicasInput) error
	}
)
