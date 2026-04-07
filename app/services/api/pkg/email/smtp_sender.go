package email

import (
	"context"
	"fmt"

	gomail "github.com/wneessen/go-mail"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/internal/config"
)

// SMTPSender sends emails via SMTP using go-mail.
type SMTPSender struct {
	host     string
	port     int
	username string
	password string
	useTLS   bool
	from     string
	fromName string
	logger   *zap.Logger
}

// NewSMTPSender creates a new SMTP sender from config.
func NewSMTPSender(cfg config.EmailConfig, logger *zap.Logger) *SMTPSender {
	return &SMTPSender{
		host:     cfg.Host,
		port:     cfg.Port,
		username: cfg.Username,
		password: cfg.Password,
		useTLS:   cfg.UseTLS,
		from:     cfg.FromAddress,
		fromName: cfg.FromName,
		logger:   logger,
	}
}

// Send sends a single email via SMTP.
func (s *SMTPSender) Send(ctx context.Context, msg *OutboxEmail) error {
	m := gomail.NewMsg()

	if err := m.EnvelopeFromFormat(s.fromName, s.from); err != nil {
		return fmt.Errorf("set from: %w", err)
	}

	if err := m.To(msg.ToAddresses...); err != nil {
		return fmt.Errorf("set to: %w", err)
	}

	if len(msg.CCAddresses) > 0 {
		if err := m.Cc(msg.CCAddresses...); err != nil {
			return fmt.Errorf("set cc: %w", err)
		}
	}

	m.Subject(msg.Subject)

	if msg.BodyHTML != "" {
		m.SetBodyString(gomail.TypeTextHTML, msg.BodyHTML)
		if msg.BodyText != "" {
			m.AddAlternativeString(gomail.TypeTextPlain, msg.BodyText)
		}
	} else if msg.BodyText != "" {
		m.SetBodyString(gomail.TypeTextPlain, msg.BodyText)
	}

	// Build client options
	opts := []gomail.Option{
		gomail.WithPort(s.port),
		gomail.WithTimeout(gomail.DefaultTimeout),
	}

	if s.useTLS {
		opts = append(opts, gomail.WithTLSPortPolicy(gomail.TLSMandatory))
	} else {
		opts = append(opts, gomail.WithTLSPortPolicy(gomail.NoTLS))
	}

	if s.username != "" {
		opts = append(opts,
			gomail.WithSMTPAuth(gomail.SMTPAuthPlain),
			gomail.WithUsername(s.username),
			gomail.WithPassword(s.password),
		)
	}

	client, err := gomail.NewClient(s.host, opts...)
	if err != nil {
		return fmt.Errorf("create smtp client: %w", err)
	}

	if err := client.DialAndSendWithContext(ctx, m); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}

	return nil
}
