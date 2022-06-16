package http

import "net/http"

type Header struct {
	http.Header
}

func newHeader() *Header {
	return &Header{
		Header: http.Header{},
	}
}

func (h *Header) View() http.Header {
	return h.Header
}
