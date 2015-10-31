package arguments

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func split(s string) []string {
	return strings.Split(s, " ")
}

func TestStringSingle(t *testing.T) {
	var args struct {
		Foo string
	}
	err := ParseFrom(&args, split("--foo bar"))
	require.NoError(t, err)
	assert.Equal(t, "bar", args.Foo)
}

func TestMixed(t *testing.T) {
	var args struct {
		Foo  string `arg:"-f"`
		Bar  int
		Ham  bool
		Spam float32
	}
	args.Bar = 3
	err := ParseFrom(&args, split("-spam=1.2 -ham -f xyz"))
	require.NoError(t, err)
	assert.Equal(t, "xyz", args.Foo)
	assert.Equal(t, 3, args.Bar)
	assert.Equal(t, true, args.Ham)
	assert.Equal(t, 1.2, args.Spam)
}

func TestRequired(t *testing.T) {
	var args struct {
		Foo string `arg:"required"`
	}
	err := ParseFrom(&args, nil)
	require.Error(t, err, "--foo is required")
}

func TestShortFlag(t *testing.T) {
	var args struct {
		Foo string `arg:"-f"`
	}

	err := ParseFrom(&args, split("-f xyz"))
	require.NoError(t, err)
	assert.Equal(t, "xyz", args.Foo)

	err = ParseFrom(&args, split("-foo xyz"))
	require.NoError(t, err)
	assert.Equal(t, "xyz", args.Foo)

	err = ParseFrom(&args, split("--foo xyz"))
	require.NoError(t, err)
	assert.Equal(t, "xyz", args.Foo)
}

func TestCaseSensitive(t *testing.T) {
	var args struct {
		Lower bool `arg:"-v"`
		Upper bool `arg:"-V"`
	}

	err := ParseFrom(&args, split("-v"))
	require.NoError(t, err)
	assert.True(t, args.Lower)
	assert.False(t, args.Upper)
}

func TestCaseSensitive2(t *testing.T) {
	var args struct {
		Lower bool `arg:"-v"`
		Upper bool `arg:"-V"`
	}

	err := ParseFrom(&args, split("-V"))
	require.NoError(t, err)
	assert.False(t, args.Lower)
	assert.True(t, args.Upper)
}
