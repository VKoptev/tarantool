package transport

type Header map[uint8]uint32
type Error map[uint8]string

type Requester interface {
	Header() Header
	Body() interface{}
}
