package tarantool

import (
	"encoding/binary"
	"fmt"
	"tarantool/msgpack"
)

type response struct {
	IsError bool
	Header  map[byte]interface{}
}

func parseResponse(bb []byte) (r response, err error) {
	fmt.Printf("bb: %x\n", bb)
	if len(bb) == 0 {
		return r, fmt.Errorf("empty response")
	}
	var t byte
	t, bb = bb[0], bb[1:]
	if t != msgpack.Uint32 {
		return r, fmt.Errorf("wrong length type")
	}

	var l []byte
	l, bb = bb[:4], bb[4:]

	if binary.BigEndian.Uint32(l) != uint32(len(bb)) {
		return r, fmt.Errorf("wrong response length %d; expected %d", len(bb), binary.BigEndian.Uint32(l))
	}

	var res interface{}
	res, bb, err = msgpack.Unmarshal(bb)
	if err != nil {
		return r, fmt.Errorf("couldn't unmarshal header: %v", err)
	}
	if _, ok := res.(map[interface{}]interface{}); !ok {
		return r, fmt.Errorf("wrong header type")
	}
	r.Header = make(map[byte]interface{})
	for k, v := range res.(map[interface{}]interface{}) {
		if _, ok := k.(byte); !ok {
			return r, fmt.Errorf("wrong header key type")
		}
		r.Header[k.(byte)] = v
	}

	if len(bb) == 0 {
		return r, nil
	}

	return r, fmt.Errorf("header: %+v", r.Header)
}
