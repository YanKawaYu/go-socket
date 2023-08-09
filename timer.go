package gosocket

import "time"

// Timer is a utility class for the framework
type Timer struct {
	ticker     *time.Ticker
	tickerDone chan bool
}

func (timer *Timer) Stop() {
	//Validate timer.ticker first to avoid accessing a closed channel
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
			//Stop the timer once the loop is over
			//循环停止后再停止计时器
			if timer.ticker != nil {
				timer.ticker.Stop()
			}
			timer.ticker = nil
			close(timer.tickerDone)
		}()
		//This label is essential here. Or else the break can only break out of select block
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
