package tarantool

import (
	"crypto/sha1"
	"fmt"
	"tarantool/msgpack"
)

type auth struct {
	User     string
	Scramble [sha1.Size]byte
}

func (a auth) Bytes() ([]byte, error) {
	l := len(a.User)
	if l > 31 {
		return nil, fmt.Errorf("username is too long")
	}

	o := make([]byte, 4+l+15+sha1.Size)
	// map with length 2
	o[0] = msgpack.Map15LengthMask | 0x02
	// key iproto_username
	o[1] = msgpack.Uint8
	o[2] = msgpack.KeyUserName
	// username
	o[3] = msgpack.String31LengthMask | byte(l)
	for i := 0; i < l; i++ {
		o[4+i] = a.User[i]
	}
	j := 4 + l
	// key iproto_tuple
	o[j+0] = msgpack.Uint8
	o[j+1] = msgpack.KeyTuple
	// array with length 2
	o[j+2] = msgpack.Array15LengthMask | 0x02
	//"chap-sha1"
	o[j+3] = msgpack.String31LengthMask | 0x09
	o[j+4] = 0x63
	o[j+5] = 0x68
	o[j+6] = 0x61
	o[j+7] = 0x70
	o[j+8] = 0x2d
	o[j+9] = 0x73
	o[j+10] = 0x68
	o[j+11] = 0x61
	o[j+12] = 0x31
	// scramble type and length
	o[j+13] = msgpack.Bin8
	o[j+14] = sha1.Size
	// scramble
	for i := 0; i < sha1.Size; i++ {
		o[j+15+i] = a.Scramble[i]
	}

	return o, nil
}
