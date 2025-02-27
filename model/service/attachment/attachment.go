package attachment

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/lyricat/goutils/crypto"
	"github.com/lyricat/goutils/model/core"
	"github.com/lyricat/goutils/storage"
	"github.com/nickalie/go-webpbin"

	"github.com/speps/go-hashids/v2"
	"gorm.io/gorm"
)

const UserAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/119.0"

var imageExtToWebp = []string{".jpg", ".jpeg", ".png"}

type Config struct {
	Bucket               string
	Base                 string
	HashIDSalt           string
	ConvertToWebp        bool
	ConvertToWebpMaxSize int64
	SupportedExts        []string
}

type AttachmentService struct {
	cfg         Config
	storage     *storage.Storage
	hashInst    *hashids.HashID
	attachments core.AttachmentStore
}

func New(cfg Config, attachments core.AttachmentStore, storage *storage.Storage) *AttachmentService {
	hd := hashids.NewData()
	hd.Salt = cfg.HashIDSalt
	hd.MinLength = 8
	hd.Alphabet = "abcdefghijklmnopqrstuvwxyz1234567890"
	hashInst, _ := hashids.NewWithData(hd)

	if cfg.ConvertToWebpMaxSize == 0 {
		cfg.ConvertToWebpMaxSize = 5 * 1024 * 1024
	}

	return &AttachmentService{
		cfg:         cfg,
		storage:     storage,
		hashInst:    hashInst,
		attachments: attachments,
	}
}

func (s *AttachmentService) GetAttachment(ctx context.Context, id uint64) (*core.Attachment, error) {
	return s.attachments.GetAttachment(ctx, id)
}

func (s *AttachmentService) UploadFile(ctx context.Context, input *core.UploadAttachmentInput) (*core.Attachment, error) {
	// checksum
	checksum, err := getFileSha1Sum(input.File)
	if err != nil {
		return nil, err
	}

	existing, err := s.attachments.GetAttachmentByChecksum(ctx, core.AttachmentChecksumMethodSHA1, checksum)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if existing != nil && existing.ID != 0 {
		// only provide view url for static storage
		existing.ViewURL = fmt.Sprintf("%s/%s/%s", s.cfg.Base, input.DstPrefix, existing.Filename)
		return existing, nil
	}

	ext := strings.ToLower(filepath.Ext(input.Filename))

	uploadFile := input.File
	// create attachment in db
	mimeType, err := s.GetFileMimeType(uploadFile, ext)
	if err != nil {
		mimeType = "application/octet-stream"
	}
	if ext == "" {
		ext = GuessExtByMimeType(mimeType)
	}

	att := &core.Attachment{
		OwnerID:          input.OwnerID,
		HashID:           "",
		Size:             input.Filesize,
		MimeType:         mimeType,
		OriginalMimeType: mimeType,
		ChecksumMethod:   core.AttachmentChecksumMethodSHA1,
		Pathname:         input.DstPrefix,
	}

	att.BucketName = s.cfg.Bucket

	//
	if s.cfg.ConvertToWebp && strings.HasPrefix(mimeType, "image/") {
		if slices.Contains(imageExtToWebp, ext) {
			if input.Filesize < s.cfg.ConvertToWebpMaxSize {
				converted, err := ConvertStream2Webp(uploadFile, ext)
				if err != nil {
					slog.Warn("[goutils.attachmentz] convert to webp failed, use the original image", "filename", input.Filename, "error", err)
					uploadFile.Seek(0, io.SeekStart)
				} else {
					if converted != nil && converted.Len() > 0 {
						att.OriginalMimeType = mimeType
						att.MimeType = "image/webp"
						att.Size = int64(converted.Len())
						ext = ".webp"
						uploadFile = bytes.NewReader(converted.Bytes())
					}
				}
			}
		}
	}

	attID, err := s.attachments.CreateAttachment(ctx, att)
	if err != nil {
		return nil, err
	}
	att.ID = attID

	// cal hash id
	hid, err := s.hashInst.EncodeInt64([]int64{int64(input.OwnerID), int64(attID)})
	if err != nil {
		return nil, err
	}

	// do the encryption
	if input.Encrypt {
		// decrypt file
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, uploadFile); err != nil {
			return nil, err
		}
		encrypedBuf, err := crypto.EncryptBytes(buf.Bytes(), input.EncryptPublicKey)
		if err != nil {
			return nil, err
		}
		uploadFile = bytes.NewReader(encrypedBuf)
		att.Size = int64(len(encrypedBuf))
	}

	// upload file to storage
	attFilename := fmt.Sprintf("%s%s", hid, ext)
	acl := storage.ACLPrivate
	if input.IsPublic {
		acl = storage.ACLPublicRead
	}
	if err := s.storage.WriteAsReader(ctx, &storage.WriteAsReaderInput{
		Filepath: input.DstPrefix,
		Filename: attFilename,
		File:     uploadFile,
		Size:     att.Size,
		MimeType: att.MimeType,
		ACL:      acl,
	}); err != nil {
		slog.Error("[goutils.attachmentz] upload file failed", "error", err)
		return nil, err
	}

	// update attachment status and info
	payload := &core.Attachment{
		ID:       attID,
		HashID:   hid,
		Pathname: input.DstPrefix,
		Filename: attFilename,
		Status:   core.AttachmentStatusDone,
		Checksum: checksum,
	}
	if err := s.attachments.UpdateAttachment(ctx, payload); err != nil {
		return nil, err
	}

	att.HashID = hid
	att.Filename = attFilename
	att.Status = core.AttachmentStatusDone
	att.Checksum = checksum
	att.ViewURL = fmt.Sprintf("%s/%s/%s", s.cfg.Base, input.DstPrefix, attFilename)

	return att, nil
}

