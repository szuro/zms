package subject

import "szuro.net/zms/zms"

type HTTPInput struct {
	baseInput
}

func NewHTTPInput(zmsConf zms.ZMSConf) (hi *HTTPInput, err error) {
	hi = &HTTPInput{}
	hi.config = zmsConf
	return
}

func (hi *HTTPInput) IsReady() bool { return false }
