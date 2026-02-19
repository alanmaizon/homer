package agents

import "strings"

func Critique(result string) string {
	if strings.TrimSpace(result) == "" {
		return result
	}
	return result + "\n\n[critic reviewed]"
}
