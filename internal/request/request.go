package request

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/lghartmann/from-tcp-to-http/internal/headers"
)

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

func (r *RequestLine) ValidHTTP() bool {
	split := strings.Split(r.HttpVersion, "/")
	if len(split) < 2 || split[0] != "HTTP" {
		return false
	}

	return split[1] == "1.1"
}

type parserState string

type Request struct {
	RequestLine RequestLine
	state       parserState
	Headers     *headers.Headers
	// Body        []byte
}

func NewRequest() *Request {
	return &Request{
		state:   StateInit,
		Headers: headers.NewHeaders(),
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
			return nil, err
		}

		buffLen += n

		readN, err := request.parse(buffer[:buffLen+n])
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
		HttpVersion:   string(parts[2]),
	}

	if !requestLine.ValidHTTP() {
		return nil, 0, ErrUnsupportedHTTPVersion
	}

	return requestLine, read, nil
}
