package testutil

import "github.com/golang/mock/gomock"

type Calls []*gomock.Call

func (c Calls) Before(calls Calls) Calls {
	for _, newer := range calls {
		c.BeforeCall(newer)
	}
	return c
}

func (c Calls) BeforeCall(call *gomock.Call) Calls {
	for _, older := range c {
		call.After(older)
	}
	return c
}

func (c Calls) After(calls Calls) Calls {
	for _, older := range calls {
		c.AfterCall(older)
	}
	return c
}

func (c Calls) AfterCall(call *gomock.Call) Calls {
	for _, newer := range c {
		newer.After(call)
	}
	return c
}
