package alerts

import (
	"strings"
	"testing"
	"time"
)

func TestFormatAlertBody_Triggered(t *testing.T) {
	detail := AlertDetail{
		RuleName:      "Low Liquid Balance",
		Status:        "triggered",
		ComputedValue: "4500.00",
		Threshold:     "5000",
		Comparison:    "<",
		Operands: []Operand{
			{Type: "bucket", Ref: "liquid", Label: "Liquid Balance", Operator: "+"},
			{Type: "account", Ref: "acct-1", Label: "Savings Acct", Operator: "-"},
		},
		OperandValues: map[string]string{
			"liquid": "6000.00",
			"acct-1": "1500.00",
		},
		Timestamp: time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC),
	}

	body := FormatAlertBody(detail)

	checks := []string{
		"Low Liquid Balance",
		"TRIGGERED",
		"4500.00",
		"< $5000",
		"Liquid Balance",
		"Savings Acct",
		"Finance Visualizer",
	}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Errorf("body missing %q.\nFull body:\n%s", check, body)
		}
	}
}

func TestFormatAlertBody_Recovered(t *testing.T) {
	detail := AlertDetail{
		RuleName:      "Low Liquid Balance",
		Status:        "recovered",
		ComputedValue: "5500.00",
		Threshold:     "5000",
		Comparison:    "<",
		Operands: []Operand{
			{Type: "bucket", Ref: "liquid", Label: "Liquid Balance", Operator: "+"},
		},
		OperandValues: map[string]string{
			"liquid": "5500.00",
		},
		Timestamp: time.Date(2024, 3, 15, 11, 0, 0, 0, time.UTC),
	}

	body := FormatAlertBody(detail)

	if !strings.Contains(body, "RECOVERED") {
		t.Errorf("body missing RECOVERED.\nFull body:\n%s", body)
	}
	if !strings.Contains(body, "5500.00") {
		t.Errorf("body missing computed value.\nFull body:\n%s", body)
	}
}

func TestFormatSubject_Triggered(t *testing.T) {
	subject := FormatSubject("Low Liquid", "triggered")
	want := "[Finance Alert] Low Liquid"
	if subject != want {
		t.Errorf("got %q, want %q", subject, want)
	}
}

func TestFormatSubject_Recovered(t *testing.T) {
	subject := FormatSubject("Low Liquid", "recovered")
	want := "[Finance Alert] Low Liquid -- recovered"
	if subject != want {
		t.Errorf("got %q, want %q", subject, want)
	}
}