func (s *AttachmentService) DownloadRemoteFile(ctx context.Context, input *core.UploadAttachmentInput) (string, error) {
	// download image from url, and then call attachmentz.UploadFileToR2 to upload to quail
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, input.DownloadURL, nil)
	if err != nil {
		return "", err
	}
	if input.DownloadReferrerHost != "" {
		req.Header.Add("Host", input.DownloadReferrerHost)
		req.Header.Add("Referer", fmt.Sprintf("https://%s/", input.DownloadReferrerHost))
	}
	req.Header.Add("User-Agent", UserAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// create a temp file and copy the response body to it
	tempFile, err := os.CreateTemp("", "quailyimage-")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return "", err
	}

	// open the temp file and upload to quail
	file, err := os.Open(tempFile.Name())
	if err != nil {
		return "", err
	}

	fi, err := file.Stat()
	if err != nil {
		return "", err
	}

	// remove the query string from the url
	trimUrl := strings.Split(input.DownloadURL, "?")[0]

	// get filename from url
	filename := filepath.Base(trimUrl)
	input.Filename = filename
	input.Filesize = fi.Size()
	input.File = file

	att, err := s.UploadFile(ctx, input)
	if err != nil {
		return "", err
	}

	return att.ViewURL, nil
}

func (s *AttachmentService) GetFileMimeType(file io.ReadSeeker, ext string) (string, error) {
	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)

	_, err := file.Read(buffer)
	if err != nil {
		return "", err
	}

	// Reset the file cursor to the beginning
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	contentType := http.DetectContentType(buffer)

	// special cases
	switch ext {
	case ".svg":
		if strings.HasPrefix(contentType, "text/") {
			return "image/svg+xml", nil
		}
	case ".xml":
		if strings.HasPrefix(contentType, "text/") {
			return "application/xml", nil
		}
	case ".json":
		if strings.HasPrefix(contentType, "text/") {
			return "application/json", nil
		}
	}
	return strings.ToLower(contentType), nil
}

func GuessExtByMimeType(mimeType string) string {
	if mimeType == "" {
		return ""
	}

	if strings.HasPrefix(mimeType, "image/jpeg") || strings.HasPrefix(mimeType, "image/jpg") {
		return ".jpg"
	} else if strings.HasPrefix(mimeType, "image/png") {
		return ".png"
	} else if strings.HasPrefix(mimeType, "image/gif") {
		return ".gif"
	} else if strings.HasPrefix(mimeType, "image/webp") {
		return ".webp"
	} else if strings.HasPrefix(mimeType, "image/svg") {
		return ".svg"
	}
	return ""
}

func getFileSha1Sum(file io.ReadSeeker) (string, error) {
	hasher := sha1.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	md5sum := hex.EncodeToString(hasher.Sum(nil))
	return md5sum, nil
}

func ConvertStream2Webp(file io.ReadSeeker, ext string) (*bytes.Buffer, error) {
	var img image.Image
	var err error
	if ext == ".jpg" || ext == ".jpeg" {
		img, err = jpeg.Decode(file)
		if err != nil {
			return nil, err
		}
	} else if ext == ".png" {
		img, err = png.Decode(file)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, nil
	}

	buf := &bytes.Buffer{}
	if err := webpbin.Encode(buf, img); err != nil {
		return nil, err
	}
	return buf, nil
}
