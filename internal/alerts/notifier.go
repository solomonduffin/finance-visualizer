package alerts

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	mail "github.com/wneessen/go-mail"
)

// SMTPConfig holds the SMTP server configuration for sending alert emails.
type SMTPConfig struct {
	Host        string
	Port        int
	Username    string
	Password    string
	FromAddress string
	ToAddress   string
}

// AlertDetail holds the information needed to format an alert email.
type AlertDetail struct {
	RuleName      string
	Status        string // "triggered" or "recovered"
	ComputedValue string
	Threshold     string
	Comparison    string
	Operands      []Operand
	OperandValues map[string]string // operand ref -> formatted value
	Timestamp     time.Time
}

// FormatSubject returns the email subject line for an alert.
// Triggered: "[Finance Alert] {name}"
// Recovered: "[Finance Alert] {name} -- recovered"
func FormatSubject(ruleName, state string) string {
	if state == "recovered" {
		return fmt.Sprintf("[Finance Alert] %s -- recovered", ruleName)
	}
	return fmt.Sprintf("[Finance Alert] %s", ruleName)
}

// FormatAlertBody formats the plain-text email body for an alert notification.
func FormatAlertBody(detail AlertDetail) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Alert: %s\n", detail.RuleName))
	b.WriteString(fmt.Sprintf("Status: %s\n", strings.ToUpper(detail.Status)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Computed Value: $%s\n", detail.ComputedValue))
	b.WriteString(fmt.Sprintf("Threshold: %s $%s\n", detail.Comparison, detail.Threshold))
	b.WriteString(fmt.Sprintf("Time: %s\n", detail.Timestamp.Format(time.RFC822)))
	b.WriteString("\n")
	b.WriteString("Account Breakdown:\n")
	for _, op := range detail.Operands {
		val := detail.OperandValues[op.Ref]
		b.WriteString(fmt.Sprintf("  - %s: $%s\n", op.Label, val))
	}
	b.WriteString("\n--\nFinance Visualizer\n")

	return b.String()
}

// SendAlert sends an alert email using the provided SMTP configuration.
func SendAlert(ctx context.Context, cfg SMTPConfig, detail AlertDetail) error {
	msg := mail.NewMsg()
	if err := msg.From(cfg.FromAddress); err != nil {
		return fmt.Errorf("set from: %w", err)
	}
	if err := msg.To(cfg.ToAddress); err != nil {
		return fmt.Errorf("set to: %w", err)
	}
	msg.Subject(FormatSubject(detail.RuleName, detail.Status))
	msg.SetBodyString(mail.TypeTextPlain, FormatAlertBody(detail))

	client, err := mail.NewClient(cfg.Host,
		mail.WithPort(cfg.Port),
		mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
		mail.WithUsername(cfg.Username),
		mail.WithPassword(cfg.Password),
	)
	if err != nil {
		return fmt.Errorf("create mail client: %w", err)
	}

	if err := client.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}

// LoadSMTPConfig loads SMTP configuration from the settings table.
// Returns nil (not an error) if smtp_host is not configured.
// The SMTP password is decrypted using a key derived from jwtSecret.
func LoadSMTPConfig(ctx context.Context, db *sql.DB, jwtSecret string) (*SMTPConfig, error) {
	settings := make(map[string]string)
	keys := []string{"smtp_host", "smtp_port", "smtp_username", "smtp_password", "smtp_from", "smtp_to"}

	for _, key := range keys {
		var value string
		err := db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("query setting %s: %w", key, err)
		}
		settings[key] = value
	}

	host := settings["smtp_host"]
	if host == "" {
		return nil, nil // SMTP not configured
	}

	port := 587
	if portStr, ok := settings["smtp_port"]; ok && portStr != "" {
		p, err := strconv.Atoi(portStr)
		if err == nil {
			port = p
		}
	}

	password := settings["smtp_password"]
	if password != "" {
		key := DeriveKey(jwtSecret)
		decrypted, err := Decrypt(password, key)
		if err != nil {
			return nil, fmt.Errorf("decrypt smtp password: %w", err)
		}
		password = decrypted
	}

	return &SMTPConfig{
		Host:        host,
		Port:        port,
		Username:    settings["smtp_username"],
		Password:    password,
		FromAddress: settings["smtp_from"],
		ToAddress:   settings["smtp_to"],
	}, nil
}
