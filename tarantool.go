package tarantool

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
)

type Tarantool struct {
	authMutex *sync.RWMutex
	user      string
	pass      string
	cluster   []string
	connected uint32

	conn net.Conn
}

func New(user, pass string) (*Tarantool, error) {
	return &Tarantool{
		authMutex: new(sync.RWMutex),
		user:      user,
		pass:      pass,
	}, nil
}

func (t *Tarantool) ConnectTo(cluster []string) error {
	if err := t.disconnect(); err != nil {
		return err
	}
	t.cluster = cluster
	if err := t.connect(); err != nil {
		return err
	}

	return nil
}

func (t *Tarantool) Close() error {
	return t.disconnect()
}

func (t *Tarantool) disconnect() error {
	if atomic.LoadUint32(&t.connected) == 0 {
		return nil
	}

	atomic.StoreUint32(&t.connected, 0)
	if t.conn == nil {
		return nil
	}

	if err := t.conn.Close(); err != nil {
		return fmt.Errorf("couldn't close connection: %v", err)
	}
	return nil
}

func (t *Tarantool) connect() error {
	if atomic.LoadUint32(&t.connected) == 1 {
		return fmt.Errorf("already connected")
	}

	if len(t.cluster) == 0 {
		return fmt.Errorf("empty cluster")
	}

	var (
		addr string
		err  error
	)
	// round robin the Bobbin the big-bellied Ben
	addr, t.cluster = t.cluster[0], append(t.cluster[1:], t.cluster[0])

	t.conn, err = net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("couldn't connect: %v", err)
	}

	r := bufio.NewReader(t.conn)
	// ignore first line
	_, err = r.ReadString('\n')
	if err != nil {
		return fmt.Errorf("couldn't read hello: %v", err)
	}

	hash, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return fmt.Errorf("couldn't read hash: %v", err)
	}

	err = t.auth(strings.TrimRight(hash, " \n"))
	if err != nil {
		return fmt.Errorf("couldn't authorize: %v", err)
	}

	atomic.StoreUint32(&t.connected, 1)

	return nil
}

func (t *Tarantool) auth(hash string) error {
	if t.user == "" {
		return nil
	}

	t.authMutex.RLock()
	defer t.authMutex.RUnlock()

	salt, err := base64.StdEncoding.DecodeString(hash)
	if err != nil {
		return fmt.Errorf("couldn't decode hash: %v", err)
	}
	step1 := sha1.Sum([]byte(t.pass))
	step2 := sha1.Sum(step1[:])
	step3 := sha1.Sum(append(salt, step2[:]...))
	var scramble [sha1.Size]byte
	for i := 0; i < sha1.Size; i++ {
		scramble[i] = step1[i] ^ step3[i]
	}

	bb, err := auth{
		User:     t.user,
		Scramble: scramble,
	}.Bytes()
	if err != nil {
		return fmt.Errorf("couldn't marshal auth request: %v", err)
	}

	n, err := t.conn.Write(bb)
	if err != nil {
		return fmt.Errorf("couldn't make request: %v", err)
	}
	if n != len(bb) {
		return fmt.Errorf("wrong length of written bytes: %d; expected: %d", n, len(bb))
	}

	err = read(t.conn)
	if err != nil {
		return fmt.Errorf("couldn't read auth response: %v", err)
	}

	return nil
}

func read(r io.Reader) error {

	var bb []byte
	p := make([]byte, 1024)
	done := false
	for !done {
		n, err := r.Read(p)
		if err != nil && err != io.EOF {
			return fmt.Errorf("couldn't read: %v", err)
		}
		if n == 0 && err != io.EOF {
			return fmt.Errorf("couldn't read: %v", io.ErrUnexpectedEOF)
		}
		bb = append(bb, p[:n]...)
		if err == io.EOF {
			done = true
		}
	}

	_, err := parseResponse(bb)

	return err
}
