// source: https://justapengu.in/blog/ticker
package mticker

import (
	"context"
	"time"
)

// NewMutableTicker provides a ticker with an interval which can be changed
func NewMutableTicker(t time.Duration) *MutableTicker {
	ticker := time.NewTicker(t)

	c := make(chan time.Time)
	d := make(chan time.Duration)

	ctx, cfn := context.WithCancel(context.Background())

	tk := &MutableTicker{
		ctx:      ctx,
		cfn:      cfn,
		ticker:   ticker,
		updateCh: d,
		c:        c,
		C:        c,
	}

	go tk.loop()

	return tk
}

// MutableTicker is a time.Ticker which can change update intervals
type MutableTicker struct {
	ctx      context.Context
	cfn      context.CancelFunc
	ticker   *time.Ticker
	updateCh chan time.Duration

	// c is a read/write private channel
	c chan time.Time
	// C is a read only channel
	C <-chan time.Time
}

func (m *MutableTicker) loop() {
	for {
		select {
		// the interval of the ticker has been updated.
		case d := <-m.updateCh:
			// stop the old ticker and replace it with a new one
			m.ticker.Stop()
			m.ticker = time.NewTicker(d)
		case t := <-m.ticker.C:
			// updates from the underlying ticker are passed into the writable version of C
			m.c <- t
		case <-m.ctx.Done():
			// use context.Done() to quit the loop when the CancelFunc is called.
			return
		}
	}
}

// UpdateInterval modifies the interval of the Ticker.
func (m *MutableTicker) UpdateInterval(t time.Duration) {
	m.updateCh <- t
}

// Stop the Ticker.
func (m *MutableTicker) Stop() {
	m.cfn()

	if m.ticker != nil {
		m.ticker.Stop()
	}

	m.c = nil
	m.updateCh = nil
}
