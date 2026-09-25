package chat

import (
	"slices"
	"testing"
)

func TestSplitText(t *testing.T) {
	cases := []struct {
		name     string
		text     string
		limit    int
		expected []string
	}{
		{name: "empty text", text: "", limit: 0, expected: []string{""}},
		{name: "no limit", text: "привет", limit: 0, expected: []string{"привет"}},
		{name: "text > limit", text: "привет", limit: 5, expected: []string{"приве", "т"}},
		{name: "text == limit", text: "привет", limit: 6, expected: []string{"привет"}},
		{name: "several parts", text: "abcdefg", limit: 3, expected: []string{"abc", "def", "g"}},
	}

	for _, c := range cases {
		result := SplitText(c.text, c.limit)
		if !slices.Equal(result, c.expected) {
			t.Errorf("%s: expected %v, got %v", c.name, c.expected, result)
		}
	}
}
