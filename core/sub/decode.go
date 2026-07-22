package sub

import (
	"encoding/base64"
	"strings"
)

func tryBase64Decode(str string) (string, bool) {
	str = strings.TrimSpace(str)
	if str == "" {
		return "", false
	}

	i := len(str) % 4
	switch i {
	case 1:
		str = str[:len(str)-1]
	case 2:
		str += "=="
	case 3:
		str += "="
	}
	var data []byte
	var err error
	if strings.Contains(str, "-") || strings.Contains(str, "_") {
		data, err = base64.URLEncoding.DecodeString(str)
	} else {
		data, err = base64.StdEncoding.DecodeString(str)
	}
	if err != nil {
		return "", false
	}
	return string(data), true
}

func base64Decode(str string) string {
	data, _ := tryBase64Decode(str)
	return data

}

func normalizeSubText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return text
}

func splitSubLines(text string) []string {
	text = normalizeSubText(text)
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{}
	}

	list := strings.Split(text, "\n")
	result := make([]string, 0, len(list))
	for _, item := range list {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		result = append(result, item)
	}
	return result

}

func tryYAMLSubLinks(text string) ([]string, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, false
	}

	links, err := clashProxiesToLinks([]byte(trimmed))
	if err != nil {
		return nil, false
	}
	return links, true
}

// Sub2links parses subscription text into share links.
func Sub2links(subtext string) []string {
	decoded, ok := tryBase64Decode(subtext)
	if ok {
		if links, isYAML := tryYAMLSubLinks(decoded); isYAML && len(links) > 0 {
			return links
		}
	}

	if links, isYAML := tryYAMLSubLinks(subtext); isYAML && len(links) > 0 {
		return links
	}

	if ok {
		return splitSubLines(decoded)
	}
	return splitSubLines(subtext)
}
