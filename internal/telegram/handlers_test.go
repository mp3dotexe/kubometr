package telegram

import (
	"slices"
	"testing"
)

type splitMessageCase struct {
	name string
	text string
	limit int
	expected []string
}

func TestSplitMessage(t *testing.T) {
	cases := []splitMessageCase{
		{
			name: "empty text",
			text: "",
			limit: 0,
			expected: []string{""},
		},
		{
			name:	"text > limit",
			text:	"привет",
			limit:	5,
			expected: []string{"приве","т"},
		},
		{
			name:	"text == limit",
			text:	"привет",
			limit: 6,
			expected: []string{"привет"},
		},
	}

	for _, c := range cases {
		result := splitMessage(c.text, c.limit)
		if !slices.Equal(result, c.expected) {
			t.Errorf("%s: expected %v, got %v", c.name, c.expected, result)
		}
	}
}