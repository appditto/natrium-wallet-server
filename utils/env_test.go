package utils

import (
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	os.Setenv("MY_ENV", "value")
	defer os.Unsetenv("MY_ENV")

	AssertEqual(t, "value", GetEnv("MY_ENV", "default"))
	AssertEqual(t, "default", GetEnv("MY_ENV_UNKNOWN", "default"))
}
