//software: GoLand
//file: cmd_test.go
//time: 2021-03-16 14:37
package main

import (
	"testing"
	"time"
)

func TestA(t *testing.T) {
	ch := make(chan int)
	go func() {
		time.Sleep(5 * time.Second)
		ch <- 1
	}()
	t.Log("Lis start  ", time.Now())
	<-ch
	t.Log("Lis end  ", time.Now())
}
