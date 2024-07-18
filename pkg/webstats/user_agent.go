package webstats

import "strings"

func IsBot(ua string) bool {
	if strings.Contains(ua, "bot") {
		return true
	}

	if strings.Contains(ua, "Bot") {
		return true
	}

	if strings.Contains(ua, "Headless") {
		return true
	}

	return false
}
