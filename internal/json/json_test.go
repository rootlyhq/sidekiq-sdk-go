package json

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid JSON",
			input:   `{"key":"value","num":42}`,
			wantErr: false,
		},
		{
			name:    "empty object",
			input:   `{}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			input:   `{invalid}`,
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   ``,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetString(t *testing.T) {
	m := map[string]interface{}{
		"name":   "test",
		"number": 42,
		"nil":    nil,
	}

	tests := []struct {
		key  string
		want string
	}{
		{"name", "test"},
		{"number", ""},  // wrong type
		{"missing", ""}, // not found
		{"nil", ""},     // nil value
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := GetString(m, tt.key); got != tt.want {
				t.Errorf("GetString(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestGetInt64(t *testing.T) {
	m := map[string]interface{}{
		"float":  float64(42),
		"int64":  int64(100),
		"int":    int(200),
		"string": "not a number",
		"nil":    nil,
	}

	tests := []struct {
		key  string
		want int64
	}{
		{"float", 42},
		{"int64", 100},
		{"int", 200},
		{"string", 0},  // wrong type
		{"missing", 0}, // not found
		{"nil", 0},     // nil value
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := GetInt64(m, tt.key); got != tt.want {
				t.Errorf("GetInt64(%q) = %d, want %d", tt.key, got, tt.want)
			}
		})
	}
}

func TestGetFloat64(t *testing.T) {
	m := map[string]interface{}{
		"float":  float64(3.14),
		"string": "not a number",
	}

	tests := []struct {
		key  string
		want float64
	}{
		{"float", 3.14},
		{"string", 0},  // wrong type
		{"missing", 0}, // not found
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := GetFloat64(m, tt.key); got != tt.want {
				t.Errorf("GetFloat64(%q) = %f, want %f", tt.key, got, tt.want)
			}
		})
	}
}

func TestGetBool(t *testing.T) {
	m := map[string]interface{}{
		"true":   true,
		"false":  false,
		"string": "true",
	}

	tests := []struct {
		key  string
		want bool
	}{
		{"true", true},
		{"false", false},
		{"string", false},  // wrong type
		{"missing", false}, // not found
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := GetBool(m, tt.key); got != tt.want {
				t.Errorf("GetBool(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestGetTime(t *testing.T) {
	now := time.Now()
	unixFloat := float64(now.Unix()) + float64(now.Nanosecond())/1e9

	m := map[string]interface{}{
		"time":   unixFloat,
		"zero":   float64(0),
		"string": "not a time",
	}

	t.Run("valid time", func(t *testing.T) {
		got := GetTime(m, "time")
		if got.Unix() != now.Unix() {
			t.Errorf("GetTime() unix = %d, want %d", got.Unix(), now.Unix())
		}
	})

	t.Run("zero", func(t *testing.T) {
		got := GetTime(m, "zero")
		if !got.IsZero() {
			t.Errorf("GetTime(zero) should return zero time, got %v", got)
		}
	})

	t.Run("missing", func(t *testing.T) {
		got := GetTime(m, "missing")
		if !got.IsZero() {
			t.Errorf("GetTime(missing) should return zero time, got %v", got)
		}
	})
}

func TestGetStringSlice(t *testing.T) {
	m := map[string]interface{}{
		"tags":    []interface{}{"tag1", "tag2", "tag3"},
		"mixed":   []interface{}{"str", 42, "another"},
		"numbers": []interface{}{1, 2, 3},
		"string":  "not a slice",
	}

	t.Run("valid slice", func(t *testing.T) {
		got := GetStringSlice(m, "tags")
		want := []string{"tag1", "tag2", "tag3"}
		if len(got) != len(want) {
			t.Fatalf("len = %d, want %d", len(got), len(want))
		}
		for i, v := range got {
			if v != want[i] {
				t.Errorf("index %d = %q, want %q", i, v, want[i])
			}
		}
	})

	t.Run("mixed types filters non-strings", func(t *testing.T) {
		got := GetStringSlice(m, "mixed")
		want := []string{"str", "another"}
		if len(got) != len(want) {
			t.Fatalf("len = %d, want %d", len(got), len(want))
		}
	})

	t.Run("missing key", func(t *testing.T) {
		got := GetStringSlice(m, "missing")
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("wrong type", func(t *testing.T) {
		got := GetStringSlice(m, "string")
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}

func TestGetSlice(t *testing.T) {
	m := map[string]interface{}{
		"args":   []interface{}{"arg1", 42, true},
		"string": "not a slice",
	}

	t.Run("valid slice", func(t *testing.T) {
		got := GetSlice(m, "args")
		if len(got) != 3 {
			t.Errorf("len = %d, want 3", len(got))
		}
	})

	t.Run("missing key", func(t *testing.T) {
		got := GetSlice(m, "missing")
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}

func TestGetMap(t *testing.T) {
	m := map[string]interface{}{
		"nested": map[string]interface{}{
			"key": "value",
		},
		"string": "not a map",
	}

	t.Run("valid map", func(t *testing.T) {
		got := GetMap(m, "nested")
		if got == nil {
			t.Fatal("expected map, got nil")
		}
		if got["key"] != "value" {
			t.Errorf("nested[key] = %v, want value", got["key"])
		}
	})

	t.Run("missing key", func(t *testing.T) {
		got := GetMap(m, "missing")
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("wrong type", func(t *testing.T) {
		got := GetMap(m, "string")
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}

func TestDecompressBacktrace(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		got, err := DecompressBacktrace("")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("valid compressed backtrace", func(t *testing.T) {
		// Create a compressed backtrace
		lines := []string{
			"app/workers/my_worker.rb:10:in `perform'",
			"sidekiq/processor.rb:100:in `process'",
		}
		jsonData, _ := json.Marshal(lines)

		var buf bytes.Buffer
		w := zlib.NewWriter(&buf)
		_, _ = w.Write(jsonData)
		_ = w.Close()

		encoded := base64.StdEncoding.EncodeToString(buf.Bytes())

		got, err := DecompressBacktrace(encoded)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
		if got[0] != lines[0] {
			t.Errorf("line 0 = %q, want %q", got[0], lines[0])
		}
	})

	t.Run("invalid base64", func(t *testing.T) {
		_, err := DecompressBacktrace("not-valid-base64!!!")
		if err == nil {
			t.Error("expected error for invalid base64")
		}
	})

	t.Run("invalid zlib", func(t *testing.T) {
		encoded := base64.StdEncoding.EncodeToString([]byte("not zlib data"))
		_, err := DecompressBacktrace(encoded)
		if err == nil {
			t.Error("expected error for invalid zlib")
		}
	})
}

func TestMarshal(t *testing.T) {
	t.Run("valid struct", func(t *testing.T) {
		data := map[string]interface{}{
			"key":   "value",
			"count": 42,
		}
		got, err := Marshal(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == "" {
			t.Error("expected non-empty string")
		}
		// Verify it's valid JSON
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(got), &parsed); err != nil {
			t.Errorf("result is not valid JSON: %v", err)
		}
	})

	t.Run("nil value", func(t *testing.T) {
		got, err := Marshal(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "null" {
			t.Errorf("got %q, want null", got)
		}
	})
}
