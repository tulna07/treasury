package email

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTemplateRenderer(t *testing.T) {
	r, err := NewTemplateRenderer()
	require.NoError(t, err)
	assert.NotNil(t, r)
	assert.Contains(t, r.templates, "deal_cancelled")
	assert.Contains(t, r.templates, "deal_voided")
	assert.Contains(t, r.templates, "generic_notification")
}

func TestRenderDealCancelled(t *testing.T) {
	r, err := NewTemplateRenderer()
	require.NoError(t, err)

	data := map[string]string{
		"DealModule":       "FX",
		"TicketNumber":     "FX-2026-001",
		"CounterpartyName": "Vietcombank",
		"Amount":           "1,000,000",
		"Currency":         "USD",
		"CancelReason":     "Khách hàng yêu cầu hủy",
		"RequestedBy":      "Nguyễn Văn A",
		"ApprovedBy":       "Trần Thị B",
		"CancelledAt":      "04/04/2026 10:30:00",
	}

	html, text, err := r.Render("deal_cancelled", data)
	require.NoError(t, err)

	// HTML checks
	assert.Contains(t, html, "Thông báo hủy giao dịch")
	assert.Contains(t, html, "FX-2026-001")
	assert.Contains(t, html, "Vietcombank")
	assert.Contains(t, html, "1,000,000")
	assert.Contains(t, html, "USD")
	assert.Contains(t, html, "Khách hàng yêu cầu hủy")
	assert.Contains(t, html, "Nguyễn Văn A")
	assert.Contains(t, html, "Trần Thị B")
	assert.Contains(t, html, "KienlongBank Treasury")

	// Plain text checks
	assert.Contains(t, text, "FX-2026-001")
	assert.Contains(t, text, "Vietcombank")
	assert.NotContains(t, text, "<table")
}

func TestRenderDealVoided(t *testing.T) {
	r, err := NewTemplateRenderer()
	require.NoError(t, err)

	data := map[string]string{
		"DealModule":       "FX",
		"TicketNumber":     "FX-2026-002",
		"CounterpartyName": "BIDV",
		"Amount":           "500,000",
		"Currency":         "EUR",
		"VoidReason":       "Bút toán sai",
		"VoidedBy":         "Lê Văn C",
		"VoidedAt":         "04/04/2026 14:00:00",
	}

	html, text, err := r.Render("deal_voided", data)
	require.NoError(t, err)

	assert.Contains(t, html, "Thông báo hủy bút toán (Void)")
	assert.Contains(t, html, "FX-2026-002")
	assert.Contains(t, html, "BIDV")
	assert.Contains(t, html, "Bút toán sai")
	assert.Contains(t, html, "Lê Văn C")

	assert.Contains(t, text, "FX-2026-002")
	assert.NotContains(t, text, "<table")
}

func TestRenderGeneric(t *testing.T) {
	r, err := NewTemplateRenderer()
	require.NoError(t, err)

	data := map[string]string{
		"Title":        "Thông báo hệ thống",
		"Message":      "Hệ thống sẽ bảo trì vào 22:00 ngày 05/04/2026.",
		"DealModule":   "FX",
		"TicketNumber": "FX-2026-003",
	}

	html, text, err := r.Render("generic_notification", data)
	require.NoError(t, err)

	assert.Contains(t, html, "Thông báo hệ thống")
	assert.Contains(t, html, "bảo trì")
	assert.Contains(t, html, "FX-2026-003")

	assert.Contains(t, text, "bảo trì")
}

func TestRenderGeneric_NoTicket(t *testing.T) {
	r, err := NewTemplateRenderer()
	require.NoError(t, err)

	data := map[string]string{
		"Title":   "Thông báo",
		"Message": "Nội dung thông báo.",
	}

	html, _, err := r.Render("generic_notification", data)
	require.NoError(t, err)

	assert.Contains(t, html, "Thông báo")
	// No deal reference table when TicketNumber is empty
	assert.NotContains(t, html, "Mã giao dịch")
}

func TestRenderUnknownTemplate(t *testing.T) {
	r, err := NewTemplateRenderer()
	require.NoError(t, err)

	_, _, err = r.Render("nonexistent_template", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
