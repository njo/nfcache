# About
A service to proxy API requests and cache a subset.

API Server is the http service including handlers and custom views.

The cached API datasource uses a provided API Client to make calls to an upstream API. Endpoints set to watched are automatically updated on an interval in a separate thread. Other endpoints proxied through this datasource are not cached.

A github client is provided as the only API client.

# Build
1. Install go -> https://go.dev/doc/install (tested with v1.20.2)
2. Clone this repo
3. From the repo root run `go build .` which should produce a `nfcache` binary

# Run
Run the compiled binary with the optional -p flag to set the port
```
./nfcache -p 7101
```

`GITHUB_API_TOKEN` can be specified as an env var or in a .env file.
If the API token isn't set requests will still be made without it.

When the service starts it will sequentially fetch the preset cached endpoints before the http server becomes available.

# Test
Run tests from root with `go test ./...`

```
➜ go test ./...
?   	github.com/njo/nfcache/pkg/apiclient	[no test files]
?   	github.com/njo/nfcache	[no test files]
ok  	github.com/njo/nfcache/pkg/apiserver	0.601s
ok  	github.com/njo/nfcache/pkg/datasource	0.464s
```

# Considerations
Things that were either skipped for time or just felt out of scope given the assignment.

 - Urls are fetched sequentially on server boot to avoid coordinating thread completion.
 - View data results should be cached after the sort as it's expensive to compute.
 - Use the stored date when updating the cache to request using If-Modified-Since (or use Etag).
 - Killing the service will wait for urls being updated to finish.
 - Threads for updating urls are unbound, should be a max in-flight figure.
 - Data provider could cache proxied requests without adding to the auto-update pool.
 - API Client doesn't handle upstream 4xx/5xx differently, just blindly proxies response.
 - API Client & Data provider doesn't pass through status codes, headers etc.
 - API Client should provide a flexible logger interface. Currently just bubbles up errors.
 - Service metrics: endpoint counters & histograms at a minimum.
 - Tests for the Github Client & Cached API background fetcher