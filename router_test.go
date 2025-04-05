package heimdall_test

import (
	"testing"

	"github.com/arthurdotwork/heimdall"
	"github.com/stretchr/testify/require"
)

func TestNewRouter(t *testing.T) {
	t.Parallel()

	t.Run("it should return an error if it can not parse the target URL", func(t *testing.T) {
		endpoints := []heimdall.EndpointConfig{{Target: "://invalid"}}

		_, err := heimdall.NewRouter(endpoints)
		require.Error(t, err)
	})

	t.Run("it should build the router", func(t *testing.T) {
		endpoints := []heimdall.EndpointConfig{{Path: "/", Target: "https://www.google.com/", Method: "GET"}}

		router, err := heimdall.NewRouter(endpoints)
		require.NoError(t, err)
		require.NotNil(t, router)
		require.NotEmpty(t, router.Routes)
		require.Len(t, router.Routes, 1)
		require.Len(t, router.Routes["/"], 1)
		require.Equal(t, "GET", router.Routes["/"]["GET"].Method)
		require.Equal(t, "https://www.google.com/", router.Routes["/"]["GET"].Target.String())
	})
}

func TestRouter_GetRoute(t *testing.T) {
	t.Parallel()

	endpoints := []heimdall.EndpointConfig{{Path: "/foo", Method: "GET"}}
	router, err := heimdall.NewRouter(endpoints)
	require.NoError(t, err)

	t.Run("it should return an error if it can not find the route by path", func(t *testing.T) {
		route, ok := router.GetRoute("/bar", "GET")
		require.False(t, ok)
		require.Nil(t, route)
	})

	t.Run("it should return an error if it can not find the route by method", func(t *testing.T) {
		route, ok := router.GetRoute("/foo", "POST")
		require.False(t, ok)
		require.Nil(t, route)
	})

	t.Run("it should return the route", func(t *testing.T) {
		route, ok := router.GetRoute("/foo", "GET")
		require.True(t, ok)
		require.NotNil(t, route)
		require.Equal(t, "/foo", route.OriginalPath)
		require.Equal(t, "GET", route.Method)
	})
}
