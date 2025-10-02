package headers

import (
	"bytes"
	"fmt"
)

type Headers map[string]string

var CRLF = []byte("\r\n") // Registered Nurse by Prime

var ErrorMalformedFieldLine error = fmt.Errorf("malformed field line")
var ErrorMalformedFieldName error = fmt.Errorf("malformed field name")

func NewHeaders() Headers {
	return Headers{}
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

func (h Headers) Parse(data []byte) (int, bool, error) {
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

		read += idx + len(CRLF)
		h[name] = value
	}

	return read, done, nil
}
