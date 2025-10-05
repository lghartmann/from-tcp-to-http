package headers

import (
	"bytes"
	"fmt"
	"strings"
)

func isToken(str []byte) bool {
	for _, ch := range str {
		found := false
		if ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' {
			found = true
		}
		switch ch {
		case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
			found = true
		}
		if !found {
			return false
		}

	}

	return true
}

func parseHeader(fieldLine []byte) (string, string, error) {
	parts := bytes.SplitN(fieldLine, []byte(":"), 2)
	if len(parts) != 2 {
		return "", "", ErrorMalformedFieldLine
	}

	name := parts[0]
	value := bytes.TrimSpace(parts[1])

	if bytes.HasSuffix(name, []byte(" ")) {
		return "", "", ErrorMalformedFieldName
	}

	return string(name), string(value), nil
}

type Headers struct {
	headers map[string]string
}

var CRLF = []byte("\r\n") // Registered Nurse by Prime

var ErrorMalformedFieldLine error = fmt.Errorf("malformed field line")
var ErrorMalformedFieldName error = fmt.Errorf("malformed field name")

func NewHeaders() *Headers {
	return &Headers{
		headers: map[string]string{},
	}
}

func (h *Headers) Get(name string) (string, bool) {
	str, ok := h.headers[strings.ToLower(name)]
	return str, ok
}

func (h *Headers) Set(name, value string) {
	mappedName := strings.ToLower(name)

	if v, ok := h.headers[mappedName]; ok {
		h.headers[mappedName] = fmt.Sprintf("%s,%s", v, value)
		return
	}

	h.headers[mappedName] = value
}

func (h *Headers) Replace(name, value string) {
	mappedName := strings.ToLower(name)

	h.headers[mappedName] = value
}

func (h *Headers) ForEach(cb func(n, v string)) {
	for name, value := range h.headers {
		cb(name, value)
	}
}

func (h *Headers) Parse(data []byte) (int, bool, error) {
	read := 0
	done := false
	for {
		idx := bytes.Index(data[read:], CRLF)
		if idx == -1 {
			break
		}

		// EMPTY HEADER
		if idx == 0 {
			done = true
			read += len(CRLF)
			break
		}

		name, value, err := parseHeader(data[read : read+idx])
		if err != nil {
			return 0, false, err
		}

		if !isToken([]byte(name)) {
			return 0, false, ErrorMalformedFieldName
		}

		read += idx + len(CRLF)
		h.Set(name, value)
	}

	return read, done, nil
}
