package mcpacknpc_test

import (
	"testing"

	. "gitlab.baidu.com/ksarch/gomcpack/mcpacknpc"
	"gitlab.baidu.com/niushaofeng/gomcpack/npc/npctest"
)

type Ping struct {
	Data string
}

type Pong struct {
	Data string
}

func TestHandler(t *testing.T) {
	handler, err := NewHandler(func(in Ping, out *Pong) error {
		if in.Data != "ping" {
			t.Fatalf("expected ping, got %q", in.Data)
		}
		out.Data = "pong"
		return nil
	})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	s := npctest.NewServer(handler)
	defer s.Close()

	c := NewClient([]string{s.Listener.Addr().String()})
	defer c.Close()
	var pong Pong
	err = c.Call(Ping{"ping"}, &pong)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if pong.Data != "pong" {
		t.Fatalf("expected pong, got %q", pong.Data)
	}
}

func BenchmarkClientServer(b *testing.B) {
	b.ReportAllocs()
	b.StopTimer()
	handler, err := NewHandler(func(in Ping, out *Pong) error {
		if in.Data != "ping" {
			b.Fatalf("expected ping, got %q", in.Data)
		}
		out.Data = "pong"
		return nil
	})
	if err != nil {
		b.Fatalf("NewHandler: %v", err)
	}

	s := npctest.NewServer(handler)
	defer s.Close()

	c := NewClient([]string{s.Listener.Addr().String()})
	defer c.Close()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		var pong Pong
		err = c.Call(Ping{"ping"}, &pong)
		if err != nil {
			b.Fatalf("Call: %v", err)
		}
		if pong.Data != "pong" {
			b.Fatalf("expected pong, got %q", pong.Data)
		}
	}
}

func benchmarkClientServerParallel(b *testing.B, parallelism int) {
	b.ReportAllocs()
	b.StopTimer()
	handler, err := NewHandler(func(in Ping, out *Pong) error {
		if in.Data != "ping" {
			b.Fatalf("expected ping, got %q", in.Data)
		}
		out.Data = "pong"
		return nil
	})
	if err != nil {
		b.Fatalf("NewHandler: %v", err)
	}

	s := npctest.NewServer(handler)
	defer s.Close()

	c := NewClient([]string{s.Listener.Addr().String()})
	defer c.Close()
	b.SetParallelism(parallelism)
	b.StartTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var pong Pong
			err = c.Call(Ping{"ping"}, &pong)
			if err != nil {
				b.Fatalf("Call: %v", err)
			}
			if pong.Data != "pong" {
				b.Fatalf("expected pong, got %q", pong.Data)
			}
		}
	})
}
func BenchmarkClientServerParallel4(b *testing.B) {
	benchmarkClientServerParallel(b, 4)
}

func BenchmarkClientServerParallel64(b *testing.B) {
	benchmarkClientServerParallel(b, 64)
}
