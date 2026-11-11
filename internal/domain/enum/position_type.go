package enum

import (
	"fmt"
	"strings"
)

type PositionType string

const (
	PositionTypeLong  PositionType = "LONG"
	PositionTypeShort PositionType = "SHORT"
)

func (p PositionType) String() string {
	switch p {
	case PositionTypeLong:
		return "خریدار (Long)"
	case PositionTypeShort:
		return "فروشنده (Short)"
	default:
		return "unknown"
	}
}

func ParsePositionType(raw string) (PositionType, error) {
	switch PositionType(strings.ToUpper(strings.TrimSpace(raw))) {
	case PositionTypeLong:
		return PositionTypeLong, nil
	case PositionTypeShort:
		return PositionTypeShort, nil
	default:
		return "", fmt.Errorf("invalid position type %q: must be LONG or SHORT", raw)
	}
}
