package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ArdaGnsrn/opsvault/internal/config"
)

type TelegramNotifier struct {
	token  string
	chatID string
	client *http.Client
}

func NewTelegramNotifier(cfg config.TelegramConfig) *TelegramNotifier {
	token := cfg.BotToken
	return &TelegramNotifier{
		token:  token,
		chatID: cfg.ChatID,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (t *TelegramNotifier) Name() string { return "telegram" }

func (t *TelegramNotifier) Send(ctx context.Context, msg Message) error {
	icon := "✅"
	if msg.Level == LevelError {
		icon = "❌"
	}

	text := fmt.Sprintf("%s *%s*\n\n%s", icon, escapeMarkdown(msg.Subject), escapeMarkdown(msg.Body))

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.token)

	params := url.Values{}
	params.Set("chat_id", t.chatID)
	params.Set("text", text)
	params.Set("parse_mode", "Markdown")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(params.Encode()))
	if err != nil {
		return fmt.Errorf("telegram: creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: sending message: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("telegram: decoding response: %w", err)
	}
	if !result.OK {
		return fmt.Errorf("telegram: API error: %s", result.Description)
	}
	return nil
}

func escapeMarkdown(s string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"`", "\\`",
	)
	return replacer.Replace(s)
}
