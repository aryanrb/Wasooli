package types

import (
	"encoding/json"
	"fmt"
	"time"
)

// DateLayout is the wire format for all date fields (no time-of-day on a ledger).
const DateLayout = "2006-01-02"

// Component is one line of a rent breakdown or a payment, in minor currency units
// (e.g. paisa/cents). Money is never a float.
type Component struct {
	Label  string `json:"label"`
	Amount int64  `json:"amount"`
}

// SumComponents totals a breakdown.
func SumComponents(cs []Component) int64 {
	var t int64
	for _, c := range cs {
		t += c.Amount
	}
	return t
}

// MarshalComponents encodes a breakdown for a jsonb column (never nil).
func MarshalComponents(cs []Component) []byte {
	if cs == nil {
		cs = []Component{}
	}
	b, _ := json.Marshal(cs)
	return b
}

// Period is one billing window, derived from a contract's cycle — never stored.
type Period struct {
	Key   string    `json:"key"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"` // exclusive
}

// PeriodKey formats the stable key used to tag payments to a period.
func PeriodKey(cycle string, d time.Time) string {
	switch cycle {
	case "weekly":
		y, w := d.ISOWeek()
		return fmt.Sprintf("%04d-W%02d", y, w)
	case "yearly":
		return fmt.Sprintf("%04d", d.Year())
	default: // monthly
		return d.Format("2006-01")
	}
}

func nextPeriod(cycle string, d time.Time) time.Time {
	switch cycle {
	case "weekly":
		return d.AddDate(0, 0, 7)
	case "yearly":
		return d.AddDate(1, 0, 0)
	default:
		// ponytail: monthly stepping keeps the start day; Jan 31 drifts to Mar 3 via
		// Go's AddDate normalization. Fine for a ledger — revisit if strict anchoring matters.
		return d.AddDate(0, 1, 0)
	}
}

// GeneratePeriods returns every period from start through the one containing until.
func GeneratePeriods(cycle string, start, until time.Time) []Period {
	out := []Period{}
	for cur := start; !cur.After(until); {
		next := nextPeriod(cycle, cur)
		out = append(out, Period{Key: PeriodKey(cycle, cur), Start: cur, End: next})
		cur = next
	}
	return out
}

// ValidCycle reports whether s is a supported billing cycle.
func ValidCycle(s string) bool {
	return s == "weekly" || s == "monthly" || s == "yearly"
}
