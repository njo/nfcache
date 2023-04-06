package datasource

import (
	"context"
	"testing"

	"github.com/njo/nfcache/pkg/apiclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func tLog(t *testing.T) *zap.SugaredLogger {
	return zaptest.NewLogger(t).Sugar()
}

func TestCachedApi(t *testing.T) {
	m := new(apiclient.ApiClientMock)
	c := context.Background()
	cache := NewCachedAPI(m, tLog(t))
	path := "/myendpoint"

	response1 := []byte(`["First Call"]`)
	response2 := []byte(`["Second Call"]`)
	response3 := []byte(`["Third Call"]`)

	// Test fetching something new calls the client Fetch
	m.On("Fetch", mock.Anything, path).Return(response1, nil).Once()
	r, e := cache.Fetch(c, path)
	m.AssertExpectations(t)
	assert.Equal(t, r, response1)
	assert.Nil(t, e)

	// Second call returns new data, not first call data
	m.On("Fetch", mock.Anything, path).Return(response2, nil).Once()
	r, e = cache.Fetch(c, path)
	m.AssertExpectations(t)
	assert.Equal(t, r, response2)
	assert.Nil(t, e)

	// Add url to be watched
	m.On("FetchAll", mock.Anything, path).Return(response3, nil).Once()
	e = cache.WatchEndpoint(path)
	m.AssertCalled(t, "FetchAll", mock.Anything, path)
	assert.Nil(t, e)

	// Final fetch returns cached version
	r, e = cache.Fetch(c, path)
	m.AssertNotCalled(t, "Fetch")
	assert.Equal(t, r, response3)
	assert.Nil(t, e)
}
