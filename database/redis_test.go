package database

import (
	"os"
	"testing"

	"github.com/go-redis/redis/v9"
	"github.com/stretchr/testify/assert"
)

func init() {
	os.Setenv("MOCK_REDIS", "true")
	defer os.Unsetenv("MOCK_REDIS")
}

func TestMockRedis(t *testing.T) {
	// Ensure we are using the mock redis
	redis := GetRedisDB()
	assert.Equal(t, true, redis.Mock)
}

func TestSet(t *testing.T) {
	k := "key"
	v := "v"
	err := GetRedisDB().Set(k, v, 0)
	assert.Equal(t, nil, err)
}

func TestGet(t *testing.T) {
	k := "key"
	v := "v"
	err := GetRedisDB().Set(k, v, 0)
	assert.Equal(t, nil, err)
	val, err := GetRedisDB().Get(k)
	assert.Equal(t, nil, err)
	assert.Equal(t, v, val)
}

func TestDel(t *testing.T) {
	k := "key"
	v := "v"
	err := GetRedisDB().Set(k, v, 0)
	assert.Equal(t, nil, err)
	count, err := GetRedisDB().Del(k)
	assert.Equal(t, nil, err)
	assert.Equal(t, int64(1), count)
}

func TestHset(t *testing.T) {
	h := "hashes"
	k := "key"
	v := "v"
	err := GetRedisDB().Hset(h, k, v)
	assert.Equal(t, nil, err)
}

func TestHget(t *testing.T) {
	h := "hashes"
	k := "key"
	v := "v"
	err := GetRedisDB().Hset(h, k, v)
	assert.Equal(t, nil, err)
	val, err := GetRedisDB().Hget(h, k)
	assert.Equal(t, nil, err)
	assert.Equal(t, v, val)
}

func TestHlen(t *testing.T) {
	h := "hashes"
	k := "key"
	v := "v"
	err := GetRedisDB().Hset(h, k, v)
	assert.Equal(t, nil, err)
	count, err := GetRedisDB().Hlen(h)
	assert.Equal(t, nil, err)
	assert.Equal(t, int64(1), count)
}

func TestHdel(t *testing.T) {
	h := "hashes"
	k := "key"
	v := "v"
	err := GetRedisDB().Hset(h, k, v)
	assert.Equal(t, nil, err)
	err = GetRedisDB().Hdel(h, k)
	assert.Equal(t, nil, err)
	_, err = GetRedisDB().Hget(h, k)
	assert.Equal(t, redis.Nil, err)
}

func TestHgetall(t *testing.T) {
	h := "hashes"
	k := "key"
	k2 := "key2"
	v := "v"
	v2 := "v2"
	err := GetRedisDB().Hset(h, k, v)
	assert.Equal(t, nil, err)
	err = GetRedisDB().Hset(h, k2, v2)
	assert.Equal(t, nil, err)
	vals, err := GetRedisDB().Hgetall(h)
	assert.Equal(t, nil, err)
	assert.Equal(t, 2, len(vals))
	for _, val := range vals {
		assert.Contains(t, []string{v, v2}, val)
	}
}
