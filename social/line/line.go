package line

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/lyricat/goutils/uuid"
)

type (
	Client struct {
		cfg         Config
		bot         *messaging_api.MessagingApiAPI
		blobBot     *messaging_api.MessagingApiBlobAPI
		accessToken string
	}
	Config struct {
		ChannelID  string
		ChannelKey string
		PrivateKey string
	}
)

func New(cfg Config) (*Client, error) {
	decoded, err := base64.StdEncoding.DecodeString(cfg.PrivateKey)
	if err != nil {
		slog.Error("[goutils] failed to decode line private key", "error", err)
		return nil, err
	}
	cfg.PrivateKey = string(decoded)

	return &Client{
		cfg: cfg,
		bot: nil,
	}, nil
}

func NewFromAccessToken(token string) (*Client, error) {
	bot, err := messaging_api.NewMessagingApiAPI(token)
	if err != nil {
		return nil, err
	}
	blobBot, err := messaging_api.NewMessagingApiBlobAPI(token)
	if err != nil {
		return nil, err
	}
	return &Client{
		cfg:     Config{},
		bot:     bot,
		blobBot: blobBot,
	}, nil
}

func (s *Client) GenerateToken() (string, *time.Time, error) {
	jwt, err := s.GenerateJWTFromJWK(s.cfg.PrivateKey, s.cfg.ChannelKey)
	if err != nil {
		return "", nil, err
	}

	token, expiredAt, err := getChannelStatelessAccessToken(jwt)
	if err != nil {
		return "", nil, err
	}

	s.bot, err = messaging_api.NewMessagingApiAPI(token)
	if err != nil {
		return "", nil, err
	}

	s.blobBot, err = messaging_api.NewMessagingApiBlobAPI(token)
	if err != nil {
		return "", nil, err
	}
	return token, expiredAt, nil
}

func (s *Client) SendPushMessage(ctx context.Context, groupID, title, summary, url string) error {
	content := fmt.Sprintf("%s\n%s\n\n👉 %s", title, summary, url)
	_, err := s.bot.PushMessage(&messaging_api.PushMessageRequest{
		To: groupID,
		Messages: []messaging_api.MessageInterface{
			messaging_api.TextMessage{
				Text: content,
			},
		},
	}, uuid.New())

	if err != nil {
		return err
	}
	return nil
}

func (s *Client) ReplyTextMessage(replyToken, quoteToken string, text string) (*messaging_api.ReplyMessageResponse, error) {
	msg := messaging_api.TextMessage{
		Text: text,
	}
	if quoteToken != "" {
		msg.QuoteToken = quoteToken
	}
	resp, err := s.bot.ReplyMessage(
		&messaging_api.ReplyMessageRequest{
			ReplyToken: replyToken,
			Messages: []messaging_api.MessageInterface{
				msg,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Client) GetContent(messageID string) ([]byte, error) {
	resp, err := s.blobBot.GetMessageContent(messageID)
	if err != nil {
		slog.Error("[line] failed to get content", "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	// read the content as buffer
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("[line] failed to read content", "error", err)
		return nil, err
	}

	return buf, nil
}
