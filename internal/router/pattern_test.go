package router

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsePattern(t *testing.T) {
	t.Parallel()

	t.Run("parses simple static path", func(t *testing.T) {
		p, err := parsePattern("/users/new")
		require.NoError(t, err)
		require.NotNil(t, p)
		require.Len(t, p.segments, 2)
		require.Equal(t, "users", p.segments[0].value)
		require.False(t, p.segments[0].isParam)
		require.Equal(t, "new", p.segments[1].value)
		require.False(t, p.segments[1].isParam)
	})

	t.Run("parses path with single parameter", func(t *testing.T) {
		p, err := parsePattern("/users/:id")
		require.NoError(t, err)
		require.NotNil(t, p)
		require.Len(t, p.segments, 2)
		require.Equal(t, "users", p.segments[0].value)
		require.False(t, p.segments[0].isParam)
		require.Equal(t, "id", p.segments[1].value)
		require.True(t, p.segments[1].isParam)
	})

	t.Run("parses path with multiple parameters", func(t *testing.T) {
		p, err := parsePattern("/users/:id/posts/:postId")
		require.NoError(t, err)
		require.NotNil(t, p)
		require.Len(t, p.segments, 4)
		require.Equal(t, "users", p.segments[0].value)
		require.False(t, p.segments[0].isParam)
		require.Equal(t, "id", p.segments[1].value)
		require.True(t, p.segments[1].isParam)
		require.Equal(t, "posts", p.segments[2].value)
		require.False(t, p.segments[2].isParam)
		require.Equal(t, "postId", p.segments[3].value)
		require.True(t, p.segments[3].isParam)
	})

	t.Run("parses root path", func(t *testing.T) {
		p, err := parsePattern("/")
		require.NoError(t, err)
		require.NotNil(t, p)
		require.Len(t, p.segments, 1)
		require.Equal(t, "", p.segments[0].value)
		require.False(t, p.segments[0].isParam)
	})

	t.Run("parses path with parameter at root", func(t *testing.T) {
		p, err := parsePattern("/:version")
		require.NoError(t, err)
		require.NotNil(t, p)
		require.Len(t, p.segments, 1)
		require.Equal(t, "version", p.segments[0].value)
		require.True(t, p.segments[0].isParam)
	})
}

func TestValidatePattern(t *testing.T) {
	t.Parallel()

	t.Run("rejects duplicate parameter names", func(t *testing.T) {
		p, err := parsePattern("/users/:id/posts/:id")
		require.NoError(t, err)
		err = p.validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "duplicate")
	})

	t.Run("rejects wildcard patterns", func(t *testing.T) {
		p, err := parsePattern("/api/*")
		require.NoError(t, err)
		err = p.validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "*")
	})

	t.Run("rejects empty parameter names", func(t *testing.T) {
		p, err := parsePattern("/users/:")
		require.NoError(t, err)
		err = p.validate()
		require.Error(t, err)
	})

	t.Run("accepts valid pattern", func(t *testing.T) {
		p, err := parsePattern("/users/:id")
		require.NoError(t, err)
		err = p.validate()
		require.NoError(t, err)
	})

	t.Run("accepts valid pattern with multiple params", func(t *testing.T) {
		p, err := parsePattern("/users/:id/posts/:postId")
		require.NoError(t, err)
		err = p.validate()
		require.NoError(t, err)
	})
}

func TestMatchPath(t *testing.T) {
	t.Parallel()

	t.Run("matches exact static path", func(t *testing.T) {
		p, err := parsePattern("/users/new")
		require.NoError(t, err)
		params, ok := p.match("/users/new")
		require.True(t, ok)
		require.Empty(t, params)
	})

	t.Run("matches path with single parameter", func(t *testing.T) {
		p, err := parsePattern("/users/:id")
		require.NoError(t, err)
		params, ok := p.match("/users/42")
		require.True(t, ok)
		require.Equal(t, map[string]string{"id": "42"}, params)
	})

	t.Run("matches path with multiple parameters", func(t *testing.T) {
		p, err := parsePattern("/users/:id/posts/:postId")
		require.NoError(t, err)
		params, ok := p.match("/users/42/posts/7")
		require.True(t, ok)
		require.Equal(t, map[string]string{"id": "42", "postId": "7"}, params)
	})

	t.Run("rejects path with empty parameter value", func(t *testing.T) {
		p, err := parsePattern("/users/:id")
		require.NoError(t, err)
		params, ok := p.match("/users/")
		require.False(t, ok)
		require.Nil(t, params)
	})

	t.Run("rejects path with mismatched segment count", func(t *testing.T) {
		p, err := parsePattern("/users/:id")
		require.NoError(t, err)
		params, ok := p.match("/users/42/extra")
		require.False(t, ok)
		require.Nil(t, params)
	})

	t.Run("rejects path with mismatched static segment", func(t *testing.T) {
		p, err := parsePattern("/users/new")
		require.NoError(t, err)
		params, ok := p.match("/users/old")
		require.False(t, ok)
		require.Nil(t, params)
	})

	t.Run("matches root path", func(t *testing.T) {
		p, err := parsePattern("/")
		require.NoError(t, err)
		params, ok := p.match("/")
		require.True(t, ok)
		require.Empty(t, params)
	})
}

func TestSpecificity(t *testing.T) {
	t.Parallel()

	t.Run("exact segment has higher specificity than parameter", func(t *testing.T) {
		exact, err := parsePattern("/users/new")
		require.NoError(t, err)
		param, err := parsePattern("/users/:id")
		require.NoError(t, err)
		require.Greater(t, exact.specificity(), param.specificity())
	})

	t.Run("more exact segments have higher specificity", func(t *testing.T) {
		p1, err := parsePattern("/a/b/c")
		require.NoError(t, err)
		p2, err := parsePattern("/a/:b/c")
		require.NoError(t, err)
		p3, err := parsePattern("/a/:b/:c")
		require.NoError(t, err)
		require.Greater(t, p1.specificity(), p2.specificity())
		require.Greater(t, p2.specificity(), p3.specificity())
	})

	t.Run("specificity calculation is correct", func(t *testing.T) {
		// /users/:id = 2 (exact) + 1 (param) = 3
		p, err := parsePattern("/users/:id")
		require.NoError(t, err)
		require.Equal(t, 3, p.specificity())
	})

	t.Run("specificity for all exact segments", func(t *testing.T) {
		// /users/new = 2 + 2 = 4
		p, err := parsePattern("/users/new")
		require.NoError(t, err)
		require.Equal(t, 4, p.specificity())
	})

	t.Run("specificity for all parameters", func(t *testing.T) {
		// /:id/:postId = 1 + 1 = 2
		p, err := parsePattern("/:id/:postId")
		require.NoError(t, err)
		require.Equal(t, 2, p.specificity())
	})
}
