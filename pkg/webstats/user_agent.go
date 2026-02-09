package webstats

import "strings"

func IsBot(ua string) bool {
	if ua == "" {
		return true
	}

	if !strings.HasPrefix(ua, "Mozilla/") && !strings.HasPrefix(ua, "Opera/9") {
		return true
	}

	if strings.Contains(ua, "bot") {
		return true
	}

	if strings.Contains(ua, "Bot") {
		return true
	}

	if strings.Contains(ua, "Bytespider") {
		return true
	}

	// Mozilla/5.0 (compatible; CensysInspect/1.1; +https://about.censys.io/)
	if strings.Contains(ua, "+http") {
		return true
	}

	if strings.Contains(ua, "Headless") {
		return true
	}

	return false
}
