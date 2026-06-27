package routine

import (
	"sync"
)

// FireAndForget executes `f()` in a new go routine and auto recovers if panic.
//
// **Note:** Use this only if you are not interested in the result of `f()`
// and don't want to block the parent go routine.
func FireAndForget(f func(), wg ...*sync.WaitGroup) {
	if len(wg) > 0 && wg[0] != nil {
		wg[0].Add(1)
	}

	go func() {
		if len(wg) > 0 && wg[0] != nil {
			defer wg[0].Done()
		}

		defer func() {
			recover()
		}()

		f()
	}()
}
