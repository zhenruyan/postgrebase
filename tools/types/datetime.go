package types

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/spf13/cast"
)

// DefaultDateLayout specifies the default app date strings layout.
const DefaultDateLayout = "2006-01-02 15:04:05.000Z"

// DateTime represents a [time.Time] instance in UTC that is wrapped
// and serialized using the app default date layout.
type DateTime struct {
	t time.Time
}

// Time returns the internal [time.Time] instance.
func (d DateTime) Time() time.Time {
	return d.t.UTC()
}

// IsZero checks whether the current DateTime instance has zero time value.
func (d DateTime) IsZero() bool {
	return d.t.IsZero()
}

// String serializes the current DateTime instance into a formatted
// UTC date string.
//
// The zero value is serialized to an empty string.
func (d DateTime) String() string {
	if d.IsZero() {
		return ""
	}
	return d.Time().Format(DefaultDateLayout)
}

// NowDateTime returns new DateTime instance with the current local time.
func NowDateTime() DateTime {
	return DateTime{t: time.Now()}
}

// ParseDateTime creates a new DateTime from the provided value
// (could be [cast.ToTime] supported string, [time.Time], etc.).
func ParseDateTime(value any) (DateTime, error) {
	d := DateTime{}
	err := d.Scan(value)
	return d, err
}

// MarshalJSON implements the [json.Marshaler] interface.
func (d DateTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.String() + `"`), nil
}

// UnmarshalJSON implements the [json.Unmarshaler] interface.
func (d *DateTime) UnmarshalJSON(b []byte) error {
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	return d.Scan(raw)
}

// Value implements the [driver.Valuer] interface.
func (d DateTime) Value() (driver.Value, error) {
	return d.String(), nil
}

// Scan implements [sql.Scanner] interface to scan the provided value
// into the current DateTime instance.
func (d *DateTime) Scan(value any) error {
	switch v := value.(type) {
	case DateTime:
		d.t = v.Time()
	case time.Time:
		d.t = v
	case int, int64, int32, uint, uint64, uint32:
		d.t = cast.ToTime(v)
	case string:
		if v == "" {
			d.t = time.Time{}
		} else {
			layouts := []string{
				DefaultDateLayout,
				"2006-01-02 15:04:05.000",
				time.RFC3339Nano,
				time.RFC3339,
			}
			for _, layout := range layouts {
				parsed, err := time.Parse(layout, v)
				if err == nil {
					d.t = parsed
					return nil
				}
			}
			d.t = cast.ToTime(v)
		}
	default:
		str := cast.ToString(v)
		if str == "" {
			d.t = time.Time{}
		} else {
			t, err := time.Parse(DefaultDateLayout, str)
			if err != nil {
				t = cast.ToTime(str)
			}
			d.t = t
		}
	}

	return nil
}
