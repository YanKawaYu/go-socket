package gosocket

import "time"

type Timer struct {
	ticker     *time.Ticker
	tickerDone chan bool
}

func (timer *Timer) Stop() {
	//防止channel已关闭
	if timer.ticker != nil {
		timer.tickerDone <- true
	}
}

type Callback func()

func NewTimer(interval time.Duration, callback Callback) *Timer {
	timer := &Timer{
		ticker:     time.NewTicker(interval),
		tickerDone: make(chan bool),
	}
	go func() {
		defer func() {
			//循环停止后再停止计时器
			if timer.ticker != nil {
				timer.ticker.Stop()
			}
			timer.ticker = nil
			close(timer.tickerDone)
		}()
		//必须通过Label，否则break只会跳出select
	Loop:
		for {
			select {
			case <-timer.ticker.C:
				callback()
			case <-timer.tickerDone:
				break Loop
			}
		}
	}()
	return timer
}
