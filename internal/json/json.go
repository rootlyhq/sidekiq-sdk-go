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
func Parse(data string) (map[string]interface{}, error) {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		return nil, err
	}
	return m, nil
}

// GetString returns a string value from a map, or empty string if not found/wrong type.
func GetString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetInt64 returns an int64 value from a map, or 0 if not found/wrong type.
// Handles both int64 and float64 (JSON numbers).
func GetInt64(m map[string]interface{}, key string) int64 {
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
func GetFloat64(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}

// GetBool returns a bool value from a map, or false if not found/wrong type.
func GetBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// GetTime returns a time.Time from a Unix timestamp float64, or zero time if not found.
func GetTime(m map[string]interface{}, key string) time.Time {
	if f := GetFloat64(m, key); f > 0 {
		return time.Unix(int64(f), int64((f-float64(int64(f)))*1e9))
	}
	return time.Time{}
}

// GetStringSlice returns a string slice from a map, or nil if not found/wrong type.
func GetStringSlice(m map[string]interface{}, key string) []string {
	if v, ok := m[key]; ok {
		if arr, ok := v.([]interface{}); ok {
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
func GetSlice(m map[string]interface{}, key string) []interface{} {
	if v, ok := m[key]; ok {
		if arr, ok := v.([]interface{}); ok {
			return arr
		}
	}
	return nil
}

// GetMap returns a map from a map, or nil if not found/wrong type.
func GetMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key]; ok {
		if mp, ok := v.(map[string]interface{}); ok {
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
func Marshal(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
