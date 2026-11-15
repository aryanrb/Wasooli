package services

import (
	"testing"
	"time"

	"wasooli/internal/db/sqlc"
)

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// termTotalAt must pick the rent term in effect at a given date, so a mid-contract
// rent increase only affects periods on/after its effective date.
func TestTermTotalAt(t *testing.T) {
	terms := []sqlc.ContractTerm{
		{EffectiveFrom: day(2026, time.January, 1), Total: 1000000}, // 10,000.00
		{EffectiveFrom: day(2026, time.March, 1), Total: 1200000},   // raised to 12,000.00
	}
	cases := []struct {
		at   time.Time
		want int64
	}{
		{day(2026, time.February, 15), 1000000}, // before raise
		{day(2026, time.March, 1), 1200000},     // on the raise date
		{day(2026, time.June, 1), 1200000},      // after raise
		{day(2025, time.December, 1), 0},         // before any term
	}
	for _, c := range cases {
		if got := termTotalAt(terms, c.at); got != c.want {
			t.Errorf("termTotalAt(%s) = %d, want %d", c.at.Format("2006-01-02"), got, c.want)
		}
	}
}

// periodStatus drives the Due/Partial/Paid chips, including the partial-payment case.
func TestPeriodStatus(t *testing.T) {
	cases := []struct {
		due, paid int64
		want      string
	}{
		{1000000, 0, "due"},
		{1000000, 400000, "partial"},
		{1000000, 1000000, "paid"},
		{1000000, 1200000, "paid"}, // overpaid still reads as paid
		{0, 0, "due"},
	}
	for _, c := range cases {
		if got := periodStatus(c.due, c.paid); got != c.want {
			t.Errorf("periodStatus(due=%d paid=%d) = %q, want %q", c.due, c.paid, got, c.want)
		}
	}
}
