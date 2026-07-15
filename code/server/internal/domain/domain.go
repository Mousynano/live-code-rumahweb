package domain

import (
	"net/url"
	"regexp"
	"strings"
)

const MaxBatch = 100

var tldPattern = regexp.MustCompile(`^[a-z]{2,63}$`)

type Parsed struct {
	Domains    []string `json:"domains"`
	Detected   int      `json:"detected"`
	Duplicates int      `json:"duplicatesRemoved"`
	Invalid    []string `json:"invalid"`
}

func Parse(values []string) Parsed {
	var input []string
	for _, value := range values {
		input = append(input, strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t' || r == ' ' })...)
	}
	out := Parsed{Detected: len(input)}
	seen := make(map[string]bool)
	for _, raw := range input {
		d, ok := Normalize(raw)
		if !ok {
			out.Invalid = append(out.Invalid, raw)
			continue
		}
		if seen[d] {
			out.Duplicates++
			continue
		}
		seen[d] = true
		out.Domains = append(out.Domains, d)
	}
	return out
}

func Normalize(raw string) (string, bool) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return "", false
	}
	if !strings.Contains(raw, "://") {
		raw = "//" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	host := u.Hostname()
	host = strings.TrimSuffix(host, ".")
	if host == "" || len(host) > 253 || strings.ContainsAny(host, "\x00_/\\") {
		return "", false
	}
	labels := strings.Split(host, ".")
	if len(labels) < 2 || !tldPattern.MatchString(labels[len(labels)-1]) {
		return "", false
	}
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", false
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				return "", false
			}
		}
	}
	return host, true
}
