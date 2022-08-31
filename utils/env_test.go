package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) {
	os.Setenv("MY_ENV", "value")
	defer os.Unsetenv("MY_ENV")

	assert.Equal(t, "value", GetEnv("MY_ENV", "default"))
	assert.Equal(t, "default", GetEnv("MY_ENV_UNKNOWN", "default"))
}
