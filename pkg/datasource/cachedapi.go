package datasource

import (
	"context"
	"sync"
	"time"

	"github.com/njo/nfcache/pkg/apiclient"
	"go.uber.org/zap"
)

const DefaultFetchTimeoutSec = 20
const DefaultUpdateIntervalSec = 60

type ApiData struct {
	lastUpdated time.Time
	data        []byte
}

// API Cache that passes through requests on cache misses to underlying API.
// Will keep cached URLs up to date automatically if Run() is called.
// Safe for concurrent use.
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

// Runs in a thread to keep the cache up to date until stopped with c.Done.
// Cache updates are done in parallel their own goroutines.
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
			c.lock.RLock() // We don't hold this long since we update in goroutines
			for path := range c.cachedData {
				// Keep in mind if updateEndpoint() is changed to block this will deadlock
				go c.updateEndpoint(path, DefaultFetchTimeoutSec*time.Second)
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

	if len(pageData) == 0 {
		c.log.Infof("Empty body found on update for: %s", path)
		return nil
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	// Clients receiving data from the old buffer will be able to complete the read before GC cleans up.
	c.cachedData[path] = &ApiData{time.Now().UTC(), pageData}
	c.log.Debugf("Updated %s", path)
	return nil
}

// Run the auto updater in another thread. Non-Blocking.
func (c *CachedAPI) Run(updateInterval time.Duration) {
	go c.dataUpdater(updateInterval)
	c.running = true
}

// Gracefully stop all the threads managed by the cached api.
func (c *CachedAPI) Shutdown() {
	if !c.running {
		c.log.Warn("Tried to stop the data provider before it was started")
		return
	}
	c.done <- struct{}{}
	c.wg.Wait() // Blocks until all workers are finished
	c.running = false
	c.log.Info("Data provider stopped")
}

// If the endpoint isn't being cached, fetch the endpoint, cache it and auto-update.
func (c *CachedAPI) WatchEndpoint(path string) error {
	c.lock.RLock()
	_, ok := c.cachedData[path]
	c.lock.RUnlock()
	if ok {
		return nil // Already being watched
	}
	return c.updateEndpoint(path, DefaultFetchTimeoutSec*time.Second)
}

// Fetch the path from the cache if it's there, otherwise proxy directly from api.
func (c *CachedAPI) Fetch(ctx context.Context, path string) ([]byte, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	cachedPage, ok := c.cachedData[path]
	if !ok { // cache miss, direct fetch
		return c.client.Fetch(ctx, path)
	}
	return cachedPage.data, nil
}
