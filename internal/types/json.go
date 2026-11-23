package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSON preserves JSON values across both PostgreSQL and SQLite drivers.
type JSON json.RawMessage

func (raw JSON) Value() (driver.Value, error) {
	if len(raw) == 0 {
		return "[]", nil
	}
	return string(raw), nil
}
func (raw *JSON) Scan(value any) error {
	switch value := value.(type) {
	case nil:
		*raw = JSON("[]")
	case []byte:
		*raw = append((*raw)[:0], value...)
	case string:
		*raw = JSON(value)
	default:
		return fmt.Errorf("invalid JSON database value %T", value)
	}
	return nil
}
func (raw JSON) MarshalJSON() ([]byte, error) {
	if len(raw) == 0 {
		return []byte("[]"), nil
	}
	return raw, nil
}
func (raw *JSON) UnmarshalJSON(data []byte) error {
	if !json.Valid(data) {
		return fmt.Errorf("invalid JSON")
	}
	*raw = append((*raw)[:0], data...)
	return nil
}
func AsJSON(value any) JSON	{ encoded, _ := json.Marshal(value); return encoded }
