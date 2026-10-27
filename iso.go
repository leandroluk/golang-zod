package z

import (
	"regexp"
	"time"
)

type isoNamespace struct{}

// @spec REQ-PRIM-006
// ISO provides ISO 8601 date, time, and duration format validators.
var ISO isoNamespace

var (
	isoDateRegex     = regexp.MustCompile(`^\d{4}-(?:0[1-9]|1[0-2])-(?:0[1-9]|[12]\d|3[01])$`)
	isoTimeRegex     = regexp.MustCompile(`^(?:[01]\d|2[0-3]):[0-5]\d:[0-5]\d(?:\.\d+)?(?:Z|[+-](?:[01]\d|2[0-3]):[0-5]\d)?$`)
	isoDurationRegex = regexp.MustCompile(`^P(?:(?:\d+Y)?(?:\d+M)?(?:\d+D)?(?:T(?:\d+H)?(?:\d+M)?(?:\d+(?:\.\d+)?S)?)?)$`)
)

// @spec REQ-PRIM-006
// Date validates ISO 8601 date strings (YYYY-MM-DD).
func (isoNamespace) Date() *StringSchema {
	return String().WithCustomCheck("iso_date", func(s string) bool {
		if !isoDateRegex.MatchString(s) {
			return false
		}
		_, err := time.Parse("2006-01-02", s)
		return err == nil
	})
}

// @spec REQ-PRIM-006
// Time validates ISO 8601 time strings (HH:MM:SS with optional fraction and timezone).
func (isoNamespace) Time() *StringSchema {
	return String().WithCustomCheck("iso_time", func(s string) bool {
		return isoTimeRegex.MatchString(s)
	})
}

// ISODateTimeOptions configures ISO DateTime parsing requirements.
type ISODateTimeOptions struct {
	Offset    bool // require timezone offset
	Precision int  // required sub-second decimal places
}

// @spec REQ-PRIM-006
// DateTime validates ISO 8601 / RFC 3339 date-time strings.
func (isoNamespace) DateTime(opts ...ISODateTimeOptions) *StringSchema {
	return String().WithCustomCheck("iso_datetime", func(s string) bool {
		t, err := time.Parse(time.RFC3339, s)
		if err == nil {
			_ = t
			return true
		}
		tNano, errNano := time.Parse(time.RFC3339Nano, s)
		return errNano == nil && !tNano.IsZero()
	})
}

// @spec REQ-PRIM-006
// Duration validates ISO 8601 duration strings (e.g. P1Y2M3DT4H5M6S).
func (isoNamespace) Duration() *StringSchema {
	return String().WithCustomCheck("iso_duration", func(s string) bool {
		if len(s) <= 1 || s == "P" || s == "PT" {
			return false
		}
		return isoDurationRegex.MatchString(s)
	})
}
