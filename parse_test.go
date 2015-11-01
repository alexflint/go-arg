package arg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parse(cmdline string, dest interface{}) error {
	p, err := NewParser(dest)
	if err != nil {
		return err
	}
	return p.Parse(strings.Split(cmdline, " "))
}

func TestStringSingle(t *testing.T) {
	var args struct {
		Foo string
	}
	err := parse("--foo bar", &args)
	require.NoError(t, err)
	assert.Equal(t, "bar", args.Foo)
}

func TestMixed(t *testing.T) {
	var args struct {
		Foo  string `arg:"-f"`
		Bar  int
		Baz  uint `arg:"positional"`
		Ham  bool
		Spam float32
	}
	args.Bar = 3
	err := parse("123 -spam=1.2 -ham -f xyz", &args)
	require.NoError(t, err)
	assert.Equal(t, "xyz", args.Foo)
	assert.Equal(t, 3, args.Bar)
	assert.Equal(t, uint(123), args.Baz)
	assert.Equal(t, true, args.Ham)
	assert.EqualValues(t, 1.2, args.Spam)
}

func TestRequired(t *testing.T) {
	var args struct {
		Foo string `arg:"required"`
	}
	err := parse("", &args)
	require.Error(t, err, "--foo is required")
}

func TestShortFlag(t *testing.T) {
	var args struct {
		Foo string `arg:"-f"`
	}

	err := parse("-f xyz", &args)
	require.NoError(t, err)
	assert.Equal(t, "xyz", args.Foo)

	err = parse("-foo xyz", &args)
	require.NoError(t, err)
	assert.Equal(t, "xyz", args.Foo)

	err = parse("--foo xyz", &args)
	require.NoError(t, err)
	assert.Equal(t, "xyz", args.Foo)
}

func TestCaseSensitive(t *testing.T) {
	var args struct {
		Lower bool `arg:"-v"`
		Upper bool `arg:"-V"`
	}

	err := parse("-v", &args)
	require.NoError(t, err)
	assert.True(t, args.Lower)
	assert.False(t, args.Upper)
}

func TestCaseSensitive2(t *testing.T) {
	var args struct {
		Lower bool `arg:"-v"`
		Upper bool `arg:"-V"`
	}

	err := parse("-V", &args)
	require.NoError(t, err)
	assert.False(t, args.Lower)
	assert.True(t, args.Upper)
}

func TestPositional(t *testing.T) {
	var args struct {
		Input  string `arg:"positional"`
		Output string `arg:"positional"`
	}
	err := parse("foo", &args)
	require.NoError(t, err)
	assert.Equal(t, "foo", args.Input)
	assert.Equal(t, "", args.Output)
}

func TestRequiredPositional(t *testing.T) {
	var args struct {
		Input  string `arg:"positional"`
		Output string `arg:"positional,required"`
	}
	err := parse("foo", &args)
	assert.Error(t, err)
}

func TestTooManyPositional(t *testing.T) {
	var args struct {
		Input  string `arg:"positional"`
		Output string `arg:"positional"`
	}
	err := parse("foo bar baz", &args)
	assert.Error(t, err)
}

func TestMultiple(t *testing.T) {
	var args struct {
		Foo []int
		Bar []string
	}
	err := parse("--foo 1 2 3 --bar x y z", &args)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, args.Foo)
	assert.Equal(t, []string{"x", "y", "z"}, args.Bar)
}
