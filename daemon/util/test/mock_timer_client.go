package test

import (
	"sync/atomic"
	"time"
)

// MockTimerClient is a mock for timer client.
type MockTimerClient struct {
	afterCalled       int32
	tickCalled        int32
	currentTime       int64
	killAll           bool
	killRoutinesCount int32
	doneChannel       []chan bool
	update            []chan bool
}

// IncrementDuration increments mock timer by d duration.
func (t *MockTimerClient) IncrementDuration(d time.Duration) {
	atomic.AddInt64(&t.currentTime, int64(d))
	for _, update := range t.update {
		update <- true
	}
	for _, done := range t.doneChannel {
		<-done
	}
}

// Dispose kills all mock timer clients.
func (t *MockTimerClient) Dispose() {
	t.killAll = true
	for _, update := range t.update {
		update <- true
	}
	for _, done := range t.doneChannel {
		<-done
	}
}

// TickRoutine is a routine for Timer.Tick().
func (t *MockTimerClient) TickRoutine(d int64, c chan time.Time, done chan bool, update <-chan bool, startOfTick int64) {
	for {
		<-update
		if t.killAll {
			break
		}
		currentDuration := atomic.LoadInt64(&t.currentTime)
		divisor := (currentDuration - startOfTick) / d
		if d > 0 && divisor >= 1 {
			var i int64
			for i = 0; i < divisor; i++ {
				t := time.Now()
				c <- t
			}
			startOfTick = startOfTick + divisor*d
		}
		done <- true
	}
	atomic.AddInt32(&t.killRoutinesCount, 1)
	done <- true
}

// AfterRoutine is a routine for Timer.After().
func (t *MockTimerClient) AfterRoutine(d int64, c chan time.Time, done chan bool, startOfAfter int64) {
	for !t.killAll {
		currentDuration := atomic.LoadInt64(&t.currentTime)
		if d > 0 && (currentDuration-startOfAfter)/d >= 1 {
			c <- time.Now()
			break
		}
	}
	atomic.AddInt32(&t.killRoutinesCount, 1)
}

// Tick mocks Timer.Tick().
func (t *MockTimerClient) Tick(d time.Duration) <-chan time.Time {
	// We use done and update channels on tick because it is long going process and infinite loop will
	// consume lot of CPU
	c := make(chan time.Time, 10)
	done := make(chan bool, 1)
	update := make(chan bool, 1)
	go t.TickRoutine(int64(d), c, done, update, atomic.LoadInt64(&t.currentTime))
	t.doneChannel = append(t.doneChannel, done)
	t.update = append(t.update, update)
	atomic.AddInt32(&t.tickCalled, 1)
	return c
}

// After mocks Timer.After().
func (t *MockTimerClient) After(d time.Duration) <-chan time.Time {
	c := make(chan time.Time, 10)
	done := make(chan bool, 1)
	go t.AfterRoutine(int64(d), c, done, atomic.LoadInt64(&t.currentTime))
	atomic.AddInt32(&t.afterCalled, 1)
	return c
}

// AfterCalledTimes calculates number of times after is called.
func (t *MockTimerClient) AfterCalledTimes() int32 {
	return atomic.LoadInt32(&t.afterCalled)
}

// TickCalledTimes calculates number of times tick is called.
func (t *MockTimerClient) TickCalledTimes() int32 {
	return atomic.LoadInt32(&t.tickCalled)
}
