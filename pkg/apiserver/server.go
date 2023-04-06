package apiserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/njo/nfcache/pkg/datasource"
)

const (
	NetflixOrgURL        = "/orgs/Netflix"
	NetflixOrgMembersURL = NetflixOrgURL + "/members"
	NetflixOrgReposURL   = NetflixOrgURL + "/repos"
)

type ApiServer struct {
	githubCachedAPI *datasource.CachedAPI
	log             *zap.SugaredLogger
	httpServer      *http.Server
}

func CachedEndpoints() []string {
	return []string{"/", NetflixOrgURL, NetflixOrgMembersURL, NetflixOrgReposURL}
}

func New(githubCache *datasource.CachedAPI, logger *zap.SugaredLogger) *ApiServer {
	return &ApiServer{
		githubCachedAPI: githubCache,
		log:             logger,
		httpServer:      nil, // gets added when we start the server
	}
}

func (s *ApiServer) Run(address string) {
	s.httpServer = &http.Server{
		Addr:    address,
		Handler: s.bootstrapHandler(),
	}
	s.log.Infof("Listening on %s", address)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.log.Fatal("Problem starting http server", err)
	}
}

func (s *ApiServer) Shutdown(timeout time.Duration) {
	if s.httpServer == nil {
		s.log.Error("Called shutdown without a running http server")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.log.Fatal("Problem gracefully closing http server ", err)
	}
	s.log.Infof("HTTP server stopped")
}

func (s *ApiServer) bootstrapHandler() http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.GET("/healthcheck", healthcheck)

	// views look like: /view/bottom/10/forks
	r.GET(fmt.Sprintf("/view/bottom/:%s/:%s", ParamNum, ParamSortAttribute), viewBottomRepos(s))

	for _, url := range CachedEndpoints() {
		r.GET(url, githubCachedFetch(s, url))
	}
	r.NoRoute(githubProxyRequest(s)) // Proxy unknown urls instead of 404ing
	return r.Handler()
}
