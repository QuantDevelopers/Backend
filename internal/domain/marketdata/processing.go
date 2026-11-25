package marketdata

import (
	"context"
	"fmt"
	"time"
)

// Window is inclusive of From and exclusive of To. Dates refer to trading days.
type Window struct{ From, To time.Time }

func (w Window) Validate() error {
	if w.From.IsZero() || !w.From.Before(w.To) || w.To.Sub(w.From) > 366*24*time.Hour*20 {
		return fmt.Errorf("require a nonempty date range of at most 20 years")
	}
	for _, d := range []time.Time{w.From, w.To} {
		if d.Hour() != 0 || d.Minute() != 0 || d.Second() != 0 || d.Nanosecond() != 0 || d.Location() != time.UTC {
			return fmt.Errorf("date boundaries must be UTC midnight")
		}
	}
	return nil
}

type Result struct {
	Candles     int64 `json:"candles"`
	OptionTicks int64 `json:"option_ticks"`
}

// Processor consumes the clean layer produced by E4-1; it never cleans raw ticks.
type Processor interface {
	Process(context.Context, Window, time.Duration) (Result, error)
}
