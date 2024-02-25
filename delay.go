package easynet

import (
	"time"
)

const (
	maxDelayDuration = 1 * time.Second
)

type Delay struct {
	duration time.Duration
}

func (d *Delay) Sleep() {
	d.Up()
	d.sleep()
}

func (d *Delay) sleep() {
	if d.duration > 0 {
		time.Sleep(d.duration)
	}
}

func (d *Delay) Up() {
	if d.duration == 0 {
		d.duration = 5 * time.Millisecond
		return
	}

	d.duration *= 2

	if d.duration > maxDelayDuration {
		d.duration = maxDelayDuration
	}
}

func (d *Delay) Reset() {
	d.duration = 0
}
