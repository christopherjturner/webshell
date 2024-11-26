package main

import (
	"context"
	"os"
	"sync"
	"time"
)

type Timeout interface {
	Start()
	Ping()
}

type NoOpTimeout struct{}

func (n *NoOpTimeout) Start() {}

func (n *NoOpTimeout) Ping() {}

type InactivityTimeout struct {
	C          chan time.Time
	ticker     *time.Ticker
	shutdown   func()
	lastActive time.Time
	ttl        time.Duration
	ctx        context.Context
	once       sync.Once
}

func NewInactivityTimeout(ctx context.Context, ttl time.Duration) *InactivityTimeout {
	return &InactivityTimeout{
		C:          make(chan time.Time, 2),
		ticker:     time.NewTicker(10 * time.Second), // 2x the client's ping interval
		shutdown:   func() { os.Exit(0) },
		ttl:        ttl,
		lastActive: time.Now(),
		ctx:        ctx,
	}
}

func (ka *InactivityTimeout) Start() {

	ka.once.Do(func() {
		go func() {
			logger.Info("Keep Alive active")
			ka.lastActive = time.Now()
			for {
				select {
				case <-ka.ctx.Done():
					// Global shutdown listener
					return
				case <-ka.C:
					ka.lastActive = time.Now()
				case <-ka.ticker.C:
					if time.Since(ka.lastActive) >= ka.ttl {
						logger.Info("Stopping server due to inactivity")
						ka.shutdown()
					}
				}
			}
		}()
	})
}

func (ka *InactivityTimeout) Ping() {
	ka.C <- time.Now()
}
