# About
A service to proxy API requests and cache a subset of responses.

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

When the service starts it will sequentially fetch the preset cached endpoints before the http server becomes available. This may take upwards of 10 seconds.

# Test
Run tests from root with `go test ./...`

```
➜ go test ./...
?   	github.com/njo/nfcache/pkg/apiclient	[no test files]
?   	github.com/njo/nfcache	[no test files]
ok  	github.com/njo/nfcache/pkg/apiserver	0.601s
ok  	github.com/njo/nfcache/pkg/datasource	0.464s
```

# Design Overview

## Components
API Server is the http service which contains response handlers and the logic for custom views.

The Cached API datasource uses a pluggable API Client to make calls to an upstream API. Endpoints set to be watched are automatically updated on an interval in a separate thread. Other endpoints proxied through this datasource are not cached.

A github client is provided as the only API Client implementation.

## Design Decisions
The service was written with the idea that adding new upstream APIs should be straightforward. Each upstream API would likely require different cache settings so the intention is to create a new Cached API for each external API. This approach also allows us to keep the ingress handler logic simple and easily warm the cache before starting the service. 

Allowing the Cached API to update based on inbound requests & the cached version expiry time rather than using the background thread would be straight forward to add to the current implementation.

A generic cache called explicitly from the ingress handler would have been another viable design.

## Omissions
Things that were either skipped for time or just felt out of scope for the exercise.

 - No service metrics. We'd want ingress and egress counters & histograms at a minimum.
 - Urls are fetched sequentially on server boot to avoid coordinating thread completion.
 - The pre-sorted view data should be cached as it's expensive to parse and compute.
 - Use the stored date when updating the cache to request using If-Modified-Since (or Etag).
 - Killing the service will wait for urls being updated to finish. Should be cancelled sooner.
 - Threads for updating urls are unbound, should be a max in-flight setting.
 - Data provider could cache proxied requests without adding to the auto-update pool.
 - API Client doesn't handle upstream 4xx/5xx differently, just blindly proxies response.
 - API Client & Data provider should consider sending headers from requests and returning response headers/status codes.
 - API Client should provide an optional logger interface. Currently just bubbles up errors.
 - Tests for the Github Client & Cached API background fetcher.
