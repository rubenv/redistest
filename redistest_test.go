package redistest_test

import (
	"testing"

	"github.com/gomodule/redigo/redis"
	"github.com/rubenv/redistest"
	"github.com/stretchr/testify/assert"
)

func TestRedis(t *testing.T) {
	assert := assert.New(t)

	red, err := redistest.Start()
	assert.NoError(err)
	assert.NotNil(red)

	conn := red.Pool.Get()
	assert.NotNil(conn)

	_, err = conn.Do("SET", "foo", "bar")
	assert.NoError(err)

	s, err := redis.String(conn.Do("GET", "foo"))
	assert.NoError(err)
	assert.Equal(s, "bar")

	err = red.Stop()
	assert.NoError(err)
}
