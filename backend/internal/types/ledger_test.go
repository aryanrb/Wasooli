package types

import (
	"testing"
	"time"
)

func d(y int, m time.Month, day int) time.Time {
	return time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
}

func TestSumComponents(t *testing.T) {
	// rent 10000.00 + water 1000.00 + internet 1000.00, in minor units
	got := SumComponents([]Component{{"rent", 1000000}, {"water", 100000}, {"internet", 100000}})
	if got != 1200000 {
		t.Fatalf("SumComponents = %d, want 1200000", got)
	}
	if SumComponents(nil) != 0 {
		t.Fatal("empty breakdown should total 0")
	}
}

func TestGeneratePeriodsMonthly(t *testing.T) {
	ps := GeneratePeriods("monthly", d(2026, time.January, 1), d(2026, time.March, 15))
	if len(ps) != 3 {
		t.Fatalf("got %d periods, want 3", len(ps))
	}
	want := []string{"2026-01", "2026-02", "2026-03"}
	for i, w := range want {
		if ps[i].Key != w {
			t.Errorf("period %d key = %q, want %q", i, ps[i].Key, w)
		}
	}
}

func TestPeriodKey(t *testing.T) {
	if got := PeriodKey("yearly", d(2026, time.July, 9)); got != "2026" {
		t.Errorf("yearly key = %q, want 2026", got)
	}
	// 2026-01-01 falls in ISO week 1.
	if got := PeriodKey("weekly", d(2026, time.January, 1)); got != "2026-W01" {
		t.Errorf("weekly key = %q, want 2026-W01", got)
	}
}

func TestValidCycle(t *testing.T) {
	for _, c := range []string{"weekly", "monthly", "yearly"} {
		if !ValidCycle(c) {
			t.Errorf("%q should be valid", c)
		}
	}
	if ValidCycle("daily") {
		t.Error("daily should be invalid")
	}
}
