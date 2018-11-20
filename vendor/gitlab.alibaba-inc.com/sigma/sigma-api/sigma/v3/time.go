package sigma_v3

import (
	"encoding/json"
	"time"
)

// Time is a wrapper around time.Time which supports unified JSON marshalling
// of format RFC3339 and timezone UTC.
type Time struct {
	time.Time
}

// MarshalJSON implements json.Marshaler interface
func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		// unset Time object wihtout `omitempty` tag would be shown as "null"
		return []byte("null"), nil
	}
	return json.Marshal(t.UTC().Format(time.RFC3339))
}

// UnmarshalJSON implements the json.Unmarshaller interface.
func (t *Time) UnmarshalJSON(b []byte) error {
	if len(b) == 4 && string(b) == "null" {
		t.Time = time.Time{}
		return nil
	}

	var str string
	err := json.Unmarshal(b, &str)
	if err != nil {
		return err
	}
	pt, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return err
	}
	t.Time = pt.Local()
	return nil
}
