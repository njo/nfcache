package apiclient

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	API_URL             = "https://api.github.com/"
	USER_AGENT          = "nfcache/0.1" // Note: Github API won't pretty print without a browser UA
	DEFAULT_TIMEOUT_SEC = 10
	MAX_PAGE_FOLLOW     = 100
	PER_PAGE_DEFAULT    = 100
)

// Note: could use option struct here
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
	client := http.Client{Timeout: DEFAULT_TIMEOUT_SEC * time.Second}
	return NewWithHttpClient(apiKey, &client)
}

func NewWithHttpClient(apiKey string, client *http.Client) ApiClient {
	return &GithubClient{apiKey, client}
}

func (g *GithubClient) createRequest(ctx context.Context, path string) (*http.Request, error) {
	fullUrl, err := url.JoinPath(API_URL, path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullUrl, nil)
	if err != nil {
		log.Fatal(err)
	}

	if g.apiKey != "" {
		req.Header.Set("Authorization", "token "+g.apiKey)
	}
	req.Header.Set("User-Agent", USER_AGENT)

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

	return extractResponseBody(res), nil
}

func (g *GithubClient) FetchAll(ctx context.Context, path string) ([]byte, error) {
	req, err := g.createRequest(ctx, path)
	if err != nil {
		return nil, err
	}

	pageCount := 1
	setRequestPagination(req, PER_PAGE_DEFAULT, pageCount)

	res, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}

	body := extractResponseBody(res)
	if body == nil {
		return body, nil
	}

	// We're only interested in continuing if it's a list
	// bit of a hack here to early return if it's an object
	if body[0] == '{' {
		return body, nil
	}

	// A slice of json objects to build the final array with
	var accumulatedJson = make([]map[string]any, 0)

	// A slice of json objects from each request
	var jsonData []map[string]any
	err = json.Unmarshal(body, &jsonData)
	if err != nil {
		return nil, err
	}

	// Unpack the slice from the first request into the accumulator
	accumulatedJson = append(accumulatedJson, jsonData...)

	for responseHasNext(res) && pageCount < MAX_PAGE_FOLLOW {
		pageCount = pageCount + 1
		setRequestPagination(req, PER_PAGE_DEFAULT, pageCount)
		res, err = g.client.Do(req)
		if err != nil {
			break // Return what we've accumulated rather than returning an error
			//todo: log here
		}

		body := extractResponseBody(res)
		if body == nil {
			break // Return what we've accumulated rather than returning an error
			// todo: log here
		}

		jsonData = nil
		err = json.Unmarshal(body, &jsonData)
		if err != nil {
			// todo: log here
		}
		accumulatedJson = append(accumulatedJson, jsonData...)
	}

	return json.Marshal(accumulatedJson)
}

// Return the response body as a byte array. Empty response returns nil.
func extractResponseBody(response *http.Response) []byte {
	if response.Body != nil {
		defer response.Body.Close()
	}

	body, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		return nil
	}

	return body
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
