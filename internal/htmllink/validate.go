package htmllink

import (
	"net/url"
	"strings"
)

type ValidationResult struct {
	Valid  bool
	URL    string
	Issues []string
}

func ValidateURL(rawURL string) ValidationResult {
	res := ValidationResult{URL: rawURL, Valid: true}
	if strings.TrimSpace(rawURL) == "" {
		res.Valid = false
		res.Issues = append(res.Issues, "empty URL")
		return res
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		res.Valid = false
		res.Issues = append(res.Issues, "parse error: "+err.Error())
		return res
	}
	if strings.Contains(rawURL, " ") {
		res.Issues = append(res.Issues, "contains spaces")
	}
	if u.Scheme != "" && u.Host == "" && u.Scheme != "mailto" && u.Scheme != "tel" && u.Scheme != "data" && u.Scheme != "javascript" {
		res.Issues = append(res.Issues, "scheme without host")
	}
	if strings.HasSuffix(u.Host, ".") {
		res.Issues = append(res.Issues, "trailing dot in host")
	}
	if len(res.Issues) > 0 {
		res.Valid = false
	}
	return res
}

func ValidateLinks(links []Link) []ValidationResult {
	results := make([]ValidationResult, len(links))
	for i, l := range links {
		target := l.Resolved
		if target == "" {
			target = l.Href
		}
		results[i] = ValidateURL(target)
	}
	return results
}

func NormalizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	if (u.Scheme == "http" && u.Port() == "80") || (u.Scheme == "https" && u.Port() == "443") {
		u.Host = u.Hostname()
	}
	if u.Path == "/" && u.RawQuery == "" && u.Fragment == "" {
		u.Path = ""
	}
	return u.String()
}

func ExtractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func IsSecure(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return strings.ToLower(u.Scheme) == "https"
}

func CountSecureLinks(links []Link) int {
	n := 0
	for _, l := range links {
		target := l.Resolved
		if target == "" {
			target = l.Href
		}
		if IsSecure(target) {
			n++
		}
	}
	return n
}
