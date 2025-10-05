package response

import (
	"bytes"
	"fmt"
	"io"

	"github.com/lghartmann/from-tcp-to-http/internal/headers"
)

type Response struct {
}

type StatusCode int

const (
	StatusOK                  StatusCode = 200
	StatusBadRequest          StatusCode = 400
	StatusInternalServerError StatusCode = 500
)

func GetDefaultHeaders(contentLen int) *headers.Headers {
	h := headers.NewHeaders()

	h.Set("content-length", fmt.Sprintf("%d", contentLen))
	h.Set("connection", "close")
	h.Set("content-type", "text/plain")

	return h
}

type Writer struct {
	writer io.Writer
}

func NewWriter(writer io.Writer) *Writer {
	return &Writer{writer: writer}
}

func (w *Writer) WriteStatusLine(statusCode StatusCode) error {
	switch statusCode {
	case StatusOK:
		_, err := w.writer.Write([]byte("HTTP/1.1 200 OK\r\n"))
		return err
	case StatusBadRequest:
		_, err := w.writer.Write([]byte("HTTP/1.1 400 Bad Request\r\n"))
		return err
	case StatusInternalServerError:
		_, err := w.writer.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n"))
		return err
	default:
		return fmt.Errorf("unrecognized status code")
	}
}

func (w *Writer) WriteHeaders(h *headers.Headers) error {
	var b bytes.Buffer
	h.ForEach(func(n, v string) {
		b.WriteString(fmt.Sprintf("%s: %s\r\n", n, v))
	})
	b.WriteString("\r\n")

	_, err := w.writer.Write(b.Bytes())
	return err
}

func (w *Writer) WriteBody(p []byte) (int, error) {
	n, err := w.writer.Write(p)

	return n, err
}
