package tarantool

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Config struct {
	User    string
	Pass    string
	Timeout time.Duration
}

type Tarantool struct {
	cfg     Config
	cluster []string
	pool    *sync.Map

	mu      *sync.RWMutex
	freeIDs []uint64
	lastID  uint64
}

func New(cfg Config) (*Tarantool, error) {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &Tarantool{
		cfg:  cfg,
		pool: new(sync.Map),
		mu:   new(sync.RWMutex),
	}, nil
}

func (t *Tarantool) ChangeHosts(ctx context.Context, hosts []string) error {
	if err := t.Close(); err != nil {
		return err
	}
	t.cluster = hosts
	return nil
}

func (t *Tarantool) Close() error {
	var err error
	t.pool.Range(func(key, value interface{}) bool {
		err = value.(*Session).Close()
		return err == nil
	})

	return err
}

func (t *Tarantool) Session(ctx context.Context) (*Session, error) {
	if len(t.cluster) == 0 {
		return nil, fmt.Errorf("empty cluster")
	}

	var addr string
	t.mu.Lock()
	// round Robin the Bobbin the big-bellied Ben
	addr, t.cluster = t.cluster[0], append(t.cluster[1:], t.cluster[0])
	t.mu.Unlock()

	c := session(sessionConfig{
		addr:    addr,
		user:    t.cfg.User,
		pass:    t.cfg.Pass,
		timeout: t.cfg.Timeout,
	})
	err := c.connect(ctx)
	if err != nil {
		return nil, err
	}

	t.mu.Lock()
	var id uint64
	if len(t.freeIDs) > 0 {
		id, t.freeIDs = t.freeIDs[0], t.freeIDs[1:]
	} else {
		id = t.lastID
		t.lastID++
	}
	t.mu.Unlock()

	t.pool.Store(id, c)
	go t.waitConnectionClose(ctx, id, c)

	return c, nil
}

func (t *Tarantool) waitConnectionClose(ctx context.Context, id uint64, c *Session) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			t.pool.Delete(id)
			t.freeIDs = append(t.freeIDs, id)
			return
		}
	}
}
