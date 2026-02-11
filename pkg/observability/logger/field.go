package logger

import (
	"fmt"
	"time"
)

// Field is a structured key-value pair attached to a log entry.
type Field struct {
	Key   string
	Value any
}

// Typed constructors for common field types.

func String(key, val string) Field                 { return Field{Key: key, Value: val} }
func Int(key string, val int) Field                { return Field{Key: key, Value: val} }
func Int64(key string, val int64) Field            { return Field{Key: key, Value: val} }
func Float64(key string, val float64) Field        { return Field{Key: key, Value: val} }
func Bool(key string, val bool) Field              { return Field{Key: key, Value: val} }
func Err(err error) Field                          { return Field{Key: "error", Value: err} }
func Duration(key string, val time.Duration) Field { return Field{Key: key, Value: val} }
func Any(key string, val any) Field                { return Field{Key: key, Value: val} }

// FormatValue returns a string representation of a field's value.
func (f Field) FormatValue() string {
	if f.Value == nil {
		return "<nil>"
	}
	switch v := f.Value.(type) {
	case string:
		return v
	case error:
		return v.Error()
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}
