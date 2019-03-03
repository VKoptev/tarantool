package tarantool

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"tarantool/transport"
)

type auth struct {
	username string
	scramble []byte
}

func (a auth) Header() transport.Header {
	return map[uint8]uint32{
		transport.KeyCode: transport.RequestAuth,
		transport.KeySync: 1,
	}
}

func (a auth) Body() interface{} {
	return map[uint8]interface{}{
		transport.KeyUserName: a.username,
		transport.KeyTuple:    [][]byte{[]byte("chap-sha1"), a.scramble},
	}
}

func scramble(hash, pass string) (scramble []byte, err error) {
	salt, err := base64.StdEncoding.DecodeString(hash)
	if err != nil {
		return scramble, fmt.Errorf("couldn't decode hash: %v", err)
	}
	step1 := sha1.Sum([]byte(pass))
	step2 := sha1.Sum(step1[0:])

	h := sha1.New()
	if n, err := h.Write(salt[0:sha1.Size]); err != nil || n != sha1.Size {
		return scramble, fmt.Errorf("couldn't write to hash: %v; written %d bytes", err, n)
	}
	if n, err := h.Write(step2[0:]); err != nil || n != sha1.Size {
		return scramble, fmt.Errorf("couldn't write to hash: %v; written %d bytes", err, n)
	}
	step3 := h.Sum(nil)

	scramble = make([]byte, sha1.Size)
	for i := 0; i < sha1.Size; i++ {
		scramble[i] = step1[i] ^ step3[i]
	}

	return scramble, nil
}
