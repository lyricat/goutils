package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type (
	Client struct {
		cfg Config
	}

	Config struct {
		botToken  string
		channelID string
	}

	TelegramVerifyResp struct {
		Ok          bool   `json:"ok"`
		ErrorCode   int    `json:"error_code,omitempty"`
		Description string `json:"description,omitempty"`

		Result struct {
			Status          string `json:"status"`
			CanPostMessages bool   `json:"can_post_messages"`
		} `json:"result"`
	}
)

func New(
	botToken, channelID string,
) *Client {
	return &Client{
		cfg: Config{
			botToken:  botToken,
			channelID: channelID,
		},
	}
}

type (
	SendMessageResp struct {
		Ok          bool   `json:"ok"`
		ErrorCode   int    `json:"error_code,omitempty"`
		Description string `json:"description,omitempty"`
	}
	SendMessageReq struct {
		ChatID             string             `json:"chat_id"`
		ParseMode          string             `json:"parse_mode"`
		Text               string             `json:"text"`
		LinkPreviewOptions LinkPreviewOptions `json:"link_preview_options"`
	}
	LinkPreviewOptions struct {
		IsDisabled    bool `json:"is_disabled"`
		ShowAboveText bool `json:"show_above_text"`
	}
)

func (s *Client) SendTextMessage(ctx context.Context, title, text, url string) error {
	smr := SendMessageReq{
		ChatID:    s.cfg.channelID,
		ParseMode: "markdown",
		Text:      fmt.Sprintf("*%s*\n\n%s", title, text),
		LinkPreviewOptions: LinkPreviewOptions{
			ShowAboveText: true,
		},
	}
	if url != "" {
		smr.Text = fmt.Sprintf("# *%s*\n\n%s\n\n👉 %s\n\n", title, text, url)
	}
	return s.SendTextMessageRaw(ctx, smr)
}

func (s *Client) SendTextMessageRaw(ctx context.Context, smr SendMessageReq) error {
	if smr.ChatID == "" {
		smr.ChatID = s.cfg.channelID
	}

	if smr.ParseMode == "" {
		smr.ParseMode = "markdown"
	}

	payload, err := json.Marshal(smr)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(payload)
	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.cfg.botToken)
	req, err := http.NewRequest(http.MethodPost, apiUrl, buf)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var body SendMessageResp
	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return err
	}

	if !body.Ok {
		return fmt.Errorf("unsuccessful telegram send message request: %d, %s", body.ErrorCode, body.Description)
	}

	return nil
}

func (s *Client) VerifyPermission(ctx context.Context) error {
	parts := strings.Split(s.cfg.botToken, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid bot token: %s", s.cfg.botToken)
	}
	botID := parts[0]
	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s/getChatMember?chat_id=%s&user_id=%s", s.cfg.botToken, s.cfg.channelID, botID)
	req, err := http.NewRequest(http.MethodPost, apiUrl, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var body TelegramVerifyResp
	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return err
	}

	if !body.Ok {
		return fmt.Errorf("unsuccessful verify bot permission: %d, %s", body.ErrorCode, body.Description)
	}

	if !body.Result.CanPostMessages {
		return fmt.Errorf("bot %s has no permission to post messages to channel %s", botID, s.cfg.channelID)
	}

	return nil
}
