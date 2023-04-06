package apiserver

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func healthcheck(c *gin.Context) {
	c.String(http.StatusOK, "Ok")
}

func githubCachedFetch(s *ApiServer, path string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// todo: handle error
		jsonData, _ := s.githubCachedAPI.Fetch(c.Request.Context(), path)
		c.Data(http.StatusOK, gin.MIMEJSON, jsonData)
	}
}

func githubProxyRequest(s *ApiServer) gin.HandlerFunc {
	return func(c *gin.Context) {
		// todo: handle error
		jsonData, _ := s.githubCachedAPI.Fetch(c.Request.Context(), c.Request.URL.Path)
		c.Data(http.StatusOK, gin.MIMEJSON, jsonData)
	}
}

func viewBottomRepos(s *ApiServer) gin.HandlerFunc {
	return func(c *gin.Context) {
		numResults, err := strconv.Atoi(c.Param("num"))
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		sortAttribute := c.Param("sortAttribute")
		var attributes = map[string]GithubSortField{
			"forks": ForksField, "open_issues": IssuesField, "stars": StarsField, "last_updated": UpdatedField}
		if _, ok := attributes[sortAttribute]; !ok {
			// Invalid sort field, just 404 rather than try proxy the request.
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		jsonData, _ := s.githubCachedAPI.Fetch(c.Request.Context(), NetflixOrgReposURL)
		jsonData, _ = BottomNRepos(jsonData, attributes[sortAttribute], numResults)

		c.Data(http.StatusOK, gin.MIMEJSON, jsonData)
	}
}
