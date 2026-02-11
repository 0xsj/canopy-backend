package types

import "encoding/base64"

// Direction indicates cursor traversal direction.
type Direction string

const (
	DirectionForward  Direction = "forward"
	DirectionBackward Direction = "backward"
)

// DefaultPageSize is the default number of items per page.
const DefaultPageSize = 20

// MaxPageSize is the maximum allowed items per page.
const MaxPageSize = 100

// PageRequest describes a cursor-based pagination request.
type PageRequest struct {
	Cursor    string    `json:"cursor,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	Direction Direction `json:"direction,omitempty"`
}

// EffectiveLimit returns the limit clamped to [1, MaxPageSize],
// defaulting to DefaultPageSize if unset.
func (p PageRequest) EffectiveLimit() int {
	if p.Limit <= 0 {
		return DefaultPageSize
	}
	if p.Limit > MaxPageSize {
		return MaxPageSize
	}
	return p.Limit
}

// EffectiveDirection returns the direction, defaulting to forward.
func (p PageRequest) EffectiveDirection() Direction {
	if p.Direction == DirectionBackward {
		return DirectionBackward
	}
	return DirectionForward
}

// PageResponse carries pagination metadata alongside a result set.
type PageResponse[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// NewPageResponse creates a PageResponse from a slice of items.
// If the slice has more items than the limit, it trims to the limit
// and sets HasMore to true, using cursorFn to derive the next cursor.
func NewPageResponse[T any](items []T, limit int, cursorFn func(T) string) PageResponse[T] {
	if len(items) > limit {
		last := items[limit-1]
		return PageResponse[T]{
			Items:      items[:limit],
			NextCursor: cursorFn(last),
			HasMore:    true,
		}
	}
	return PageResponse[T]{
		Items:   items,
		HasMore: false,
	}
}

// EncodeCursor encodes a raw cursor value to an opaque base64 string.
// This hides internal details (timestamps, IDs) from the client.
func EncodeCursor(raw string) string {
	return base64.URLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor decodes an opaque cursor back to its raw value.
// Returns an empty string if decoding fails.
func DecodeCursor(encoded string) string {
	b, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return ""
	}
	return string(b)
}
