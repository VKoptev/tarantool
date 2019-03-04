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
	"time"
)

type Session struct {
	cfg       sessionConfig
	connected uint32
	conn      net.Conn
	mu        *sync.RWMutex
	done      chan struct{}
}

type sessionConfig struct {
	addr    string
	user    string
	pass    string
	timeout time.Duration
}

func session(cfg sessionConfig) *Session {
	if cfg.timeout <= 0 {
		cfg.timeout = time.Second * 30
	}
	return &Session{
		cfg:       cfg,
		connected: 0,
		conn:      nil,
		mu:        new(sync.RWMutex),
		done:      make(chan struct{}, 1),
	}
}

func (c *Session) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if atomic.LoadUint32(&c.connected) == 0 {
		return nil
	}

	atomic.StoreUint32(&c.connected, 0)
	if c.conn == nil {
		return nil
	}

	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("couldn't close connection: %v", err)
	}
	c.conn = nil
	close(c.done)

	return nil
}

func (c *Session) connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if atomic.LoadUint32(&c.connected) == 1 {
		return nil
	}

	var err error

	d := &net.Dialer{
		Timeout: c.cfg.timeout,
	}

	c.conn, err = d.DialContext(ctx, "tcp", c.cfg.addr)
	if err != nil {
		return fmt.Errorf("couldn't connect: %v", err)
	}

	if c.cfg.user != "" {
		// not connected until auth
		err = c.auth(ctx)
		if err != nil {
			return fmt.Errorf("couldn't authorize: %v", err)
		}
	}

	atomic.StoreUint32(&c.connected, 1)

	return nil
}

func (c *Session) auth(ctx context.Context) error {
	rd := bufio.NewReader(c.conn)

	// ignore first line
	_, err := rd.ReadString('\n')
	if err != nil {
		return fmt.Errorf("couldn't read hello: %v", err)
	}

	hash, err := rd.ReadString('\n')
	if err != nil && err != io.EOF {
		return fmt.Errorf("couldn't read hash: %v", err)
	}

	scramble, err := scramble(strings.TrimRight(hash, " \n"), c.cfg.pass)
	if err != nil {
		return err
	}

	// do not request because not connected until auth and locked yet
	err = transport.Write(ctx, c.conn, auth{username: c.cfg.user, scramble: scramble})
	if err != nil {
		return fmt.Errorf("couldn't make auth request: %v", err)
	}

	r, err := transport.Read(ctx, c.conn)
	if err != nil {
		return fmt.Errorf("couldn't read auth response: %v", err)
	}

	if r.IsError {
		return fmt.Errorf("auth error #%d: %s", r.ErrorCode, r.Error)
	}

	return nil
}

func (c *Session) request(ctx context.Context, req transport.Requester) (r transport.Response, err error) {
	err = c.waitConnected(ctx)
	if err != nil {
		return r, fmt.Errorf("couldn't wait connection: %v", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	err = transport.Write(ctx, c.conn, req)
	if err != nil {
		return r, fmt.Errorf("couldn't make request: %v", err)
	}

	r, err = transport.Read(ctx, c.conn)
	if err != nil {
		return r, fmt.Errorf("couldn't read response: %v", err)
	}

	return r, nil
}

func (c *Session) waitConnected(ctx context.Context) error {
	timer := time.NewTimer(c.cfg.timeout)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return fmt.Errorf("connection timeout")
		default:
			if atomic.LoadUint32(&c.connected) == 1 {
				return nil
			}
			time.Sleep(time.Millisecond * 100)
		}
	}
}
