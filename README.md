# Build

# Run
GITHUB_API_TOKEN env var or in .env file
-p flag to specify port to listen on

# Test

Omissions / Shortcuts
 - Urls are fetched sequentially on server boot & data refresh
 - View data should be cached as it's expensive to compute
 - Url fetch can take longer than update ticker
 - Add a FetchAndCache method to the data provider
 - Killing the service while the urls are being updated will wait till that's finished, should cancel / timeout
 - Threads for updating urls are unbound