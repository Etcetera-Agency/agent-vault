package mailproxy

import (
	"io"
	"net"
	"sync"
)

func Relay(left, right net.Conn) {
	var once sync.Once
	closeBoth := func() {
		_ = left.Close()
		_ = right.Close()
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(left, right)
		once.Do(closeBoth)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(right, left)
		once.Do(closeBoth)
	}()
	wg.Wait()
}
