package params_test

import (
	"context"
	"testing"

	"github.com/arthurdotwork/heimdall/internal/params"
	"github.com/stretchr/testify/require"
)

func TestWithParams(t *testing.T) {
	t.Parallel()

	t.Run("it should store and retrieve params", func(t *testing.T) {
		ctx := context.Background()
		p := map[string]string{"id": "42"}
		ctx = params.WithParams(ctx, p)

		result := params.GetParams(ctx)
		require.Equal(t, p, result)
	})
}

func TestGetParam(t *testing.T) {
	t.Parallel()

	t.Run("it should return the param value when set", func(t *testing.T) {
		ctx := context.Background()
		ctx = params.WithParams(ctx, map[string]string{"id": "42"})

		result := params.GetParam(ctx, "id")
		require.Equal(t, "42", result)
	})

	t.Run("it should return empty string when param is missing", func(t *testing.T) {
		ctx := context.Background()
		ctx = params.WithParams(ctx, map[string]string{"id": "42"})

		result := params.GetParam(ctx, "missing")
		require.Equal(t, "", result)
	})
}

func TestGetParamsNil(t *testing.T) {
	t.Parallel()

	t.Run("it should return nil when no params in context", func(t *testing.T) {
		ctx := context.Background()

		result := params.GetParams(ctx)
		require.Nil(t, result)
	})
}

func TestGetParams(t *testing.T) {
	t.Parallel()

	t.Run("it should return full map when set", func(t *testing.T) {
		ctx := context.Background()
		p := map[string]string{"id": "42", "name": "test"}
		ctx = params.WithParams(ctx, p)

		result := params.GetParams(ctx)
		require.Equal(t, p, result)
	})
}

func TestGetParamMissing(t *testing.T) {
	t.Parallel()

	t.Run("it should return empty string when params map has other keys", func(t *testing.T) {
		ctx := context.Background()
		ctx = params.WithParams(ctx, map[string]string{"id": "42", "name": "test"})

		result := params.GetParam(ctx, "missing")
		require.Equal(t, "", result)
	})
}
