package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type EmptyStruct struct {
}

func ChannelHasData(c chan EmptyStruct) bool {
	var ok bool
	select {
	case <-c:
		ok = true
	default:
		ok = false

	}
	return ok
}

// This function is used so that test cases will not freeze if chan is not responsive
func TryToGetValue(ch chan EmptyStruct) *EmptyStruct {
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(100 * time.Millisecond)
		timeout <- true
	}()
	select {
	case v := <-ch:
		return &v
	case <-timeout:
		return nil
	}
}

func TickTestHelper(tickDuration int64, t *testing.T) {
	timer := &MockTimerClient{current: time.Unix(35534432431, 0)}
	tickChan := make(chan EmptyStruct, 1)
	tickFunc := func() {
		// Go routine started
		tickChan <- EmptyStruct{}
		t := timer.Tick(time.Duration(tickDuration))
		for {
			<-t
			tickChan <- EmptyStruct{}
		}
	}

	go tickFunc()

	// Go routine to monitor tick started
	<-tickChan
	testCasesTicksToTrigger := []int{1, 2, 1000}
	var durationIncremented int64
	for _, ticksToTrigger := range testCasesTicksToTrigger {
		for i := 0; i < ticksToTrigger; i++ {
			var ok bool
			ok = ChannelHasData(tickChan)
			assert.False(t, ok)
			initialIncrement := tickDuration / 2
			// Not enough to trigger tick
			timer.Advance(time.Duration(initialIncrement))
			durationIncremented += initialIncrement
			ok = ChannelHasData(tickChan)
			assert.False(t, ok)
			// tick triggered
			timer.Advance(time.Duration(tickDuration))
			durationIncremented += tickDuration
			val := TryToGetValue(tickChan)
			assert.NotNil(t,
				val,
				fmt.Sprintf("Expected value passed thru the channel. Tick Duration: %v, Tick Trigger Iteration: %v, Ticket To Trigger: %v Current Clock Time: %v",
					tickDuration,
					i,
					ticksToTrigger,
					timer.current))

			// Adding 4th of the duration to trigger
			durationForth := tickDuration / 4
			timer.Advance(time.Duration(durationForth))
			durationIncremented += durationForth
			ok = ChannelHasData(tickChan)
			assert.False(t, ok)

			// Leave the duration with exact divisor so that next loop can assume
			// duration increment is zero
			finalIncrement := tickDuration*2 - durationIncremented
			// tick triggered
			timer.Advance(time.Duration(finalIncrement))
			val = TryToGetValue(tickChan)
			assert.NotNil(t, val)
			durationIncremented = 0
		}
	}

	assert.EqualValues(t, 1, timer.TickCalledTimes())
}

func TestTickDuration454(t *testing.T) {
	var tickDuration int64
	tickDuration = 454
	TickTestHelper(tickDuration, t)
}

func TestAfter(t *testing.T) {
	var afterDuration int64
	afterDuration = 10
	timer := MockTimerClient{current: time.Unix(2153567564, 0)}
	afterChan := make(chan EmptyStruct, 1)
	tickFunc := func() {
		// Go routine started
		afterChan <- EmptyStruct{}
		t := timer.After(time.Duration(afterDuration))
		for {
			<-t
			afterChan <- EmptyStruct{}
		}
	}

	go tickFunc()

	// Go routine started to monitor after messages
	<-afterChan
	var ok bool
	ok = ChannelHasData(afterChan)
	assert.False(t, ok)
	initialIncrement := afterDuration / 2
	// Not enough to trigger after
	timer.Advance(time.Duration(initialIncrement))
	ok = ChannelHasData(afterChan)
	assert.False(t, ok)
	// after triggered
	timer.Advance(time.Duration(afterDuration))
	val := TryToGetValue(afterChan)
	assert.NotNil(t, val, fmt.Sprintf("Expected value passed thru the channel. After Duration: %v, Current Clock Time: %v", afterDuration, timer.current))

	// After should trigger only once compared to tick
	timer.Advance(time.Duration(afterDuration))
	ok = ChannelHasData(afterChan)
	assert.False(t, ok)

	assert.EqualValues(t, 1, timer.AfterCalledTimes())
}

func TestAfterTickTogether(t *testing.T) {
	var tickDuration int64
	tickDuration = 10
	afterDuration := tickDuration * 2
	timer := MockTimerClient{current: time.Unix(23082153551, 0)}
	tickChan := make(chan EmptyStruct, 1)
	afterChan := make(chan EmptyStruct, 1)
	tickFunc := func() {
		// Go routine started
		tick := timer.Tick(time.Duration(tickDuration))
		tickChan <- EmptyStruct{}
		for {
			select {
			case <-tick:
				tickChan <- EmptyStruct{}
			}
		}
	}
	afterFunc := func() {
		// Go routine started
		after := timer.After(time.Duration(afterDuration))
		afterChan <- EmptyStruct{}
		for {
			select {
			case <-after:
				afterChan <- EmptyStruct{}

			}
		}
	}

	go tickFunc()
	go afterFunc()

	// Go routine started to monitor tick and after events
	<-tickChan
	<-afterChan
	testCasesTicksToTrigger := []int{1, 2, 100}
	var durationIncremented int64
	for triggerIndex, ticksToTrigger := range testCasesTicksToTrigger {
		for i := 0; i < ticksToTrigger; i++ {
			var ok bool
			ok = ChannelHasData(tickChan)
			assert.False(t, ok)
			ok = ChannelHasData(afterChan)
			assert.False(t, ok)
			initialIncrement := tickDuration / 2
			// Not enough to trigger tick
			timer.Advance(time.Duration(initialIncrement))
			durationIncremented += initialIncrement
			ok = ChannelHasData(tickChan)
			assert.False(t, ok)
			ok = ChannelHasData(afterChan)
			assert.False(t, ok)
			// tick triggered
			timer.Advance(time.Duration(tickDuration))
			durationIncremented += tickDuration
			val := TryToGetValue(tickChan)
			assert.NotNil(t, val)
			ok = ChannelHasData(afterChan)
			assert.False(t, ok)

			// Adding 4th of the duration to trigger
			durationForth := tickDuration / 4
			timer.Advance(time.Duration(durationForth))
			durationIncremented += durationForth
			ok = ChannelHasData(tickChan)
			assert.False(t, ok)
			ok = ChannelHasData(afterChan)
			assert.False(t, ok)

			// Leave the duration with exact divisor so that next loop can assume
			// duration increment is zero
			finalIncrement := tickDuration*2 - durationIncremented
			// tick triggered
			timer.Advance(time.Duration(finalIncrement))
			// After will only trigger for first iteration as it only trigger once
			if (triggerIndex == 0) && (i == 0) {
				val = TryToGetValue(afterChan)
				assert.NotNil(t, val)
			} else {
				ok = ChannelHasData(afterChan)
				assert.False(t, ok)
			}
			val = TryToGetValue(tickChan)
			assert.NotNil(t, val)

			durationIncremented = 0
		}
	}

	assert.EqualValues(t, 1, timer.TickCalledTimes())
	assert.EqualValues(t, 1, timer.AfterCalledTimes())
}
