package marketdata

import (
	"testing"
	"time"
)

func TestWindowRejectsAmbiguousAndInvertedBounds(t *testing.T) {
	from := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	for _, w := range []Window{{}, {From: from, To: from}, {From: from.Add(time.Hour), To: from.AddDate(0, 0, 1)}, {From: from, To: from.AddDate(21, 0, 0)}} {
		if w.Validate() == nil {
			t.Fatalf("accepted invalid range %+v", w)
		}
	}
	if err := (Window{From: from, To: from.AddDate(0, 0, 1)}).Validate(); err != nil {
		t.Fatal(err)
	}
}
