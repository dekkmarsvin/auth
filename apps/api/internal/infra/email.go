package infra

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/mailgun/mailgun-go/v5"
)

type EmailClient interface {
	SendEmail(to string, title string, content string) error
}

type emailClient struct {
	mg     *mailgun.Client
	domain string
}

func NewEmailClient(domain string, apiKey string, apiBase string) (EmailClient, error) {
	domain = strings.TrimSpace(domain)
	apiKey = strings.TrimSpace(apiKey)
	apiBase = strings.TrimRight(strings.TrimSpace(apiBase), "/")

	if domain == "" {
		return nil, errors.New("MAILGUN_DOMAIN is required")
	}
	if apiKey == "" {
		return nil, errors.New("MAILGUN_APIKEY is required")
	}

	mg := mailgun.NewMailgun(apiKey)
	if apiBase != "" {
		if err := mg.SetAPIBase(apiBase); err != nil {
			return nil, fmt.Errorf("invalid MAILGUN_API_BASE: %w", err)
		}
	}
	return &emailClient{
		mg:     mg,
		domain: domain,
	}, nil
}

func (c *emailClient) SendEmail(to string, title string, content string) error {
	from := "轻小说机翻机器人 <no-reply@" + c.domain + ">"
	message := mailgun.NewMessage(c.domain, from, title, content, to)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, err := c.mg.Send(ctx, message)
	if err != nil {
		slog.Error("Failed to send email", "error", err)
		return err
	}
	return nil
}
