package cmd

import (
	"strings"

	"github.com/abiosoft/ishell"
)

// confirm asks the user before performing a destructive operation. prompt is the
// question text. Only "y"/"yes" (case-insensitive) returns true; anything else
// (including bare Enter) returns false. This defaults to "do not delete" so a
// stray keystroke will not wipe data.
func confirm(c *ishell.Context, prompt string) bool {
	c.Print(prompt + " [y/N]: ")
	answer := strings.ToLower(strings.TrimSpace(c.ReadLine()))
	return answer == "y" || answer == "yes"
}

func splitListArgs(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
