package email

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/kienlongbank/treasury-api/internal/config"
)

func TestNewSMTPSender(t *testing.T) {
	cfg := config.EmailConfig{
		Host:        "smtp.example.com",
		Port:        587,
		Username:    "user",
		Password:    "pass",
		FromAddress: "test@example.com",
		FromName:    "Test Sender",
		UseTLS:      true,
	}

	sender := NewSMTPSender(cfg, zap.NewNop())

	assert.Equal(t, "smtp.example.com", sender.host)
	assert.Equal(t, 587, sender.port)
	assert.Equal(t, "user", sender.username)
	assert.Equal(t, "pass", sender.password)
	assert.Equal(t, "test@example.com", sender.from)
	assert.Equal(t, "Test Sender", sender.fromName)
	assert.True(t, sender.useTLS)
}

func TestNewSMTPSender_Defaults(t *testing.T) {
	cfg := config.EmailConfig{
		Host:        "localhost",
		Port:        1025,
		FromAddress: "treasury@kienlongbank.com",
		FromName:    "KienlongBank Treasury",
	}

	sender := NewSMTPSender(cfg, zap.NewNop())

	assert.Equal(t, "localhost", sender.host)
	assert.Equal(t, 1025, sender.port)
	assert.Empty(t, sender.username)
	assert.False(t, sender.useTLS)
}
