package slack

type (
	Message struct {
		Text   string `json:"text"`
		Mrkdwn bool   `json:"mrkdwn"`
	}
)
