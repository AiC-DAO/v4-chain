// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	types "github.com/cosmos/cosmos-sdk/types"
	mock "github.com/stretchr/testify/mock"
)

// MsgRouter is an autogenerated mock type for the MsgRouter type
type MsgRouter struct {
	mock.Mock
}

// Handler provides a mock function with given fields: msg
func (_m *MsgRouter) Handler(msg types.Msg) func(types.Context, types.Msg) (*types.Result, error) {
	ret := _m.Called(msg)

	var r0 func(types.Context, types.Msg) (*types.Result, error)
	if rf, ok := ret.Get(0).(func(types.Msg) func(types.Context, types.Msg) (*types.Result, error)); ok {
		r0 = rf(msg)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(func(types.Context, types.Msg) (*types.Result, error))
		}
	}

	return r0
}

type mockConstructorTestingTNewMsgRouter interface {
	mock.TestingT
	Cleanup(func())
}

// NewMsgRouter creates a new instance of MsgRouter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMsgRouter(t mockConstructorTestingTNewMsgRouter) *MsgRouter {
	mock := &MsgRouter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
