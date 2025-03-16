// Code generated by mockery v2.46.3. DO NOT EDIT.

package fundrebalancer

import (
	db "github.com/skip-mev/go-fast-solver/db/gen/db"
	context "golang.org/x/net/context"

	mock "github.com/stretchr/testify/mock"
)

// MockDatabase is an autogenerated mock type for the Database type
type MockDatabase struct {
	mock.Mock
}

type MockDatabase_Expecter struct {
	mock *mock.Mock
}

func (_m *MockDatabase) EXPECT() *MockDatabase_Expecter {
	return &MockDatabase_Expecter{mock: &_m.Mock}
}

// GetAllPendingRebalanceTransfers provides a mock function with given fields: ctx
func (_m *MockDatabase) GetAllPendingRebalanceTransfers(ctx context.Context) ([]db.GetAllPendingRebalanceTransfersRow, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for GetAllPendingRebalanceTransfers")
	}

	var r0 []db.GetAllPendingRebalanceTransfersRow
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) ([]db.GetAllPendingRebalanceTransfersRow, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) []db.GetAllPendingRebalanceTransfersRow); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]db.GetAllPendingRebalanceTransfersRow)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockDatabase_GetAllPendingRebalanceTransfers_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetAllPendingRebalanceTransfers'
type MockDatabase_GetAllPendingRebalanceTransfers_Call struct {
	*mock.Call
}

// GetAllPendingRebalanceTransfers is a helper method to define mock.On call
//   - ctx context.Context
func (_e *MockDatabase_Expecter) GetAllPendingRebalanceTransfers(ctx interface{}) *MockDatabase_GetAllPendingRebalanceTransfers_Call {
	return &MockDatabase_GetAllPendingRebalanceTransfers_Call{Call: _e.mock.On("GetAllPendingRebalanceTransfers", ctx)}
}

func (_c *MockDatabase_GetAllPendingRebalanceTransfers_Call) Run(run func(ctx context.Context)) *MockDatabase_GetAllPendingRebalanceTransfers_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockDatabase_GetAllPendingRebalanceTransfers_Call) Return(_a0 []db.GetAllPendingRebalanceTransfersRow, _a1 error) *MockDatabase_GetAllPendingRebalanceTransfers_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockDatabase_GetAllPendingRebalanceTransfers_Call) RunAndReturn(run func(context.Context) ([]db.GetAllPendingRebalanceTransfersRow, error)) *MockDatabase_GetAllPendingRebalanceTransfers_Call {
	_c.Call.Return(run)
	return _c
}

// GetPendingRebalanceTransfersToChain provides a mock function with given fields: ctx, destinationChainID
func (_m *MockDatabase) GetPendingRebalanceTransfersToChain(ctx context.Context, destinationChainID string) ([]db.GetPendingRebalanceTransfersToChainRow, error) {
	ret := _m.Called(ctx, destinationChainID)

	if len(ret) == 0 {
		panic("no return value specified for GetPendingRebalanceTransfersToChain")
	}

	var r0 []db.GetPendingRebalanceTransfersToChainRow
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) ([]db.GetPendingRebalanceTransfersToChainRow, error)); ok {
		return rf(ctx, destinationChainID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) []db.GetPendingRebalanceTransfersToChainRow); ok {
		r0 = rf(ctx, destinationChainID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]db.GetPendingRebalanceTransfersToChainRow)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, destinationChainID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockDatabase_GetPendingRebalanceTransfersToChain_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetPendingRebalanceTransfersToChain'
type MockDatabase_GetPendingRebalanceTransfersToChain_Call struct {
	*mock.Call
}

// GetPendingRebalanceTransfersToChain is a helper method to define mock.On call
//   - ctx context.Context
//   - destinationChainID string
func (_e *MockDatabase_Expecter) GetPendingRebalanceTransfersToChain(ctx interface{}, destinationChainID interface{}) *MockDatabase_GetPendingRebalanceTransfersToChain_Call {
	return &MockDatabase_GetPendingRebalanceTransfersToChain_Call{Call: _e.mock.On("GetPendingRebalanceTransfersToChain", ctx, destinationChainID)}
}

func (_c *MockDatabase_GetPendingRebalanceTransfersToChain_Call) Run(run func(ctx context.Context, destinationChainID string)) *MockDatabase_GetPendingRebalanceTransfersToChain_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *MockDatabase_GetPendingRebalanceTransfersToChain_Call) Return(_a0 []db.GetPendingRebalanceTransfersToChainRow, _a1 error) *MockDatabase_GetPendingRebalanceTransfersToChain_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockDatabase_GetPendingRebalanceTransfersToChain_Call) RunAndReturn(run func(context.Context, string) ([]db.GetPendingRebalanceTransfersToChainRow, error)) *MockDatabase_GetPendingRebalanceTransfersToChain_Call {
	_c.Call.Return(run)
	return _c
}

