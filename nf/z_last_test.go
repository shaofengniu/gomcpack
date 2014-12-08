package nf_test

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

func interestingGoroutines() (gs []string) {
	buf := make([]byte, 2<<20)
	buf = buf[:runtime.Stack(buf, true)]
	for _, g := range strings.Split(string(buf), "\n\n") {
		sl := strings.SplitN(g, "\n", 2)
		if len(sl) != 2 {
			continue
		}
		stack := strings.TrimSpace(sl[1])
		if stack == "" ||
			strings.Contains(stack, "created by testing.RunTests") ||
			strings.Contains(stack, "testing.Main") ||
			strings.Contains(stack, "runtime.goexit") ||
			strings.Contains(stack, "created by runtime.gc") ||
			strings.Contains(stack, "runtime.MHeap_Scavenger") {
			continue
		}
		gs = append(gs, stack)
		fmt.Println("---------")
		fmt.Println(stack)
	}
	sort.Strings(gs)
	return
}

func afterTest(t *testing.T) {
	time.Sleep(1 * time.Second)
	interestingGoroutines()
}
