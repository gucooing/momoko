package utils

import (
	"fmt"
	"strings"
)

func UniqueNonEmpty(ids []string, exclude string) []string {
	seen := make(map[string]struct{}, len(ids))
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || id == exclude {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func PrivacyString(str string) string {
	if len(str) <= 10 {
		return "******"
	}
	return fmt.Sprintf("%s*******%s", str[:6], str[len(str)-3:])
}
