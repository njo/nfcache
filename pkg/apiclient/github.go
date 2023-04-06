package apiclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	GithubApiURL      = "https://api.github.com/"
	UserAgent         = "nfcache/0.1" // Note: Github API won't pretty print without a browser UA
	DefaultTimeoutSec = 10
	MaxPageFollow     = 100
	PerPageDefault    = 100
)

// Simple interface to make api calls. Should move to a separate file if we create other clients.
type ApiClient interface {
	Fetch(context.Context, string) ([]byte, error)    // Standard api fetch
	FetchAll(context.Context, string) ([]byte, error) // Make API call & flatten paginated results
}

// Conforms to the api client interface. Can be used concurrently.
type GithubClient struct {
	apiKey string
	client *http.Client
}

func NewGithub(apiKey string) ApiClient {
	client := http.Client{Timeout: DefaultTimeoutSec * time.Second}
	return NewGithubWithHttpClient(apiKey, &client)
}

func NewGithubWithHttpClient(apiKey string, client *http.Client) ApiClient {
	return &GithubClient{apiKey, client}
}

func (g *GithubClient) createRequest(ctx context.Context, path string) (*http.Request, error) {
	fullUrl, err := url.JoinPath(GithubApiURL, path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullUrl, nil)
	if err != nil {
		return nil, err
	}

	if g.apiKey != "" {
		req.Header.Set("Authorization", "token "+g.apiKey)
	}
	req.Header.Set("User-Agent", UserAgent)

	return req, nil
}

func (g *GithubClient) Fetch(ctx context.Context, path string) ([]byte, error) {
	req, err := g.createRequest(ctx, path)
	if err != nil {
		return nil, err
	}

	res, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}

	return extractResponseBody(res)
}

func (g *GithubClient) FetchAll(ctx context.Context, path string) ([]byte, error) {
	req, err := g.createRequest(ctx, path)
	if err != nil {
		return nil, err
	}

	pageCount := 1
	setRequestPagination(req, PerPageDefault, pageCount)

	res, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := extractResponseBody(res)
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return body, nil
	}

	// We're only interested in continuing if it's a list
	// Bit of a hack here to early return if it's an object
	if body[0] == '{' {
		return body, nil
	}

	// A slice of json objects to build the final array with
	var accumulatedJson = make([]map[string]any, 0)

	// A slice of json objects for a single request
	var jsonData []map[string]any
	err = json.Unmarshal(body, &jsonData)
	if err != nil {
		return nil, err
	}

	// Unpack the slice from the first request into the accumulator
	accumulatedJson = append(accumulatedJson, jsonData...)

	// Keep adding to the accumulator
	for responseHasNext(res) && pageCount < MaxPageFollow {
		pageCount = pageCount + 1
		setRequestPagination(req, PerPageDefault, pageCount)
		res, err = g.client.Do(req)
		if err != nil {
			return nil, err
		}

		body, err := extractResponseBody(res)
		if err != nil {
			return nil, err
		}
		if body == nil {
			// No explicit error here, just break and send back what we have
			break
		}

		jsonData = nil
		err = json.Unmarshal(body, &jsonData)
		if err != nil {
			return nil, err
		}
		accumulatedJson = append(accumulatedJson, jsonData...)
	}

	return json.Marshal(accumulatedJson)
}

// Return the response body as a byte array.
func extractResponseBody(response *http.Response) ([]byte, error) {
	if response.Body != nil {
		defer response.Body.Close()
	}

	body, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		return nil, readErr
	}

	return body, nil
}

func responseHasNext(response *http.Response) bool {
	// Paginated results contain a link header with a "next" link
	//... <https://api.github.com/organizations/913567/members?per_page=100&page=2>; rel="next", ....
	// https://docs.github.com/en/rest/guides/using-pagination-in-the-rest-api?apiVersion=2022-11-28
	linkHeader := response.Header.Get("link")
	return strings.Contains(linkHeader, `rel="next"`)
}

func setRequestPagination(req *http.Request, perPage int, pageNum int) {
	q := req.URL.Query()
	q.Set("per_page", strconv.Itoa(perPage))
	q.Set("page", strconv.Itoa(pageNum))
	req.URL.RawQuery = q.Encode()
}
