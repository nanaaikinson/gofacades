package gofacades

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testStruct struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

// setupTestRedis creates a mock Redis server for testing
func setupTestRedis(t *testing.T) (*Client, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	p, err := strconv.Atoi(mr.Port())
	require.NoError(t, err)

	client, err := New(Config{
		Host: mr.Host(),
		Port: p,
	})
	require.NoError(t, err)

	return client, mr
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid configuration",
			cfg: Config{
				Host: "localhost",
				Port: 6379,
			},
			wantErr: false,
		},
		{
			name: "invalid port",
			cfg: Config{
				Host: "localhost",
				Port: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClient_Put(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()

	t.Run("store string with TTL", func(t *testing.T) {
		err := client.Put(ctx, "test-key", "test-value", time.Hour)
		assert.NoError(t, err)

		// Verify value was stored
		val, err := client.Get(ctx, "test-key")
		assert.NoError(t, err)
		assert.Equal(t, "test-value", val)

		// Verify TTL was set
		ttl := mr.TTL("test-key")
		assert.True(t, ttl > 0)
	})

	t.Run("overwrite existing key", func(t *testing.T) {
		// Set initial value
		err := client.Put(ctx, "test-key", "initial-value", time.Hour)
		assert.NoError(t, err)

		// Overwrite value
		err = client.Put(ctx, "test-key", "new-value", time.Hour)
		assert.NoError(t, err)

		// Verify new value
		val, err := client.Get(ctx, "test-key")
		assert.NoError(t, err)
		assert.Equal(t, "new-value", val)
	})
}

func TestClient_Get(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()

	t.Run("get existing key", func(t *testing.T) {
		err := client.Put(ctx, "test-key", "test-value", time.Hour)
		assert.NoError(t, err)

		val, err := client.Get(ctx, "test-key")
		assert.NoError(t, err)
		assert.Equal(t, "test-value", val)
	})

	t.Run("get non-existent key", func(t *testing.T) {
		val, err := client.Get(ctx, "non-existent-key")
		assert.Error(t, err)
		assert.Equal(t, ErrKeyNotFound, err)
		assert.Empty(t, val)
	})

	t.Run("get expired key", func(t *testing.T) {
		err := client.Put(ctx, "expired-key", "test-value", time.Millisecond*10)
		assert.NoError(t, err)

		// Wait for the key to expire
		time.Sleep(time.Millisecond * 50)

		// Attempt to get the key
		val, err := client.Get(ctx, "expired-key")
		assert.Error(t, err)
		assert.Equal(t, ErrKeyNotFound, err)
		assert.Empty(t, val)
	})
}

func TestClient_Has(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()

	t.Run("existing key", func(t *testing.T) {
		err := client.Put(ctx, "test-key", "test-value", time.Hour)
		assert.NoError(t, err)

		exists, err := client.Has(ctx, "test-key")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("non-existent key", func(t *testing.T) {
		exists, err := client.Has(ctx, "non-existent-key")
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestClient_Remember(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()

	t.Run("remember new value", func(t *testing.T) {
		callCount := 0
		callback := func() (interface{}, error) {
			callCount++
			return testStruct{Name: "test", Value: 123}, nil
		}

		// First call should execute callback
		val, err := client.Remember(ctx, "test-key", time.Hour, callback)
		assert.NoError(t, err)
		assert.Equal(t, 1, callCount)

		var result testStruct
		err = json.Unmarshal([]byte(val), &result)
		assert.NoError(t, err)
		assert.Equal(t, "test", result.Name)
		assert.Equal(t, 123, result.Value)

		// Second call should use cached value
		val, err = client.Remember(ctx, "test-key", time.Hour, callback)
		assert.NoError(t, err)
		assert.Equal(t, 1, callCount) // Callback should not be called again
	})

	t.Run("nil callback", func(t *testing.T) {
		_, err := client.Remember(ctx, "nil-callback", time.Hour, nil)
		assert.Error(t, err)
		assert.Equal(t, ErrNilCallback, err)
	})
}

func TestClient_Pull(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()

	t.Run("pull existing key", func(t *testing.T) {
		err := client.Put(ctx, "test-key", "test-value", time.Hour)
		assert.NoError(t, err)

		// Pull the value
		val, err := client.Pull(ctx, "test-key")
		assert.NoError(t, err)
		assert.Equal(t, "test-value", val)

		// Verify key was deleted
		exists, err := client.Has(ctx, "test-key")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("pull non-existent key", func(t *testing.T) {
		val, err := client.Pull(ctx, "non-existent-key")
		assert.Error(t, err)
		assert.Equal(t, ErrKeyNotFound, err)
		assert.Empty(t, val)
	})
}

func TestClient_Forever(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()

	t.Run("store forever", func(t *testing.T) {
		err := client.Forever(ctx, "test-key", "test-value")
		assert.NoError(t, err)

		// Verify value was stored
		val, err := client.Get(ctx, "test-key")
		assert.NoError(t, err)
		assert.Equal(t, "test-value", val)

		// Verify no TTL was set
		ttl := mr.TTL("test-key")
		assert.Equal(t, time.Duration(-1), ttl) // -1 indicates no TTL
	})
}

func TestClient_Forget(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()

	t.Run("forget existing key", func(t *testing.T) {
		err := client.Put(ctx, "test-key", "test-value", time.Hour)
		assert.NoError(t, err)

		err = client.Forget(ctx, "test-key")
		assert.NoError(t, err)

		exists, err := client.Has(ctx, "test-key")
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("forget non-existent key", func(t *testing.T) {
		err := client.Forget(ctx, "non-existent-key")
		assert.NoError(t, err) // Redis DEL command returns success even if key doesn't exist
	})
}

func TestClient_Flush(t *testing.T) {
	client, mr := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()

	t.Run("flush all keys", func(t *testing.T) {
		// Store multiple keys
		err := client.Put(ctx, "key1", "value1", time.Hour)
		assert.NoError(t, err)
		err = client.Put(ctx, "key2", "value2", time.Hour)
		assert.NoError(t, err)
		err = client.Forever(ctx, "key3", "value3")
		assert.NoError(t, err)

		// Flush all keys
		err = client.Flush(ctx)
		assert.NoError(t, err)

		// Verify all keys are gone
		exists1, _ := client.Has(ctx, "key1")
		exists2, _ := client.Has(ctx, "key2")
		exists3, _ := client.Has(ctx, "key3")
		assert.False(t, exists1)
		assert.False(t, exists2)
		assert.False(t, exists3)
	})
}
