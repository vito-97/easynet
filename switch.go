package easynet

import "sync/atomic"

type Switcher interface {
	isStarted() bool
	isStopped() bool

	setStarted() bool
	setStopped() bool

	reset() bool
}

type Switch struct {
	started uint32
	stopped uint32
}

// isStarted 判断是否为打开
func (s *Switch) isStarted() bool {
	return atomic.LoadUint32(&s.started) != 0
}

// setStarted 标识为已打开
func (s *Switch) setStarted() bool {
	return atomic.CompareAndSwapUint32(&s.started, 0, 1)
}

// isStopped 判断是否为关闭
func (s *Switch) isStopped() bool {
	return atomic.LoadUint32(&s.stopped) != 0
}

// setStopped 标识为已关闭
func (s *Switch) setStopped() bool {
	return atomic.CompareAndSwapUint32(&s.stopped, 0, 1)
}

// reset 将改动的状态还原
func (s *Switch) reset() bool {
	if s.isStarted() && !atomic.CompareAndSwapUint32(&s.started, 1, 0) {
		return false
	}

	if s.isStopped() && !atomic.CompareAndSwapUint32(&s.stopped, 1, 0) {
		return false
	}

	return true
}
