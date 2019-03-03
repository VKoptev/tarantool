package transport

import (
	"bytes"
	"context"
	"fmt"
	"github.com/vmihailenco/msgpack"
	"io"
)

func Write(ctx context.Context, wr io.Writer, r Requester) error {
	buf := bytes.NewBuffer([]byte{})
	e := msgpack.NewEncoder(buf)

	err := e.Encode(e.EncodeMulti(r.Header(), r.Body()))
	if err != nil {
		return fmt.Errorf("couldn't encode header and body: %v", err)
	}
	l := buf.Len()
	p := make([]byte, l+5)
	copy(p[5:l+5], buf.Bytes())

	buf.Reset()
	err = e.EncodeUint32(uint32(l))
	if err != nil {
		return fmt.Errorf("couldn't encode length: %v", err)
	}
	copy(p[0:5], buf.Bytes())

	n, err := wr.Write(p)
	if err != nil {
		return fmt.Errorf("couldn't do request: %v", err)
	}
	if n != l+5 {
		return fmt.Errorf("wrong length of written bytes: %d; expected: %d", n, l+5)
	}

	return nil
}
