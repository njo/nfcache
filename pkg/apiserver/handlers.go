package apiserver

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Simple health check
func healthcheck(c *gin.Context) {
	c.String(http.StatusOK, "Ok")
}

// Fetch the path from the github cached api.
// Note: currently no difference between this and the request proxy
func githubCachedFetch(s *ApiServer, path string) gin.HandlerFunc {
	return func(c *gin.Context) {
		jsonData, err := s.githubCachedAPI.Fetch(c.Request.Context(), path)
		if err != nil {
			s.log.Errorf("Fetch %s failed with: %v", path, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if len(jsonData) == 0 {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Data(http.StatusOK, gin.MIMEJSON, jsonData)
	}
}

// Fetch the path from the github cached api. This is expected to be a cache miss.
func githubProxyRequest(s *ApiServer) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		jsonData, err := s.githubCachedAPI.Fetch(c.Request.Context(), path)
		if err != nil {
			s.log.Errorf("Fetch %s failed with: %v", path, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if len(jsonData) == 0 {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Data(http.StatusOK, gin.MIMEJSON, jsonData)
	}
}

const ParamSortAttribute = "sortAttribute"
const ParamNum = "num"

// Custom view over the cached github repo data.
// Note: the underlying data is cached but the view is built each time.
func viewBottomRepos(s *ApiServer) gin.HandlerFunc {
	var attributes = map[string]GithubSortField{ // Map valid urls to their sort field
		"forks": ForksField, "open_issues": IssuesField, "stars": StarsField, "last_updated": UpdatedField}
	return func(c *gin.Context) {
		numResults, err := strconv.Atoi(c.Param(ParamNum))
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		sortAttribute := c.Param(ParamSortAttribute)
		if _, ok := attributes[sortAttribute]; !ok {
			// Invalid sort field, just 404 rather than try proxy the request.
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		jsonRepos, err := s.githubCachedAPI.Fetch(c.Request.Context(), ApiPathNetflixOrgRepos)
		if err != nil || len(jsonRepos) == 0 {
			s.log.Errorf("Fetch bottomRepo data failed with: %v", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		sortedJsonRepos, err := BottomNRepos(jsonRepos, attributes[sortAttribute], numResults)
		if err != nil {
			s.log.Errorf("BottomNRepos sort failed with: %v", err)
			s.log.Debug("full repoData:\n%v", jsonRepos)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.Data(http.StatusOK, gin.MIMEJSON, sortedJsonRepos)
	}
}
