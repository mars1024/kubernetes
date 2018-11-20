package cmdb

import (
	"github.com/stretchr/testify/mock"
)

type MockClient struct {
	mock.Mock
}

func NewMockClient() *MockClient {
	return &MockClient{}
}

func (c *MockClient) AddContainerInfo(reqInfo []byte) error {
	args := c.Called(reqInfo)
	return args.Error(0)
}

func (c *MockClient) UpdateContainerInfo(reqInfo []byte) error {
	args := c.Called(reqInfo)
	return args.Error(0)
}

func (c *MockClient) DeleteContainerInfo(sn string) error {
	args := c.Called(sn)
	return args.Error(0)
}

func (c *MockClient) GetContainerInfo(sn string) (*CMDBResp, error) {
	args := c.Called(sn)
	return args.Get(0).(*CMDBResp), args.Error(1)
}
