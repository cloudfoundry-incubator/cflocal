package wait

import "time"

func New(d time.Duration) (waiter <-chan time.Time, done func()) {
	wait := make(chan time.Time)
	stop := make(chan struct{})
	go func() {
		for {
			select {
			case wait <- time.Now():
				time.Sleep(d)
			case <-stop:
				return
			}
		}
	}()
	return wait, func() {
		close(stop)
	}
}
