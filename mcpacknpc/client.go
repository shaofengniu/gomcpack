package mcpacknpc

import (
	"bytes"

	"gitlab.baidu.com/niushaofeng/gomcpack/mcpack"
	"gitlab.baidu.com/niushaofeng/gomcpack/npc"
)

type Client struct {
	*npc.Client
}

func NewClient(server []string) *Client {
	c := npc.NewClient(server)
	return &Client{Client: c}
}

func (c *Client) Call(args interface{}, reply interface{}) error {
	content, err := mcpack.Marshal(args)
	if err != nil {
		return err
	}
	resp, err := c.Client.Do(npc.NewRequest(bytes.NewReader(content)))
	if err != nil {
		return err
	}
	return mcpack.Unmarshal(resp.Body, reply)
}
