package memcached

import (
	"io"
	"fmt"
)

type MemcachedResponse interface {
	WriteResponse(io.Writer)
}

type ItemResponse struct {
	Item *Item
}

func (r *ItemResponse) WriteResponse(writer io.Writer) (n int) {
	fmt.Fprintf(writer, StatusValue, r.Item.Key, r.Item.Flags, len(r.Item.Value))
	n = writer.Write(r.Item.Value)
	n = n + writer.Write(crlf)
}

type BulkResponse struct {
	Responses []MemcachedResponse
}


func (r *BulkResponse) WriteResponse(writer io.Writer)  (n int) {
	for _, response := range r.Responses {
		if response != nil {
			n = n + response.WriteResponse(writer)
		}
	}
}

type ClientErrorResponse struct {
	Reason string
}

func (r *ClientErrorResponse) WriteResponse(writer io.Writer)  (n int) {
	fmt.Fprintf(writer, StatusClientError, r.Reason)
	n = 0
	return n
}
