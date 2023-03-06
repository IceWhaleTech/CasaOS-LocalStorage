package utils

import (
	"sync"
	"time"
)

func MustParseCNTime(str string) time.Time {
	lastOpTime, _ := time.ParseInLocation("2006-01-02 15:04:05 -07", str+" +08", time.Local)
	return lastOpTime
}

func NewDebounce(interval time.Duration) func(f func()) {
	var timer *time.Timer
	var lock sync.Mutex
	return func(f func()) {
		lock.Lock()
		defer lock.Unlock()
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(interval, f)
	}
}

func NewDebounce2(interval time.Duration, f func()) func() {
	var timer *time.Timer
	var lock sync.Mutex
	return func() {
		lock.Lock()
		defer lock.Unlock()
		if timer == nil {
			timer = time.AfterFunc(interval, f)
		}
		(*time.Timer)(timer).Reset(interval)
	}
}
