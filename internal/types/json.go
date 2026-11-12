package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSON preserves JSON values across both PostgreSQL and SQLite drivers.
type JSON json.RawMessage

func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "[]", nil
	}
	return string(j), nil
}
func (j *JSON) Scan(v any) error {
	switch x := v.(type) {
	case nil:
		*j = JSON("[]")
	case []byte:
		*j = append((*j)[:0], x...)
	case string:
		*j = JSON(x)
	default:
		return fmt.Errorf("invalid JSON database value %T", v)
	}
	return nil
}
func (j JSON) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("[]"), nil
	}
	return j, nil
}
func (j *JSON) UnmarshalJSON(b []byte) error {
	if !json.Valid(b) {
		return fmt.Errorf("invalid JSON")
	}
	*j = append((*j)[:0], b...)
	return nil
}
func AsJSON(v any) JSON { b, _ := json.Marshal(v); return b }
