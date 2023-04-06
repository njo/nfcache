package apiclient

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// Simple interface to make api calls.
type ApiClient interface {
	Fetch(context.Context, string) ([]byte, error)    // Standard api fetch
	FetchAll(context.Context, string) ([]byte, error) // Make API call & flatten paginated results
}

type ApiClientMock struct {
	mock.Mock
}

func (a *ApiClientMock) Fetch(c context.Context, s string) ([]byte, error) {
	args := a.Called(c, s)
	return args.Get(0).([]byte), args.Error(1)
}

func (a *ApiClientMock) FetchAll(c context.Context, s string) ([]byte, error) {
	args := a.Called(c, s)
	return args.Get(0).([]byte), args.Error(1)
}
