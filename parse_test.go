package arg

import (
	"os"
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

func TestInvalidShortFlag(t *testing.T) {
	var args struct {
		Foo string `arg:"-foo"`
	}
	err := parse("", &args)
	assert.Error(t, err)
}

func TestLongFlag(t *testing.T) {
	var args struct {
		Foo string `arg:"--abc"`
	}

	err := parse("-abc xyz", &args)
	require.NoError(t, err)
	assert.Equal(t, "xyz", args.Foo)

	err = parse("--abc xyz", &args)
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

func TestPositionalPointer(t *testing.T) {
	var args struct {
		Input  string    `arg:"positional"`
		Output []*string `arg:"positional"`
	}
	err := parse("foo bar baz", &args)
	require.NoError(t, err)
	assert.Equal(t, "foo", args.Input)
	bar := "bar"
	baz := "baz"
	assert.Equal(t, []*string{&bar, &baz}, args.Output)
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

func TestMultipleWithEq(t *testing.T) {
	var args struct {
		Foo []int
		Bar []string
	}
	err := parse("--foo 1 2 3 --bar=x", &args)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, args.Foo)
	assert.Equal(t, []string{"x"}, args.Bar)
}

func TestExemptField(t *testing.T) {
	var args struct {
		Foo string
		Bar interface{} `arg:"-"`
	}
	err := parse("--foo xyz", &args)
	require.NoError(t, err)
	assert.Equal(t, "xyz", args.Foo)
}

func TestUnknownField(t *testing.T) {
	var args struct {
		Foo string
	}
	err := parse("--bar xyz", &args)
	assert.Error(t, err)
}

func TestMissingRequired(t *testing.T) {
	var args struct {
		Foo string   `arg:"required"`
		X   []string `arg:"positional"`
	}
	err := parse("x", &args)
	assert.Error(t, err)
}

func TestMissingValue(t *testing.T) {
	var args struct {
		Foo string
	}
	err := parse("--foo", &args)
	assert.Error(t, err)
}

func TestInvalidInt(t *testing.T) {
	var args struct {
		Foo int
	}
	err := parse("--foo=xyz", &args)
	assert.Error(t, err)
}

func TestInvalidUint(t *testing.T) {
	var args struct {
		Foo uint
	}
	err := parse("--foo=xyz", &args)
	assert.Error(t, err)
}

func TestInvalidFloat(t *testing.T) {
	var args struct {
		Foo float64
	}
	err := parse("--foo xyz", &args)
	require.Error(t, err)
}

func TestInvalidBool(t *testing.T) {
	var args struct {
		Foo bool
	}
	err := parse("--foo=xyz", &args)
	require.Error(t, err)
}

func TestInvalidIntSlice(t *testing.T) {
	var args struct {
		Foo []int
	}
	err := parse("--foo 1 2 xyz", &args)
	require.Error(t, err)
}

func TestInvalidPositional(t *testing.T) {
	var args struct {
		Foo int `arg:"positional"`
	}
	err := parse("xyz", &args)
	require.Error(t, err)
}

func TestInvalidPositionalSlice(t *testing.T) {
	var args struct {
		Foo []int `arg:"positional"`
	}
	err := parse("1 2 xyz", &args)
	require.Error(t, err)
}

func TestNoMoreOptions(t *testing.T) {
	var args struct {
		Foo string
		Bar []string `arg:"positional"`
	}
	err := parse("abc -- --foo xyz", &args)
	require.NoError(t, err)
	assert.Equal(t, "", args.Foo)
	assert.Equal(t, []string{"abc", "--foo", "xyz"}, args.Bar)
}

func TestHelpFlag(t *testing.T) {
	var args struct {
		Foo string
		Bar interface{} `arg:"-"`
	}
	err := parse("--help", &args)
	assert.Equal(t, ErrHelp, err)
}

func TestPanicOnNonPointer(t *testing.T) {
	var args struct{}
	assert.Panics(t, func() {
		parse("", args)
	})
}

func TestPanicOnNonStruct(t *testing.T) {
	var args string
	assert.Panics(t, func() {
		parse("", &args)
	})
}

func TestUnsupportedType(t *testing.T) {
	var args struct {
		Foo interface{}
	}
	err := parse("--foo", &args)
	assert.Error(t, err)
}

func TestUnsupportedSliceElement(t *testing.T) {
	var args struct {
		Foo []interface{}
	}
	err := parse("--foo", &args)
	assert.Error(t, err)
}

func TestUnknownTag(t *testing.T) {
	var args struct {
		Foo string `arg:"this_is_not_valid"`
	}
	err := parse("--foo xyz", &args)
	assert.Error(t, err)
}

func TestParse(t *testing.T) {
	var args struct {
		Foo string
	}
	os.Args = []string{"example", "--foo", "bar"}
	err := Parse(&args)
	require.NoError(t, err)
	assert.Equal(t, "bar", args.Foo)
}

func TestParseError(t *testing.T) {
	var args struct {
		Foo string `arg:"this_is_not_valid"`
	}
	os.Args = []string{"example", "--bar"}
	err := Parse(&args)
	assert.Error(t, err)
}

func TestMustParse(t *testing.T) {
	var args struct {
		Foo string
	}
	os.Args = []string{"example", "--foo", "bar"}
	MustParse(&args)
	assert.Equal(t, "bar", args.Foo)
}
