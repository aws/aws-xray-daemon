package test

import (
	"sync"
	"sync/atomic"
	"time"
)

type timer struct {
	start    time.Time
	duration time.Duration
	repeat   bool
	fired    bool
	c        chan time.Time
}

// MockTimerClient contains mock timer client.
type MockTimerClient struct {
	sync.RWMutex

	current time.Time
	timers  []*timer

	afterCalled uint64
	tickCalled  uint64
}

func (m *MockTimerClient) newTimer(d time.Duration, repeat bool) *timer {
	m.RLock()
	t := &timer{
		start:    m.current,
		duration: d,
		repeat:   repeat,
		fired:    false,
		c:        make(chan time.Time, 1),
	}
	m.RUnlock()

	m.Lock()
	defer m.Unlock()
	m.timers = append(m.timers, t)

	return t
}

// After is mock of time.After().
func (m *MockTimerClient) After(d time.Duration) <-chan time.Time {
	atomic.AddUint64(&m.afterCalled, 1)

	return m.newTimer(d, false).c
}

// Tick is mock of time.Tick().
func (m *MockTimerClient) Tick(d time.Duration) <-chan time.Time {
	atomic.AddUint64(&m.tickCalled, 1)

	return m.newTimer(d, true).c
}

// Advance simulates time passing and signal timers / tickers accordingly
func (m *MockTimerClient) Advance(d time.Duration) {
	m.Lock()
	m.current = m.current.Add(d)
	m.Unlock()

	m.RLock()
	defer m.RUnlock()

	curr := m.current
	for _, t := range m.timers {
		if t.repeat {
			// for Tickers, calculate how many ticks has passed and signal accordingly
			for i := int64(0); i < int64(curr.Sub(t.start)/t.duration); i++ {
				t.c <- t.start.Add(t.duration * time.Duration(i+1))
				t.start = t.start.Add(t.duration)
			}
		} else {
			// for Afters (one-off), signal once
			if !t.fired && (curr.Sub(t.start) >= t.duration) {
				t.c <- t.start.Add(t.duration)
				t.fired = true
			}
		}
	}
}

// AfterCalledTimes calculates number of times after is called.
func (m *MockTimerClient) AfterCalledTimes() uint64 {
	return atomic.LoadUint64(&m.afterCalled)
}

// TickCalledTimes calculates number of times tick is called.
func (m *MockTimerClient) TickCalledTimes() uint64 {
	return atomic.LoadUint64(&m.tickCalled)
}
