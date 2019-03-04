package transport

import (
	"bytes"
	"context"
	"fmt"
	"github.com/vmihailenco/msgpack"
	"io"
)

type Response struct {
	IsError   bool
	ErrorCode uint16
	Error     string
	SchemaID  uint32
	Header    Header
	Body      interface{}
}

func Read(ctx context.Context, rd io.Reader) (Response, error) {
	var r Response

	p, err := read(ctx, rd)
	if err != nil {
		return r, fmt.Errorf("couldn't read: %v", err)
	}

	buf := bytes.NewBuffer(p)
	d := msgpack.NewDecoder(buf)

	var l uint32
	err = d.DecodeMulti(&l, &r.Header, &r.Body)
	if err != nil {
		return r, fmt.Errorf("couldn't decode response: %v", err)
	}
	if int(l) != len(p)-5 {
		return r, fmt.Errorf("wrong response length %d; expected %d", len(p)-5, l)
	}

	// parse header
	if _, ok := r.Header[KeyCode]; !ok {
		return r, fmt.Errorf("couldn't get response code")
	}
	if (r.Header[KeyCode] & CodeErrorMask) != 0 {
		r.IsError = true
		r.ErrorCode = uint16(r.Header[KeyCode] & ErrorCodeMask)
	}
	if v, ok := r.Header[KeySchema]; ok {
		r.SchemaID = v
	}

	// body
	if r.IsError {
		if _, ok := r.Body.(map[int8]string); !ok {
			return r, fmt.Errorf("couldn't cast type of error")
		}
		r.Error = r.Body.(map[int8]string)[int8(KeyError)]
		r.Body = map[uint8]interface{}{}
		return r, nil
	}

	return r, nil
}

func read(ctx context.Context, r io.Reader) ([]byte, error) {
	var bb []byte
	p := make([]byte, 1024)
	done := false
	for !done {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		n, err := r.Read(p)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n == 0 && err != io.EOF {
			return nil, io.ErrUnexpectedEOF
		}
		bb = append(bb, p[:n]...)
		if n < len(p) || err == io.EOF {
			done = true
		}
	}
	return bb, nil
}
