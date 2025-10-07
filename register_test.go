package arg

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegister(t *testing.T) {
	var args1 struct {
		CacheSize int `arg:"--foo-cache-size"`
	}
	var args2 struct {
		Something string `arg:"--something"`
	}

	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()
	os.Args = []string{"program", "--something=something", "--foo-cache-size=100"}

	Register(&args1)
	Parse(&args2)

	assert.Equal(t, 100, args1.CacheSize)
	assert.Equal(t, "something", args2.Something)

	registrations = nil
}

func TestRegisterMustParse(t *testing.T) {
	var args1 struct {
		CacheSize int `arg:"--foo-cache-size"`
	}
	var args2 struct {
		Something string `arg:"--something"`
	}

	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()
	os.Args = []string{"program", "--something=something", "--foo-cache-size=100"}

	Register(&args1)

	var exitCode int
	var stdout bytes.Buffer
	exit := func(code int) { exitCode = code }
	mustParse(Config{Out: &stdout, Exit: exit}, &args2)

	assert.Equal(t, 0, exitCode)
	assert.Equal(t, 0, stdout.Len())
	assert.Equal(t, 100, args1.CacheSize)
	assert.Equal(t, "something", args2.Something)

	registrations = nil
}
