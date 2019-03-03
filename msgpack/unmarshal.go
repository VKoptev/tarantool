package msgpack

import (
	"encoding/binary"
	"fmt"
)

func Unmarshal(data []byte) (interface{}, []byte, error) {
	return token(data)
}

func unmarshalMap(bb []byte) (map[interface{}]interface{}, []byte, error) {
	var l uint32
	switch {
	case (bb[0] & Map15LengthMask) == Map15LengthMask:
		l, bb = uint32(0xf&bb[0]), bb[1:]
	case bb[0] == Map16:
		l, bb = uint32(binary.BigEndian.Uint16(bb[1:3])), bb[3:]
	case bb[0] == Map32:
		l, bb = uint32(binary.BigEndian.Uint16(bb[1:5])), bb[5:]
	default:
		return nil, bb, fmt.Errorf("unexpected map type")
	}

	var (
		k, v interface{}
		err  error
	)
	res := make(map[interface{}]interface{})
	for ; l > 0; l-- {
		k, bb, err = token(bb)
		if err != nil {
			return nil, bb, fmt.Errorf("couldn't get key: %v", err)
		}
		v, bb, err = token(bb)
		if err != nil {
			return nil, bb, fmt.Errorf("couldn't get value: %v", err)
		}
		res[k] = v
	}

	return res, bb, nil
}

func token(bb []byte) (interface{}, []byte, error) {
	fmt.Printf("get token %+x\n", bb[:4])
	switch bb[0] {
	case Uint8:
		return uint8(bb[1]), bb[2:], nil
	case Uint16:
		return binary.BigEndian.Uint16(bb[1:3]), bb[3:], nil
	case Uint32:
		return binary.BigEndian.Uint32(bb[1:5]), bb[5:], nil
	case Uint64:
		return binary.BigEndian.Uint64(bb[1:9]), bb[9:], nil
	case Int8:
		return int8(bb[1]), bb[2:], nil
	case Int16:
		return int16(binary.BigEndian.Uint16(bb[1:3])), bb[3:], nil
	case Int32:
		return int32(binary.BigEndian.Uint32(bb[1:5])), bb[5:], nil
	case Int64:
		return int64(binary.BigEndian.Uint64(bb[1:9])), bb[9:], nil
	case Map16, Map32:
		return unmarshalMap(bb)
	}

	switch {
	case (bb[0] & Map15LengthMask) == Map15LengthMask:
		return unmarshalMap(bb)
	}

	// cf 00 00 00 00 00 00 00 00 05 ce

	return nil, bb, fmt.Errorf("unexpected type %x", bb[0])
}
