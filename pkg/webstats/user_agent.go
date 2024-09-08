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

	// Mozilla/5.0 (compatible; CensysInspect/1.1; +https://about.censys.io/)
	if strings.Contains(ua, "+http") {
		return true
	}

	if strings.Contains(ua, "Headless") {
		return true
	}

	// Mozilla/5.0 (Linux; Android 5.0) AppleWebKit/537.36 (KHTML, like Gecko) Mobile Safari/537.36 (compatible; Bytespider; spider-feedback@bytedance.com)
	if strings.Contains(ua, "Bytespider") {
		return true
	}

	if ua == "Mozilla/5.0 (X11; Fedora; Linux x86_64; rv:94.0) Gecko/20100101 Firefox/95.0" {
		return true
	}

	return false
}
