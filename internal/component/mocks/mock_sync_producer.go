// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/IBM/sarama (interfaces: SyncProducer)

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	sarama "github.com/IBM/sarama"
	gomock "go.uber.org/mock/gomock"
)

// MockSyncProducer is a mock of SyncProducer interface.
type MockSyncProducer struct {
	ctrl     *gomock.Controller
	recorder *MockSyncProducerMockRecorder
}

// MockSyncProducerMockRecorder is the mock recorder for MockSyncProducer.
type MockSyncProducerMockRecorder struct {
	mock *MockSyncProducer
}

// NewMockSyncProducer creates a new mock instance.
func NewMockSyncProducer(ctrl *gomock.Controller) *MockSyncProducer {
	mock := &MockSyncProducer{ctrl: ctrl}
	mock.recorder = &MockSyncProducerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSyncProducer) EXPECT() *MockSyncProducerMockRecorder {
	return m.recorder
}

// AbortTxn mocks base method.
func (m *MockSyncProducer) AbortTxn() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AbortTxn")
	ret0, _ := ret[0].(error)
	return ret0
}

// AbortTxn indicates an expected call of AbortTxn.
func (mr *MockSyncProducerMockRecorder) AbortTxn() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AbortTxn", reflect.TypeOf((*MockSyncProducer)(nil).AbortTxn))
}

// AddMessageToTxn mocks base method.
func (m *MockSyncProducer) AddMessageToTxn(arg0 *sarama.ConsumerMessage, arg1 string, arg2 *string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddMessageToTxn", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddMessageToTxn indicates an expected call of AddMessageToTxn.
func (mr *MockSyncProducerMockRecorder) AddMessageToTxn(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddMessageToTxn", reflect.TypeOf((*MockSyncProducer)(nil).AddMessageToTxn), arg0, arg1, arg2)
}

// AddOffsetsToTxn mocks base method.
func (m *MockSyncProducer) AddOffsetsToTxn(arg0 map[string][]*sarama.PartitionOffsetMetadata, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddOffsetsToTxn", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddOffsetsToTxn indicates an expected call of AddOffsetsToTxn.
func (mr *MockSyncProducerMockRecorder) AddOffsetsToTxn(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddOffsetsToTxn", reflect.TypeOf((*MockSyncProducer)(nil).AddOffsetsToTxn), arg0, arg1)
}

// BeginTxn mocks base method.
func (m *MockSyncProducer) BeginTxn() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BeginTxn")
	ret0, _ := ret[0].(error)
	return ret0
}

// BeginTxn indicates an expected call of BeginTxn.
func (mr *MockSyncProducerMockRecorder) BeginTxn() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BeginTxn", reflect.TypeOf((*MockSyncProducer)(nil).BeginTxn))
}

// Close mocks base method.
func (m *MockSyncProducer) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockSyncProducerMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockSyncProducer)(nil).Close))
}

// CommitTxn mocks base method.
func (m *MockSyncProducer) CommitTxn() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CommitTxn")
	ret0, _ := ret[0].(error)
	return ret0
}

// CommitTxn indicates an expected call of CommitTxn.
func (mr *MockSyncProducerMockRecorder) CommitTxn() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CommitTxn", reflect.TypeOf((*MockSyncProducer)(nil).CommitTxn))
}

// IsTransactional mocks base method.
func (m *MockSyncProducer) IsTransactional() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsTransactional")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsTransactional indicates an expected call of IsTransactional.
func (mr *MockSyncProducerMockRecorder) IsTransactional() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsTransactional", reflect.TypeOf((*MockSyncProducer)(nil).IsTransactional))
}

// SendMessage mocks base method.
func (m *MockSyncProducer) SendMessage(arg0 *sarama.ProducerMessage) (int32, int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendMessage", arg0)
	ret0, _ := ret[0].(int32)
	ret1, _ := ret[1].(int64)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// SendMessage indicates an expected call of SendMessage.
func (mr *MockSyncProducerMockRecorder) SendMessage(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMessage", reflect.TypeOf((*MockSyncProducer)(nil).SendMessage), arg0)
}

// SendMessages mocks base method.
func (m *MockSyncProducer) SendMessages(arg0 []*sarama.ProducerMessage) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendMessages", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMessages indicates an expected call of SendMessages.
func (mr *MockSyncProducerMockRecorder) SendMessages(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMessages", reflect.TypeOf((*MockSyncProducer)(nil).SendMessages), arg0)
}

// TxnStatus mocks base method.
func (m *MockSyncProducer) TxnStatus() sarama.ProducerTxnStatusFlag {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TxnStatus")
	ret0, _ := ret[0].(sarama.ProducerTxnStatusFlag)
	return ret0
}

// TxnStatus indicates an expected call of TxnStatus.
func (mr *MockSyncProducerMockRecorder) TxnStatus() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TxnStatus", reflect.TypeOf((*MockSyncProducer)(nil).TxnStatus))
}
