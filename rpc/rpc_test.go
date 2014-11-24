package rpc

import (
	"fmt"
	"testing"

	"gitlab.baidu.com/niushaofeng/gomcpack/mcpack"
)

type Request struct {
	Header  Header  `json:"header"`
	Content Content `json:"content"`
}

type Header struct {
}

type Content struct {
	Method string `json:"method"`
	Params Params `json:"params"`
}

type Params struct {
	Data []byte
}

func (p *Params) UnmarshalMCPACK(b []byte) error {
	p.Data = b
	return nil
}

type MethodParams struct {
	Params interface{} `json:"params"`
}

type GetReq struct {
	Key string `json:"key"`
}

func Get(req GetParams, rsp *GetResp) error {

}

func TestUnmarshal(t *testing.T) {
	raw := []byte{mcpack.MCPACKV2_OBJECT, 0, 70, 0, 0, 0, 1, 0, 0, 0,
		mcpack.MCPACKV2_OBJECT, 8, 52, 0, 0, 0, 'c', 'o', 'n', 't', 'e', 'n', 't', 0, 2, 0, 0, 0,
		mcpack.MCPACKV2_STRING, 7, 4, 0, 0, 0, 'm', 'e', 't', 'h', 'o', 'd', 0, 'g', 'e', 't', 0,
		mcpack.MCPACKV2_OBJECT, 7, 18, 0, 0, 0, 'p', 'a', 'r', 'a', 'm', 's', 0, 1, 0, 0, 0,
		mcpack.MCPACKV2_STRING, 4, 4, 0, 0, 0, 'k', 'e', 'y', 0, 'f', 'o', 'o', 0,
	}
	request := &Request{}
	err := mcpack.Unmarshal(raw, request)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("%#+v\n", request)
	if request.Content.Method == "get" {
		get := &MethodParams{Params: &GetReq{}}
		err = mcpack.Unmarshal(request.Content.Params.Data, get)
		if err != nil {
			t.Error(err)
		}
		fmt.Printf("%#+v\n", get.Params)
	}

	var method interface{} = Get

}
