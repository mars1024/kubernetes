package armory

import (
	"github.com/stretchr/testify/mock"
)

type MockClient struct {
	mock.Mock
}

func NewMockClient() *MockClient {
	return &MockClient{}
}

func (c *MockClient) QueryDevice(query string) (*DeviceInfo, error) {
	args := c.Called(query)
	return args.Get(0).(*DeviceInfo), args.Error(1)
}

func (c *MockClient) QueryNetWorkCluster(query string) (*NetWorkClusterInfo, error) {
	args := c.Called(query)
	return args.Get(0).(*NetWorkClusterInfo), args.Error(1)
}
