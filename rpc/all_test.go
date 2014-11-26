package rpc

import (
	"fmt"
	"net"
	"net/rpc"
	"testing"
)

type Args struct {
	A int `json:"A"`
	B int `json:"B"`
}

type Reply struct {
	C int `json:"C"`
}

type Arith int

func (t *Arith) Add(args *Args, reply *Reply) error {
	reply.C = args.A + args.B
	return nil
}

func init() {
	rpc.Register(new(Arith))
}

func TestClient(t *testing.T) {
	cli, srv := net.Pipe()
	go ServeConn(srv)

	client := NewClient(cli)
	defer client.Close()

	args := &Args{7, 8}
	reply := new(Reply)
	err := client.Call("Arith.Add", args, reply)
	if err != nil {
		t.Errorf("Add: expected no error but go string %q", err.Error())
	}
	fmt.Println(reply)
}
