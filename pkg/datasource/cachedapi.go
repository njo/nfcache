package datasource

import (
	"context"
	"sync"
	"time"

	"github.com/njo/nfcache/pkg/apiclient"
	"go.uber.org/zap"
)

const DefaultFetchTimeout = 20 * time.Second

type ApiData struct {
	lastUpdated time.Time
	data        []byte
}

type CachedAPI struct {
	client apiclient.ApiClient // Client is threadsafe
	log    *zap.SugaredLogger

	cachedData map[string]*ApiData // Not theadsafe, coordinate with rwMutex
	lock       *sync.RWMutex

	// Pieces to coordinate the updater goroutine
	done    chan struct{}
	wg      *sync.WaitGroup
	running bool
}

func NewCachedAPI(client apiclient.ApiClient, logger *zap.SugaredLogger) *CachedAPI {
	provider := &CachedAPI{
		client: client,
		log:    logger,

		cachedData: make(map[string]*ApiData),
		lock:       &sync.RWMutex{},

		done:    make(chan struct{}),
		wg:      &sync.WaitGroup{},
		running: false,
	}
	return provider
}

func (c *CachedAPI) dataUpdater(updateInterval time.Duration) {
	c.wg.Add(1)
	defer c.wg.Done()
	ticker := time.NewTicker(updateInterval)
	for {
		select {
		case <-c.done:
			c.log.Debug("dataUpdater worker exited")
			return
		case <-ticker.C:
			c.lock.RLock() // We don't need to hold this long since we update in goroutines
			for path := range c.cachedData {
				go c.updateEndpoint(path, DefaultFetchTimeout)
			}
			c.lock.RUnlock()
		}
	}
}

func (c *CachedAPI) updateEndpoint(path string, timeout time.Duration) error {
	c.wg.Add(1)
	defer c.wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	pageData, err := c.client.FetchAll(ctx, path)
	if err != nil {
		c.log.Error("Issue fetching %s: %v", path, err)
		return err
	}

	c.lock.Lock()
	// Clients receiving data from the old buffer will be able to complete the read before GC cleans up.
	c.cachedData[path] = &ApiData{time.Now().UTC(), pageData}
	c.lock.Unlock()
	return nil
}

func (c *CachedAPI) Run(updateInterval time.Duration) {
	go c.dataUpdater(updateInterval)
	c.running = true
}

func (c *CachedAPI) Shutdown() {
	if !c.running {
		c.log.Warn("Tried to stop the data provider before it was started")
		return
	}
	c.done <- struct{}{}
	c.wg.Wait()
	c.running = false
	c.log.Info("Data provider stopped")
}

func (c *CachedAPI) WatchEndpoint(path string) error {
	if _, ok := c.cachedData[path]; !ok {
		return c.updateEndpoint(path, DefaultFetchTimeout)
	}
	return nil
}

func (c *CachedAPI) Fetch(ctx context.Context, path string) ([]byte, error) {
	cachedPage, ok := c.cachedData[path]
	if !ok {
		return c.client.Fetch(ctx, path)
	}
	c.lock.RLock()
	defer c.lock.RUnlock()
	return cachedPage.data, nil
}
