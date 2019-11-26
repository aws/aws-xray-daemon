package timer

import "time"

// Timer interface
type Timer interface {
	Tick(d time.Duration) <-chan time.Time
	After(d time.Duration) <-chan time.Time
}

// Client is an empty timer client.
type Client struct{}

// Tick is wrapper to time.Tick().
func (t *Client) Tick(d time.Duration) <-chan time.Time {
	return time.Tick(d)
}

// After is wrapper to time.After().
func (t *Client) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}
