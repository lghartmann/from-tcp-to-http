package request

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/lghartmann/from-tcp-to-http/internal/headers"
)

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

func (r *RequestLine) ValidHTTP() bool {
	if strings.Contains(r.HttpVersion, "/") {
		split := strings.Split(r.HttpVersion, "/")
		if len(split) < 2 || split[0] != "HTTP" {
			return false
		}
		return split[1] == "1.1"
	}

	return r.HttpVersion == "1.1"
}

type parserState string

type Request struct {
	RequestLine RequestLine
	state       parserState
	Headers     *headers.Headers
	Body        string
}

func getIntHeader(headers *headers.Headers, name string, defaultValue int) int {
	valueStr, ok := headers.Get(name)
	if !ok {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

func NewRequest() *Request {
	return &Request{
		state:   StateInit,
		Headers: headers.NewHeaders(),
		Body:    "",
	}
}

func (r *Request) parse(data []byte) (int, error) {
	read := 0
outer:
	for {
		currentData := data[read:]
		switch r.state {
		case StateError:
			return 0, ErrRequestInErrorState
		case StateInit:
			rl, n, err := parseRequestLine(currentData)
			if err != nil {
				return 0, err
			}

			if n == 0 {
				break outer
			}

			r.RequestLine = *rl
			read += n
			r.state = StateHeaders
		case StateHeaders:
			n, done, err := r.Headers.Parse(currentData)
			if err != nil {
				return 0, err
			}

			if n == 0 {
				break outer
			}

			read += n

			if done {
				// If Content-Length is present and > 0 we need to parse a body next.
				cl := getIntHeader(r.Headers, "content-length", 0)
				if cl > 0 {
					r.state = StateBody
				} else {
					r.state = StateDone
				}
			}
		case StateBody:
			lenStr := getIntHeader(r.Headers, "content-length", 0)
			if lenStr == 0 {
				r.state = StateDone
				break outer
			}

			if len(currentData) == 0 {
				// no data available in this invocation
				break outer
			}

			remaining := min(lenStr-len(r.Body), len(currentData))
			r.Body += string(currentData[:remaining])
			read += remaining

			if len(r.Body) == lenStr {
				r.state = StateDone
			}
		case StateDone:
			break outer
		default:
			panic("somehow we have programmed poorly")
		}
	}
	return read, nil
}

func (r *Request) done() bool {
	return r.state == StateDone
}

func (r *Request) error() bool {
	return r.state == StateError
}

var ErrMalformedRequestLine error = fmt.Errorf("malformed request-line")
var ErrUnsupportedHTTPVersion error = fmt.Errorf("unsupported http version")
var ErrRequestInErrorState error = fmt.Errorf("request in error state")
var SEPARATOR []byte = []byte("\r\n")

const (
	StateInit    parserState = "init"
	StateHeaders parserState = "headers"
	StateBody    parserState = "body"
	StateDone    parserState = "done"
	StateError   parserState = "error"
)

func RequestFromReader(reader io.Reader) (*Request, error) {
	request := NewRequest()

	buffer := make([]byte, 1024)
	buffLen := 0

	for !request.done() && !request.error() {
		n, err := reader.Read(buffer[buffLen:])
		if err != nil {
			if err == io.EOF {
				// try to parse whatever we have in the buffer one last time
				if buffLen == 0 {
					return nil, io.EOF
				}
				readN, perr := request.parse(buffer[:buffLen])
				if perr != nil {
					return nil, perr
				}
				copy(buffer, buffer[readN:buffLen])
				buffLen -= readN

				if !request.done() {
					// reader closed early; request incomplete
					return nil, io.EOF
				}

				break
			}

			return nil, err
		}

		buffLen += n

		readN, err := request.parse(buffer[:buffLen])
		if err != nil {
			return nil, err
		}

		copy(buffer, buffer[readN:buffLen])
		buffLen -= readN
	}

	return request, nil
}

func parseRequestLine(line []byte) (*RequestLine, int, error) {
	idx := bytes.Index(line, SEPARATOR)
	if idx == -1 {
		return nil, 0, nil
	}

	startLine := line[:idx]
	read := idx + len(SEPARATOR)

	parts := bytes.Split(startLine, []byte(" "))
	if len(parts) != 3 {
		return nil, 0, ErrMalformedRequestLine
	}

	requestLine := &RequestLine{
		Method:        string(parts[0]),
		RequestTarget: string(parts[1]),
		// store the raw token for now; we'll normalize below
		HttpVersion: string(parts[2]),
	}

	if !requestLine.ValidHTTP() {
		return nil, 0, ErrUnsupportedHTTPVersion
	}
	// Normalize stored version to the numeric part (e.g. "1.1") when it
	// includes the "HTTP/" prefix so callers can compare the version easily.
	if strings.Contains(requestLine.HttpVersion, "/") {
		parts := strings.Split(requestLine.HttpVersion, "/")
		if len(parts) >= 2 {
			requestLine.HttpVersion = parts[1]
		}
	}
	return requestLine, read, nil
}