// InsertRebalanceTransfer provides a mock function with given fields: ctx, arg
func (_m *MockDatabase) InsertRebalanceTransfer(ctx context.Context, arg db.InsertRebalanceTransferParams) (int64, error) {
	ret := _m.Called(ctx, arg)

	if len(ret) == 0 {
		panic("no return value specified for InsertRebalanceTransfer")
	}

	var r0 int64
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, db.InsertRebalanceTransferParams) (int64, error)); ok {
		return rf(ctx, arg)
	}
	if rf, ok := ret.Get(0).(func(context.Context, db.InsertRebalanceTransferParams) int64); ok {
		r0 = rf(ctx, arg)
	} else {
		r0 = ret.Get(0).(int64)
	}

	if rf, ok := ret.Get(1).(func(context.Context, db.InsertRebalanceTransferParams) error); ok {
		r1 = rf(ctx, arg)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockDatabase_InsertRebalanceTransfer_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'InsertRebalanceTransfer'
type MockDatabase_InsertRebalanceTransfer_Call struct {
	*mock.Call
}

// InsertRebalanceTransfer is a helper method to define mock.On call
//   - ctx context.Context
//   - arg db.InsertRebalanceTransferParams
func (_e *MockDatabase_Expecter) InsertRebalanceTransfer(ctx interface{}, arg interface{}) *MockDatabase_InsertRebalanceTransfer_Call {
	return &MockDatabase_InsertRebalanceTransfer_Call{Call: _e.mock.On("InsertRebalanceTransfer", ctx, arg)}
}

func (_c *MockDatabase_InsertRebalanceTransfer_Call) Run(run func(ctx context.Context, arg db.InsertRebalanceTransferParams)) *MockDatabase_InsertRebalanceTransfer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(db.InsertRebalanceTransferParams))
	})
	return _c
}

func (_c *MockDatabase_InsertRebalanceTransfer_Call) Return(_a0 int64, _a1 error) *MockDatabase_InsertRebalanceTransfer_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockDatabase_InsertRebalanceTransfer_Call) RunAndReturn(run func(context.Context, db.InsertRebalanceTransferParams) (int64, error)) *MockDatabase_InsertRebalanceTransfer_Call {
	_c.Call.Return(run)
	return _c
}

// InsertSubmittedTx provides a mock function with given fields: ctx, arg
func (_m *MockDatabase) InsertSubmittedTx(ctx context.Context, arg db.InsertSubmittedTxParams) (db.SubmittedTx, error) {
	ret := _m.Called(ctx, arg)

	if len(ret) == 0 {
		panic("no return value specified for InsertSubmittedTx")
	}

	var r0 db.SubmittedTx
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, db.InsertSubmittedTxParams) (db.SubmittedTx, error)); ok {
		return rf(ctx, arg)
	}
	if rf, ok := ret.Get(0).(func(context.Context, db.InsertSubmittedTxParams) db.SubmittedTx); ok {
		r0 = rf(ctx, arg)
	} else {
		r0 = ret.Get(0).(db.SubmittedTx)
	}

	if rf, ok := ret.Get(1).(func(context.Context, db.InsertSubmittedTxParams) error); ok {
		r1 = rf(ctx, arg)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockDatabase_InsertSubmittedTx_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'InsertSubmittedTx'
type MockDatabase_InsertSubmittedTx_Call struct {
	*mock.Call
}

// InsertSubmittedTx is a helper method to define mock.On call
//   - ctx context.Context
//   - arg db.InsertSubmittedTxParams
func (_e *MockDatabase_Expecter) InsertSubmittedTx(ctx interface{}, arg interface{}) *MockDatabase_InsertSubmittedTx_Call {
	return &MockDatabase_InsertSubmittedTx_Call{Call: _e.mock.On("InsertSubmittedTx", ctx, arg)}
}

func (_c *MockDatabase_InsertSubmittedTx_Call) Run(run func(ctx context.Context, arg db.InsertSubmittedTxParams)) *MockDatabase_InsertSubmittedTx_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(db.InsertSubmittedTxParams))
	})
	return _c
}

func (_c *MockDatabase_InsertSubmittedTx_Call) Return(_a0 db.SubmittedTx, _a1 error) *MockDatabase_InsertSubmittedTx_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockDatabase_InsertSubmittedTx_Call) RunAndReturn(run func(context.Context, db.InsertSubmittedTxParams) (db.SubmittedTx, error)) *MockDatabase_InsertSubmittedTx_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateTransferStatus provides a mock function with given fields: ctx, arg
func (_m *MockDatabase) UpdateTransferStatus(ctx context.Context, arg db.UpdateTransferStatusParams) error {
	ret := _m.Called(ctx, arg)

	if len(ret) == 0 {
		panic("no return value specified for UpdateTransferStatus")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, db.UpdateTransferStatusParams) error); ok {
		r0 = rf(ctx, arg)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockDatabase_UpdateTransferStatus_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateTransferStatus'
type MockDatabase_UpdateTransferStatus_Call struct {
	*mock.Call
}

// UpdateTransferStatus is a helper method to define mock.On call
//   - ctx context.Context
//   - arg db.UpdateTransferStatusParams
func (_e *MockDatabase_Expecter) UpdateTransferStatus(ctx interface{}, arg interface{}) *MockDatabase_UpdateTransferStatus_Call {
	return &MockDatabase_UpdateTransferStatus_Call{Call: _e.mock.On("UpdateTransferStatus", ctx, arg)}
}

func (_c *MockDatabase_UpdateTransferStatus_Call) Run(run func(ctx context.Context, arg db.UpdateTransferStatusParams)) *MockDatabase_UpdateTransferStatus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(db.UpdateTransferStatusParams))
	})
	return _c
}

func (_c *MockDatabase_UpdateTransferStatus_Call) Return(_a0 error) *MockDatabase_UpdateTransferStatus_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockDatabase_UpdateTransferStatus_Call) RunAndReturn(run func(context.Context, db.UpdateTransferStatusParams) error) *MockDatabase_UpdateTransferStatus_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockDatabase creates a new instance of MockDatabase. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockDatabase(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockDatabase {
	mock := &MockDatabase{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
