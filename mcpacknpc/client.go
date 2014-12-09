package mcpacknpc

import (
	"bytes"
	"io/ioutil"

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
	content, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return mcpack.Unmarshal(content, reply)
}
