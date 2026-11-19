// Package json provides JSON parsing utilities for Sidekiq job data.
package json

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"io"
	"time"
)

// Parse parses a JSON string into a map.
func Parse(data string) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		return nil, err
	}
	return m, nil
}

// GetString returns a string value from a map, or empty string if not found/wrong type.
func GetString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetInt64 returns an int64 value from a map, or 0 if not found/wrong type.
// Handles both int64 and float64 (JSON numbers).
func GetInt64(m map[string]any, key string) int64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int64(n)
		case int64:
			return n
		case int:
			return int64(n)
		}
	}
	return 0
}

// GetFloat64 returns a float64 value from a map, or 0 if not found/wrong type.
func GetFloat64(m map[string]any, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}

// GetBool returns a bool value from a map, or false if not found/wrong type.
func GetBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// millisecondThreshold separates second-based from millisecond-based Unix
// timestamps. 1e11 seconds is year 5138; 1e11 milliseconds is March 1973.
// Any realistic job timestamp falls clearly on one side.
const millisecondThreshold = 1e11

// GetTime returns a time.Time from a Unix timestamp, or zero time if not found.
//
// Sidekiq < 8 stores timestamps as fractional seconds (e.g. 1700000000.123).
// Sidekiq 8 stores them as integer milliseconds (e.g. 1700000000123).
// Ruby's Sidekiq::API distinguishes them by type (Float vs Integer), but JSON
// numbers decode to float64 in Go, so the magnitude is used instead.
func GetTime(m map[string]any, key string) time.Time {
	f := GetFloat64(m, key)
	if f <= 0 {
		return time.Time{}
	}
	if f >= millisecondThreshold {
		return time.UnixMilli(int64(f))
	}
	sec := int64(f)
	return time.Unix(sec, int64((f-float64(sec))*1e9))
}

// GetStringSlice returns a string slice from a map, or nil if not found/wrong type.
func GetStringSlice(m map[string]any, key string) []string {
	if v, ok := m[key]; ok {
		if arr, ok := v.([]any); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return nil
}

// GetSlice returns an interface slice from a map, or nil if not found/wrong type.
func GetSlice(m map[string]any, key string) []any {
	if v, ok := m[key]; ok {
		if arr, ok := v.([]any); ok {
			return arr
		}
	}
	return nil
}

// GetMap returns a map from a map, or nil if not found/wrong type.
func GetMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if mp, ok := v.(map[string]any); ok {
			return mp
		}
	}
	return nil
}

// DecompressBacktrace decompresses a base64+zlib compressed backtrace string.
// Sidekiq compresses backtraces to save Redis memory.
func DecompressBacktrace(data string) ([]string, error) {
	if data == "" {
		return nil, nil
	}

	// Base64 decode
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}

	// Zlib decompress
	r, err := zlib.NewReader(bytes.NewReader(decoded))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	decompressed, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// Parse JSON array
	var lines []string
	if err := json.Unmarshal(decompressed, &lines); err != nil {
		return nil, err
	}

	return lines, nil
}

// Marshal serializes a value to JSON string.
func Marshal(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
