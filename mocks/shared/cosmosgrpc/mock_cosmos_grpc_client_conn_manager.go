// Code generated by mockery v2.46.3. DO NOT EDIT.

package cosmosgrpc

import (
	context "golang.org/x/net/context"

	grpc "github.com/cosmos/gogoproto/grpc"

	mock "github.com/stretchr/testify/mock"
)

// MockCosmosGRPCClientConnManager is an autogenerated mock type for the CosmosGRPCClientConnManager type
type MockCosmosGRPCClientConnManager struct {
	mock.Mock
}

type MockCosmosGRPCClientConnManager_Expecter struct {
	mock *mock.Mock
}

func (_m *MockCosmosGRPCClientConnManager) EXPECT() *MockCosmosGRPCClientConnManager_Expecter {
	return &MockCosmosGRPCClientConnManager_Expecter{mock: &_m.Mock}
}

// GetClient provides a mock function with given fields: ctx, chainID
func (_m *MockCosmosGRPCClientConnManager) GetClient(ctx context.Context, chainID string) (grpc.ClientConn, error) {
	ret := _m.Called(ctx, chainID)

	if len(ret) == 0 {
		panic("no return value specified for GetClient")
	}

	var r0 grpc.ClientConn
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (grpc.ClientConn, error)); ok {
		return rf(ctx, chainID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) grpc.ClientConn); ok {
		r0 = rf(ctx, chainID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(grpc.ClientConn)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, chainID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockCosmosGRPCClientConnManager_GetClient_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetClient'
type MockCosmosGRPCClientConnManager_GetClient_Call struct {
	*mock.Call
}

// GetClient is a helper method to define mock.On call
//   - ctx context.Context
//   - chainID string
func (_e *MockCosmosGRPCClientConnManager_Expecter) GetClient(ctx interface{}, chainID interface{}) *MockCosmosGRPCClientConnManager_GetClient_Call {
	return &MockCosmosGRPCClientConnManager_GetClient_Call{Call: _e.mock.On("GetClient", ctx, chainID)}
}

func (_c *MockCosmosGRPCClientConnManager_GetClient_Call) Run(run func(ctx context.Context, chainID string)) *MockCosmosGRPCClientConnManager_GetClient_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *MockCosmosGRPCClientConnManager_GetClient_Call) Return(_a0 grpc.ClientConn, _a1 error) *MockCosmosGRPCClientConnManager_GetClient_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockCosmosGRPCClientConnManager_GetClient_Call) RunAndReturn(run func(context.Context, string) (grpc.ClientConn, error)) *MockCosmosGRPCClientConnManager_GetClient_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockCosmosGRPCClientConnManager creates a new instance of MockCosmosGRPCClientConnManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockCosmosGRPCClientConnManager(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockCosmosGRPCClientConnManager {
	mock := &MockCosmosGRPCClientConnManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
