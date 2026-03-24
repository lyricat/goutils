package twitter

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"testing"
)

func TestCheckAPIResponseIncludesRequestContext(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}})))
	defer slog.SetDefault(prev)

	req, err := http.NewRequest(http.MethodGet, "https://api.x.com/2/tweets?expansions=author_id%2Creferenced_tweets.id&ids=1%2C2%2C3", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp := &http.Response{
		Status:     "503 Service Unavailable",
		StatusCode: http.StatusServiceUnavailable,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(nil)),
	}
	resp.Header.Set("x-rate-limit-limit", "900")
	resp.Header.Set("x-rate-limit-remaining", "0")
	resp.Header.Set("x-rate-limit-reset", "1711210000")

	body := []byte(`{"title":"Service Unavailable","detail":"Service Unavailable","type":"about:blank","status":503}`)

	err = (&Client{}).checkAPIResponse(req, resp, body, http.StatusOK)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errText := err.Error()
	for _, want := range []string{
		"GET https://api.x.com/2/tweets?expansions=author_id%2Creferenced_tweets.id&ids=1%2C2%2C3 returned 503 Service Unavailable",
		"api_error: Service Unavailable - Service Unavailable (about:blank)",
		`body: {"title":"Service Unavailable","detail":"Service Unavailable","type":"about:blank","status":503}`,
		"x-rate-limit-limit=900",
		"x-rate-limit-remaining=0",
		"x-rate-limit-reset=1711210000",
	} {
		if !bytes.Contains([]byte(errText), []byte(want)) {
			t.Fatalf("error %q does not contain %q", errText, want)
		}
	}
	if bytes.Contains([]byte(errText), []byte("%!")) {
		t.Fatalf("error %q still contains fmt formatting corruption", errText)
	}

	logText := buf.String()
	for _, want := range []string{
		"level=ERROR",
		"msg=\"x api request failed\"",
		"method=GET",
		"url=\"https://api.x.com/2/tweets?expansions=author_id%2Creferenced_tweets.id&ids=1%2C2%2C3\"",
		"status=\"503 Service Unavailable\"",
		"status_code=503",
		"x_rate_limit_limit=900",
		"x_rate_limit_remaining=0",
		"x_rate_limit_reset=1711210000",
	} {
		if !bytes.Contains([]byte(logText), []byte(want)) {
			t.Fatalf("log %q does not contain %q", logText, want)
		}
	}
}

func TestCheckAPIResponseAllowsCreated(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequest(http.MethodPost, "https://api.x.com/2/tweets", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp := &http.Response{
		Status:     "201 Created",
		StatusCode: http.StatusCreated,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(nil)),
	}

	if err := (&Client{}).checkAPIResponse(req, resp, []byte(`{"data":{"id":"1"}}`), http.StatusCreated); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestWrapAPIRequestErrorIncludesRequestContext(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}})))
	defer slog.SetDefault(prev)

	req, err := http.NewRequest(http.MethodGet, "https://api.x.com/2/tweets/123", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	wrappedErr := wrapAPIRequestError(req, "failed to execute request", io.EOF)
	errText := wrappedErr.Error()
	for _, want := range []string{
		"failed to execute request",
		"GET https://api.x.com/2/tweets/123",
		"EOF",
	} {
		if !bytes.Contains([]byte(errText), []byte(want)) {
			t.Fatalf("error %q does not contain %q", errText, want)
		}
	}

	logText := buf.String()
	for _, want := range []string{
		"level=ERROR",
		"msg=\"x api request execution failed\"",
		"message=\"failed to execute request\"",
		"method=GET",
		"url=https://api.x.com/2/tweets/123",
	} {
		if !bytes.Contains([]byte(logText), []byte(want)) {
			t.Fatalf("log %q does not contain %q", logText, want)
		}
	}
}
