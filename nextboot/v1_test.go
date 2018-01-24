package nextboot

import (
	"testing"

	"github.com/kr/pretty"
)

func TestConfig_ParseCmdline(t *testing.T) {
	type input struct {
		cmdline string
	}
	type expected struct {
		Kargs map[string]string
	}
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		// All tests are expected to succeed.
		{
			name:     "single value",
			input:    "key=val",
			expected: map[string]string{"key": "val"},
		},
		{
			name:     "single value with extra whitespace",
			input:    "key=val  \n",
			expected: map[string]string{"key": "val"},
		},
		{
			name:     "multi-value",
			input:    "key=val key2=val2",
			expected: map[string]string{"key": "val", "key2": "val2"},
		},
		{
			name:     "multi-value with multiple equal signs",
			input:    "key=val key2=val2=val3",
			expected: map[string]string{"key": "val", "key2": "val2=val3"},
		},
		{
			name:  "multi-value with URL with parameters",
			input: "key=val url=http://thing.com?a=b&c=d",
			expected: map[string]string{
				"key": "val",
				"url": "http://thing.com?a=b&c=d",
			},
		},
		{
			name:     "multi-value with special values in key",
			input:    "key=val ide-core.nodma=0.1",
			expected: map[string]string{"key": "val", "ide-core.nodma": "0.1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{}
			c.ParseCmdline(tt.input)

			if diff := pretty.Diff(c.Kargs, tt.expected); len(diff) != 0 {
				t.Errorf("Config.ParseCmdline() got = %v, want %v", c.Kargs, tt.expected)
			}
		})
	}
}
