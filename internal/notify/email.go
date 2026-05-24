package notify

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/ArdaGnsrn/opsvault/internal/config"
)

type EmailNotifier struct {
	cfg config.EmailConfig
}

func NewEmailNotifier(cfg config.EmailConfig) *EmailNotifier {
	return &EmailNotifier{cfg: cfg}
}

func (e *EmailNotifier) Name() string { return "email" }

func (e *EmailNotifier) Send(ctx context.Context, msg Message) error {
	addr := fmt.Sprintf("%s:%d", e.cfg.SMTPHost, e.cfg.SMTPPort)

	body := buildEmailBody(msg)

	var err error
	if e.cfg.SMTPTLS {
		err = e.sendSTARTTLS(addr, body)
	} else {
		err = e.sendPlain(addr, body)
	}
	return err
}

func (e *EmailNotifier) sendSTARTTLS(addr, body string) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("email: dialing %s: %w", addr, err)
	}
	defer c.Quit()

	if err := c.StartTLS(&tls.Config{ServerName: e.cfg.SMTPHost}); err != nil {
		return fmt.Errorf("email: STARTTLS: %w", err)
	}

	if e.cfg.Username != "" {
		auth := smtp.PlainAuth("", e.cfg.Username, e.getPassword(), e.cfg.SMTPHost)
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("email: auth: %w", err)
		}
	}

	if err := c.Mail(e.cfg.From); err != nil {
		return fmt.Errorf("email: MAIL FROM: %w", err)
	}
	for _, to := range e.cfg.To {
		if err := c.Rcpt(to); err != nil {
			return fmt.Errorf("email: RCPT TO %s: %w", to, err)
		}
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("email: DATA: %w", err)
	}
	if _, err := w.Write([]byte(body)); err != nil {
		return fmt.Errorf("email: writing body: %w", err)
	}
	return w.Close()
}

func (e *EmailNotifier) sendPlain(addr, body string) error {
	var auth smtp.Auth
	if e.cfg.Username != "" {
		auth = smtp.PlainAuth("", e.cfg.Username, e.getPassword(), e.cfg.SMTPHost)
	}
	return smtp.SendMail(addr, auth, e.cfg.From, e.cfg.To, []byte(body))
}

func (e *EmailNotifier) getPassword() string {
	return e.cfg.PasswordEnv // already resolved to the value by config.applyEnvOverrides
}

func buildEmailBody(msg Message) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("From: %s\r\n", "opsvault"))
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	sb.WriteString(fmt.Sprintf("Date: %s\r\n", msg.Timestamp.Format(time.RFC1123Z)))
	sb.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(msg.Body)
	return sb.String()
}
