package tarantool

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"tarantool/transport"
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

func (t *Tarantool) ConnectTo(ctx context.Context, cluster []string) error {
	if err := t.disconnect(); err != nil {
		return err
	}
	t.cluster = cluster
	if err := t.connect(ctx); err != nil {
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
	t.conn = nil

	return nil
}

func (t *Tarantool) connect(ctx context.Context) error {
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

	d := &net.Dialer{}
	t.conn, err = d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("couldn't connect: %v", err)
	}

	rd := bufio.NewReader(t.conn)

	// ignore first line
	_, err = rd.ReadString('\n')
	if err != nil {
		return fmt.Errorf("couldn't read hello: %v", err)
	}

	hash, err := rd.ReadString('\n')
	if err != nil && err != io.EOF {
		return fmt.Errorf("couldn't read hash: %v", err)
	}

	if t.user != "" {
		err = t.auth(ctx, strings.TrimRight(hash, " \n"))
		if err != nil {
			return fmt.Errorf("couldn't authorize: %v", err)
		}
	}

	atomic.StoreUint32(&t.connected, 1)

	return nil
}

func (t *Tarantool) auth(ctx context.Context, hash string) error {
	t.authMutex.RLock()
	defer t.authMutex.RUnlock()

	scramble, err := scramble(hash, t.pass)
	if err != nil {
		return err
	}

	err = transport.Write(ctx, t.conn, auth{username: t.user, scramble: scramble})
	if err != nil {
		return fmt.Errorf("couldn't make auth request: %v", err)
	}

	r, err := transport.Read(ctx, t.conn)
	if err != nil {
		return fmt.Errorf("couldn't read auth response: %v", err)
	}

	return fmt.Errorf("response: %+v", r)
}
