package transport

type Header map[uint8]uint32

type Requester interface {
	Header() Header
	Body() interface{}
}
