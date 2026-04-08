package notification

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// SendEmail delivers an HTML email via the given SMTP config.
func SendEmail(cfg *SMTPConfig, to []string, subject, htmlBody string) error {
	if cfg == nil || cfg.Host == "" || len(to) == 0 {
		return fmt.Errorf("invalid smtp config or empty recipients")
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	from := cfg.FromEmail
	if from == "" {
		from = cfg.Username
	}

	msg := buildMIME(from, to, subject, htmlBody)

	switch strings.ToLower(cfg.Encryption) {
	case "tls":
		return sendTLS(addr, cfg, from, to, msg)
	case "starttls":
		return sendSTARTTLS(addr, cfg, from, to, msg)
	default:
		return sendPlain(addr, cfg, from, to, msg)
	}
}

func buildMIME(from string, to []string, subject, htmlBody string) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + strings.Join(to, ",") + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("Date: " + time.Now().UTC().Format(time.RFC1123Z) + "\r\n")
	b.WriteString("\r\n")
	b.WriteString(htmlBody)
	return []byte(b.String())
}

func authOrNil(cfg *SMTPConfig) smtp.Auth {
	if cfg.Username == "" && cfg.Password == "" {
		return nil
	}
	return smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
}

// sendTLS connects via implicit TLS (port 465 typical).
func sendTLS(addr string, cfg *SMTPConfig, from string, to []string, msg []byte) error {
	tlsCfg := &tls.Config{ServerName: cfg.Host}
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Close()

	return smtpSend(c, cfg, from, to, msg)
}

// sendSTARTTLS connects plain then upgrades (port 587 typical).
func sendSTARTTLS(addr string, cfg *SMTPConfig, from string, to []string, msg []byte) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer c.Close()

	if err := c.StartTLS(&tls.Config{ServerName: cfg.Host}); err != nil {
		return fmt.Errorf("starttls: %w", err)
	}

	return smtpSend(c, cfg, from, to, msg)
}

// sendPlain sends without encryption.
func sendPlain(addr string, cfg *SMTPConfig, from string, to []string, msg []byte) error {
	auth := authOrNil(cfg)
	return smtp.SendMail(addr, auth, from, to, msg)
}

func smtpSend(c *smtp.Client, cfg *SMTPConfig, from string, to []string, msg []byte) error {
	if auth := authOrNil(cfg); auth != nil {
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	for _, addr := range to {
		if err := c.Rcpt(addr); err != nil {
			return fmt.Errorf("rcpt %s: %w", addr, err)
		}
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("close data: %w", err)
	}
	return c.Quit()
}

// RenderNotificationEmail produces a simple HTML email body.
func RenderNotificationEmail(title string, fields []EmailField) string {
	var rows strings.Builder
	for _, f := range fields {
		rows.WriteString(fmt.Sprintf(
			`<tr><td style="padding:8px 12px;color:#666;width:140px;border-bottom:1px solid #f0f0f0;">%s</td>`+
				`<td style="padding:8px 12px;color:#333;border-bottom:1px solid #f0f0f0;">%s</td></tr>`,
			f.Label, f.Value))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="UTF-8"></head>
<body style="font-family:Arial,sans-serif;background:#f5f5f5;padding:20px;margin:0;">
<div style="max-width:600px;margin:0 auto;background:#fff;border-radius:8px;overflow:hidden;">
  <div style="background:#1890ff;padding:20px 24px;">
    <h2 style="color:#fff;margin:0;font-size:18px;">🔔 GLKVM Cloud Notification</h2>
  </div>
  <div style="padding:24px;">
    <p style="color:#333;font-size:16px;font-weight:bold;margin:0 0 16px;">%s</p>
    <table style="width:100%%;border-collapse:collapse;">%s</table>
  </div>
  <div style="padding:16px 24px;border-top:1px solid #f0f0f0;">
    <p style="color:#999;font-size:12px;margin:0;">This is an automated notification from GLKVM Cloud. Please do not reply.</p>
  </div>
</div>
</body></html>`, title, rows.String())
}

// EmailField is a label/value pair for the email template.
type EmailField struct {
	Label string
	Value string
}

// RenderTestEmail produces a test email body.
func RenderTestEmail() (subject, body string) {
	subject = "[GLKVM Cloud] Test Notification"
	body = RenderNotificationEmail("SMTP Configuration Test", []EmailField{
		{Label: "Status", Value: "✅ Success"},
		{Label: "Message", Value: "Your SMTP settings are configured correctly. You will receive notifications at this email address."},
		{Label: "Time", Value: time.Now().UTC().Format("2006-01-02 15:04:05 UTC")},
	})
	return
}
