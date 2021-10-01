package arg

import (
	"bytes"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setenv(t *testing.T, name, val string) {
	if err := os.Setenv(name, val); err != nil {
		t.Error(err)
	}
}

func parse(cmdline string, dest interface{}) error {
	_, err := pparse(cmdline, dest)
	return err
}

func pparse(cmdline string, dest interface{}) (*Parser, error) {
	return parseWithEnv(cmdline, nil, dest)
}

func parseWithEnv(cmdline string, env []string, dest interface{}) (*Parser, error) {
	p, err := NewParser(Config{}, dest)
	if err != nil {
		return nil, err
	}

	// split the command line
	var parts []string
	if len(cmdline) > 0 {
		parts = strings.Split(cmdline, " ")
	}

	// split the environment vars
	for _, s := range env {
		pos := strings.Index(s, "=")
		if pos == -1 {
			return nil, fmt.Errorf("missing equals sign in %q", s)
		}
		err := os.Setenv(s[:pos], s[pos+1:])
		if err != nil {
			return nil, err
		}
	}

	// execute the parser
	return p, p.Parse(parts)
}

func TestString(t *testing.T) {
	var args struct {
		Foo string
		Ptr *string
	}
	err := parse("--foo bar --ptr baz", &args)
	require.NoError(t, err)
	assert.Equal(t, "bar", args.Foo)
	assert.Equal(t, "baz", *args.Ptr)
}

func TestBool(t *testing.T) {
	var args struct {
		A bool
		B bool
		C *bool
		D *bool
	}
	err := parse("--a --c", &args)
	require.NoError(t, err)
	assert.True(t, args.A)
	assert.False(t, args.B)
	assert.True(t, *args.C)
	assert.Nil(t, args.D)
}

func TestInt(t *testing.T) {
	var args struct {
		Foo int
		Ptr *int
	}
	err := parse("--foo 7 --ptr 8", &args)
	require.NoError(t, err)
	assert.EqualValues(t, 7, args.Foo)
	assert.EqualValues(t, 8, *args.Ptr)
}

func TestNegativeInt(t *testing.T) {
	var args struct {
		Foo int
	}
	err := parse("-foo -100", &args)
	require.NoError(t, err)
	assert.EqualValues(t, args.Foo, -100)
}

func TestNegativeIntAndFloatAndTricks(t *testing.T) {
	var args struct {
		Foo int
		Bar float64
		N   int `arg:"--100"`
	}
	err := parse("-foo -100 -bar -60.14 -100 -100", &args)
	require.NoError(t, err)
	assert.EqualValues(t, args.Foo, -100)
	assert.EqualValues(t, args.Bar, -60.14)
	assert.EqualValues(t, args.N, -100)
}

func TestUint(t *testing.T) {
	var args struct {
		Foo uint
		Ptr *uint
	}
	err := parse("--foo 7 --ptr 8", &args)
	require.NoError(t, err)
	assert.EqualValues(t, 7, args.Foo)
	assert.EqualValues(t, 8, *args.Ptr)
}

func TestFloat(t *testing.T) {
	var args struct {
		Foo float32
		Ptr *float32
	}
	err := parse("--foo 3.4 --ptr 3.5", &args)
	require.NoError(t, err)
	assert.EqualValues(t, 3.4, args.Foo)
	assert.EqualValues(t, 3.5, *args.Ptr)
}

func TestDuration(t *testing.T) {
	var args struct {
		Foo time.Duration
		Ptr *time.Duration
	}
	err := parse("--foo 3ms --ptr 4ms", &args)
	require.NoError(t, err)
	assert.Equal(t, 3*time.Millisecond, args.Foo)
	assert.Equal(t, 4*time.Millisecond, *args.Ptr)
}

func TestInvalidDuration(t *testing.T) {
	var args struct {
		Foo time.Duration
	}
	err := parse("--foo xxx", &args)
	require.Error(t, err)
}

func TestIntPtr(t *testing.T) {
	var args struct {
		Foo *int
	}
	err := parse("--foo 123", &args)
	require.NoError(t, err)
	require.NotNil(t, args.Foo)
	assert.Equal(t, 123, *args.Foo)
}

func TestIntPtrNotPresent(t *testing.T) {
	var args struct {
		Foo *int
	}
	err := parse("", &args)
	require.NoError(t, err)
	assert.Nil(t, args.Foo)
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

func TestRequiredWithEnv(t *testing.T) {
	var args struct {
		Foo string `arg:"required,env:FOO"`
	}
	err := parse("", &args)
	require.Error(t, err, "--foo is required (or environment variable FOO)")
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

func TestSlice(t *testing.T) {
	var args struct {
		Strings []string
	}
	err := parse("--strings a b c", &args)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, args.Strings)
}
func TestSliceOfBools(t *testing.T) {
	var args struct {
		B []bool
	}

	err := parse("--b true false true", &args)
	require.NoError(t, err)
	assert.Equal(t, []bool{true, false, true}, args.B)
}

func TestMap(t *testing.T) {
	var args struct {
		Values map[string]int
	}
	err := parse("--values a=1 b=2 c=3", &args)
	require.NoError(t, err)
	assert.Len(t, args.Values, 3)
	assert.Equal(t, 1, args.Values["a"])
	assert.Equal(t, 2, args.Values["b"])
	assert.Equal(t, 3, args.Values["c"])
}

func TestMapPositional(t *testing.T) {
	var args struct {
		Values map[string]int `arg:"positional"`
	}
	err := parse("a=1 b=2 c=3", &args)
	require.NoError(t, err)
	assert.Len(t, args.Values, 3)
	assert.Equal(t, 1, args.Values["a"])
	assert.Equal(t, 2, args.Values["b"])
	assert.Equal(t, 3, args.Values["c"])
}

func TestMapWithSeparate(t *testing.T) {
	var args struct {
		Values map[string]int `arg:"separate"`
	}
	err := parse("--values a=1 --values b=2 --values c=3", &args)
	require.NoError(t, err)
	assert.Len(t, args.Values, 3)
	assert.Equal(t, 1, args.Values["a"])
	assert.Equal(t, 2, args.Values["b"])
	assert.Equal(t, 3, args.Values["c"])
}

func TestPlaceholder(t *testing.T) {
	var args struct {
		Input    string   `arg:"positional" placeholder:"SRC"`
		Output   []string `arg:"positional" placeholder:"DST"`
		Optimize int      `arg:"-O" placeholder:"LEVEL"`
		MaxJobs  int      `arg:"-j" placeholder:"N"`
	}
	err := parse("-O 5 --maxjobs 2 src dest1 dest2", &args)
	assert.NoError(t, err)
}

func TestNoLongName(t *testing.T) {
	var args struct {
		ShortOnly string `arg:"-s,--"`
		EnvOnly   string `arg:"--,env"`
	}
	setenv(t, "ENVONLY", "TestVal")
	err := parse("-s TestVal2", &args)
	assert.NoError(t, err)
	assert.Equal(t, "TestVal", args.EnvOnly)
	assert.Equal(t, "TestVal2", args.ShortOnly)
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

func TestRequiredPositionalMultiple(t *testing.T) {
	var args struct {
		Input    string   `arg:"positional"`
		Multiple []string `arg:"positional,required"`
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

func TestMultiplePositionals(t *testing.T) {
	var args struct {
		Input    string   `arg:"positional"`
		Multiple []string `arg:"positional,required"`
	}
	err := parse("foo a b c", &args)
	assert.NoError(t, err)
	assert.Equal(t, "foo", args.Input)
	assert.Equal(t, []string{"a", "b", "c"}, args.Multiple)
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

func TestMultipleWithDefault(t *testing.T) {
	var args struct {
		Foo []int
		Bar []string
	}
	args.Foo = []int{42}
	args.Bar = []string{"foo"}
	err := parse("--foo 1 2 3 --bar x y z", &args)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, args.Foo)
	assert.Equal(t, []string{"x", "y", "z"}, args.Bar)
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

func TestNonsenseKey(t *testing.T) {
	var args struct {
		X []string `arg:"positional, nonsense"`
	}
	err := parse("x", &args)
	assert.Error(t, err)
}

func TestMissingValueAtEnd(t *testing.T) {
	var args struct {
		Foo string
	}
	err := parse("--foo", &args)
	assert.Error(t, err)
}

func TestMissingValueInMiddle(t *testing.T) {
	var args struct {
		Foo string
		Bar string
	}
	err := parse("--foo --bar=abc", &args)
	assert.Error(t, err)
}

func TestNegativeValue(t *testing.T) {
	var args struct {
		Foo int
	}
	err := parse("--foo -123", &args)
	require.NoError(t, err)
	assert.Equal(t, -123, args.Foo)
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

func TestNoMoreOptionsBeforeHelp(t *testing.T) {
	var args struct {
		Foo int
	}
	err := parse("not_an_integer -- --help", &args)
	assert.NotEqual(t, ErrHelp, err)
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
		_ = parse("", args)
	})
}

func TestErrorOnNonStruct(t *testing.T) {
	var args string
	err := parse("", &args)
	assert.Error(t, err)
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
	err := parse("--foo 3", &args)
	assert.Error(t, err)
}

func TestUnsupportedSliceElementMissingValue(t *testing.T) {
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
	parser := MustParse(&args)
	assert.Equal(t, "bar", args.Foo)
	assert.NotNil(t, parser)
}

func TestEnvironmentVariable(t *testing.T) {
	var args struct {
		Foo string `arg:"env"`
	}
	_, err := parseWithEnv("", []string{"FOO=bar"}, &args)
	require.NoError(t, err)
	assert.Equal(t, "bar", args.Foo)
}

func TestEnvironmentVariableNotPresent(t *testing.T) {
	var args struct {
		NotPresent string `arg:"env"`
	}
	_, err := parseWithEnv("", nil, &args)
	require.NoError(t, err)
	assert.Equal(t, "", args.NotPresent)
}

func TestEnvironmentVariableOverrideName(t *testing.T) {
	var args struct {
		Foo string `arg:"env:BAZ"`
	}
	_, err := parseWithEnv("", []string{"BAZ=bar"}, &args)
	require.NoError(t, err)
	assert.Equal(t, "bar", args.Foo)
}

func TestEnvironmentVariableOverrideArgument(t *testing.T) {
	var args struct {
		Foo string `arg:"env"`
	}
	_, err := parseWithEnv("--foo zzz", []string{"FOO=bar"}, &args)
	require.NoError(t, err)
	assert.Equal(t, "zzz", args.Foo)
}

func TestEnvironmentVariableError(t *testing.T) {
	var args struct {
		Foo int `arg:"env"`
	}
	_, err := parseWithEnv("", []string{"FOO=bar"}, &args)
	assert.Error(t, err)
}

func TestEnvironmentVariableRequired(t *testing.T) {
	var args struct {
		Foo string `arg:"env,required"`
	}
	_, err := parseWithEnv("", []string{"FOO=bar"}, &args)
	require.NoError(t, err)
	assert.Equal(t, "bar", args.Foo)
}

func TestEnvironmentVariableSliceArgumentString(t *testing.T) {
	var args struct {
		Foo []string `arg:"env"`
	}
	_, err := parseWithEnv("", []string{`FOO=bar,"baz, qux"`}, &args)
	require.NoError(t, err)
	assert.Equal(t, []string{"bar", "baz, qux"}, args.Foo)
}

func TestEnvironmentVariableSliceEmpty(t *testing.T) {
	var args struct {
		Foo []string `arg:"env"`
	}
	_, err := parseWithEnv("", []string{`FOO=`}, &args)
	require.NoError(t, err)
	assert.Len(t, args.Foo, 0)
}

func TestEnvironmentVariableSliceArgumentInteger(t *testing.T) {
	var args struct {
		Foo []int `arg:"env"`
	}
	_, err := parseWithEnv("", []string{`FOO=1,99`}, &args)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 99}, args.Foo)
}

func TestEnvironmentVariableSliceArgumentFloat(t *testing.T) {
	var args struct {
		Foo []float32 `arg:"env"`
	}
	_, err := parseWithEnv("", []string{`FOO=1.1,99.9`}, &args)
	require.NoError(t, err)
	assert.Equal(t, []float32{1.1, 99.9}, args.Foo)
}

func TestEnvironmentVariableSliceArgumentBool(t *testing.T) {
	var args struct {
		Foo []bool `arg:"env"`
	}
	_, err := parseWithEnv("", []string{`FOO=true,false,0,1`}, &args)
	require.NoError(t, err)
	assert.Equal(t, []bool{true, false, false, true}, args.Foo)
}

func TestEnvironmentVariableSliceArgumentWrongCsv(t *testing.T) {
	var args struct {
		Foo []int `arg:"env"`
	}
	_, err := parseWithEnv("", []string{`FOO=1,99\"`}, &args)
	assert.Error(t, err)
}

func TestEnvironmentVariableSliceArgumentWrongType(t *testing.T) {
	var args struct {
		Foo []bool `arg:"env"`
	}
	_, err := parseWithEnv("", []string{`FOO=one,two`}, &args)
	assert.Error(t, err)
}

func TestEnvironmentVariableMap(t *testing.T) {
	var args struct {
		Foo map[int]string `arg:"env"`
	}
	_, err := parseWithEnv("", []string{`FOO=1=one,99=ninetynine`}, &args)
	require.NoError(t, err)
	assert.Len(t, args.Foo, 2)
	assert.Equal(t, "one", args.Foo[1])
	assert.Equal(t, "ninetynine", args.Foo[99])
}

func TestEnvironmentVariableEmptyMap(t *testing.T) {
	var args struct {
		Foo map[int]string `arg:"env"`
	}
	_, err := parseWithEnv("", []string{`FOO=`}, &args)
	require.NoError(t, err)
	assert.Len(t, args.Foo, 0)
}

func TestEnvironmentVariableIgnored(t *testing.T) {
	var args struct {
		Foo string `arg:"env"`
	}
	setenv(t, "FOO", "abc")

	p, err := NewParser(Config{IgnoreEnv: true}, &args)
	require.NoError(t, err)

	err = p.Parse(nil)
	assert.NoError(t, err)
	assert.Equal(t, "", args.Foo)
}

func TestEnvironmentVariableInSubcommandIgnored(t *testing.T) {
	var args struct {
		Sub *struct {
			Foo string `arg:"env"`
		} `arg:"subcommand"`
	}
	setenv(t, "FOO", "abc")

	p, err := NewParser(Config{IgnoreEnv: true}, &args)
	require.NoError(t, err)

	err = p.Parse([]string{"sub"})
	assert.NoError(t, err)
	assert.Equal(t, "", args.Sub.Foo)
}

type textUnmarshaler struct {
	val int
}

func (f *textUnmarshaler) UnmarshalText(b []byte) error {
	f.val = len(b)
	return nil
}

func TestTextUnmarshaler(t *testing.T) {
	// fields that implement TextUnmarshaler should be parsed using that interface
	var args struct {
		Foo textUnmarshaler
	}
	err := parse("--foo abc", &args)
	require.NoError(t, err)
	assert.Equal(t, 3, args.Foo.val)
}

func TestPtrToTextUnmarshaler(t *testing.T) {
	// fields that implement TextUnmarshaler should be parsed using that interface
	var args struct {
		Foo *textUnmarshaler
	}
	err := parse("--foo abc", &args)
	require.NoError(t, err)
	assert.Equal(t, 3, args.Foo.val)
}

func TestRepeatedTextUnmarshaler(t *testing.T) {
	// fields that implement TextUnmarshaler should be parsed using that interface
	var args struct {
		Foo []textUnmarshaler
	}
	err := parse("--foo abc d ef", &args)
	require.NoError(t, err)
	require.Len(t, args.Foo, 3)
	assert.Equal(t, 3, args.Foo[0].val)
	assert.Equal(t, 1, args.Foo[1].val)
	assert.Equal(t, 2, args.Foo[2].val)
}

func TestRepeatedPtrToTextUnmarshaler(t *testing.T) {
	// fields that implement TextUnmarshaler should be parsed using that interface
	var args struct {
		Foo []*textUnmarshaler
	}
	err := parse("--foo abc d ef", &args)
	require.NoError(t, err)
	require.Len(t, args.Foo, 3)
	assert.Equal(t, 3, args.Foo[0].val)
	assert.Equal(t, 1, args.Foo[1].val)
	assert.Equal(t, 2, args.Foo[2].val)
}

func TestPositionalTextUnmarshaler(t *testing.T) {
	// fields that implement TextUnmarshaler should be parsed using that interface
	var args struct {
		Foo []textUnmarshaler `arg:"positional"`
	}
	err := parse("abc d ef", &args)
	require.NoError(t, err)
	require.Len(t, args.Foo, 3)
	assert.Equal(t, 3, args.Foo[0].val)
	assert.Equal(t, 1, args.Foo[1].val)
	assert.Equal(t, 2, args.Foo[2].val)
}

func TestPositionalPtrToTextUnmarshaler(t *testing.T) {
	// fields that implement TextUnmarshaler should be parsed using that interface
	var args struct {
		Foo []*textUnmarshaler `arg:"positional"`
	}
	err := parse("abc d ef", &args)
	require.NoError(t, err)
	require.Len(t, args.Foo, 3)
	assert.Equal(t, 3, args.Foo[0].val)
	assert.Equal(t, 1, args.Foo[1].val)
	assert.Equal(t, 2, args.Foo[2].val)
}

type boolUnmarshaler bool

func (p *boolUnmarshaler) UnmarshalText(b []byte) error {
	*p = len(b)%2 == 0
	return nil
}

func TestBoolUnmarhsaler(t *testing.T) {
	// test that a bool type that implements TextUnmarshaler is
	// handled as a TextUnmarshaler not as a bool
	var args struct {
		Foo *boolUnmarshaler
	}
	err := parse("--foo ab", &args)
	require.NoError(t, err)
	assert.EqualValues(t, true, *args.Foo)
}

type sliceUnmarshaler []int

func (p *sliceUnmarshaler) UnmarshalText(b []byte) error {
	*p = sliceUnmarshaler{len(b)}
	return nil
}

func TestSliceUnmarhsaler(t *testing.T) {
	// test that a slice type that implements TextUnmarshaler is
	// handled as a TextUnmarshaler not as a slice
	var args struct {
		Foo *sliceUnmarshaler
		Bar string `arg:"positional"`
	}
	err := parse("--foo abcde xyz", &args)
	require.NoError(t, err)
	require.Len(t, *args.Foo, 1)
	assert.EqualValues(t, 5, (*args.Foo)[0])
	assert.Equal(t, "xyz", args.Bar)
}

func TestIP(t *testing.T) {
	var args struct {
		Host net.IP
	}
	err := parse("--host 192.168.0.1", &args)
	require.NoError(t, err)
	assert.Equal(t, "192.168.0.1", args.Host.String())
}

func TestPtrToIP(t *testing.T) {
	var args struct {
		Host *net.IP
	}
	err := parse("--host 192.168.0.1", &args)
	require.NoError(t, err)
	assert.Equal(t, "192.168.0.1", args.Host.String())
}

func TestURL(t *testing.T) {
	var args struct {
		URL url.URL
	}
	err := parse("--url https://example.com/get?item=xyz", &args)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/get?item=xyz", args.URL.String())
}

func TestPtrToURL(t *testing.T) {
	var args struct {
		URL *url.URL
	}
	err := parse("--url http://example.com/#xyz", &args)
	require.NoError(t, err)
	assert.Equal(t, "http://example.com/#xyz", args.URL.String())
}

func TestIPSlice(t *testing.T) {
	var args struct {
		Host []net.IP
	}
	err := parse("--host 192.168.0.1 127.0.0.1", &args)
	require.NoError(t, err)
	require.Len(t, args.Host, 2)
	assert.Equal(t, "192.168.0.1", args.Host[0].String())
	assert.Equal(t, "127.0.0.1", args.Host[1].String())
}

func TestInvalidIPAddress(t *testing.T) {
	var args struct {
		Host net.IP
	}
	err := parse("--host xxx", &args)
	assert.Error(t, err)
}

func TestMAC(t *testing.T) {
	var args struct {
		Host net.HardwareAddr
	}
	err := parse("--host 0123.4567.89ab", &args)
	require.NoError(t, err)
	assert.Equal(t, "01:23:45:67:89:ab", args.Host.String())
}

func TestInvalidMac(t *testing.T) {
	var args struct {
		Host net.HardwareAddr
	}
	err := parse("--host xxx", &args)
	assert.Error(t, err)
}

func TestMailAddr(t *testing.T) {
	var args struct {
		Recipient mail.Address
	}
	err := parse("--recipient foo@example.com", &args)
	require.NoError(t, err)
	assert.Equal(t, "<foo@example.com>", args.Recipient.String())
}

func TestInvalidMailAddr(t *testing.T) {
	var args struct {
		Recipient mail.Address
	}
	err := parse("--recipient xxx", &args)
	assert.Error(t, err)
}

type A struct {
	X string
}

type B struct {
	Y int
}

func TestEmbedded(t *testing.T) {
	var args struct {
		A
		B
		Z bool
	}
	err := parse("--x=hello --y=321 --z", &args)
	require.NoError(t, err)
	assert.Equal(t, "hello", args.X)
	assert.Equal(t, 321, args.Y)
	assert.Equal(t, true, args.Z)
}

func TestEmbeddedPtr(t *testing.T) {
	// embedded pointer fields are not supported so this should return an error
	var args struct {
		*A
	}
	err := parse("--x=hello", &args)
	require.Error(t, err)
}

func TestEmbeddedPtrIgnored(t *testing.T) {
	// embedded pointer fields are not normally supported but here
	// we explicitly exclude it so the non-nil embedded structs
	// should work as expected
	var args struct {
		*A `arg:"-"`
		B
	}
	err := parse("--y=321", &args)
	require.NoError(t, err)
	assert.Equal(t, 321, args.Y)
}

func TestEmbeddedWithDuplicateField(t *testing.T) {
	// see https://github.com/alexflint/go-arg/issues/100
	type T struct {
		A string `arg:"--cat"`
	}
	type U struct {
		A string `arg:"--dog"`
	}
	var args struct {
		T
		U
	}

	err := parse("--cat=cat --dog=dog", &args)
	require.NoError(t, err)
	assert.Equal(t, "cat", args.T.A)
	assert.Equal(t, "dog", args.U.A)
}

func TestEmbeddedWithDuplicateField2(t *testing.T) {
	// see https://github.com/alexflint/go-arg/issues/100
	type T struct {
		A string
	}
	type U struct {
		A string
	}
	var args struct {
		T
		U
	}

	err := parse("--a=xyz", &args)
	require.NoError(t, err)
	assert.Equal(t, "xyz", args.T.A)
	assert.Equal(t, "", args.U.A)
}

func TestUnexportedEmbedded(t *testing.T) {
	type embeddedArgs struct {
		Foo string
	}
	var args struct {
		embeddedArgs
	}
	err := parse("--foo bar", &args)
	require.NoError(t, err)
	assert.Equal(t, "bar", args.Foo)
}

func TestIgnoredEmbedded(t *testing.T) {
	type embeddedArgs struct {
		Foo string
	}
	var args struct {
		embeddedArgs `arg:"-"`
	}
	err := parse("--foo bar", &args)
	require.Error(t, err)
}

func TestEmptyArgs(t *testing.T) {
	origArgs := os.Args

	// test what happens if somehow os.Args is empty
	os.Args = nil
	var args struct {
		Foo string
	}
	MustParse(&args)

	// put the original arguments back
	os.Args = origArgs
}

func TestTooManyHyphens(t *testing.T) {
	var args struct {
		TooManyHyphens string `arg:"---x"`
	}
	err := parse("--foo -", &args)
	assert.Error(t, err)
}

func TestHyphenAsOption(t *testing.T) {
	var args struct {
		Foo string
	}
	err := parse("--foo -", &args)
	require.NoError(t, err)
	assert.Equal(t, "-", args.Foo)
}

func TestHyphenAsPositional(t *testing.T) {
	var args struct {
		Foo string `arg:"positional"`
	}
	err := parse("-", &args)
	require.NoError(t, err)
	assert.Equal(t, "-", args.Foo)
}

func TestHyphenInMultiOption(t *testing.T) {
	var args struct {
		Foo []string
		Bar int
	}
	err := parse("--foo --- x - y --bar 3", &args)
	require.NoError(t, err)
	assert.Equal(t, []string{"---", "x", "-", "y"}, args.Foo)
	assert.Equal(t, 3, args.Bar)
}

func TestHyphenInMultiPositional(t *testing.T) {
	var args struct {
		Foo []string `arg:"positional"`
	}
	err := parse("--- x - y", &args)
	require.NoError(t, err)
	assert.Equal(t, []string{"---", "x", "-", "y"}, args.Foo)
}

func TestSeparate(t *testing.T) {
	for _, val := range []string{"-f one", "-f=one", "--foo one", "--foo=one"} {
		var args struct {
			Foo []string `arg:"--foo,-f,separate"`
		}

		err := parse(val, &args)
		require.NoError(t, err)
		assert.Equal(t, []string{"one"}, args.Foo)
	}
}

func TestSeparateWithDefault(t *testing.T) {
	args := struct {
		Foo []string `arg:"--foo,-f,separate"`
	}{
		Foo: []string{"default"},
	}

	err := parse("-f one -f=two", &args)
	require.NoError(t, err)
	assert.Equal(t, []string{"default", "one", "two"}, args.Foo)
}

func TestSeparateWithPositional(t *testing.T) {
	var args struct {
		Foo []string `arg:"--foo,-f,separate"`
		Bar string   `arg:"positional"`
		Moo string   `arg:"positional"`
	}

	err := parse("zzz --foo one -f=two --foo=three -f four aaa", &args)
	require.NoError(t, err)
	assert.Equal(t, []string{"one", "two", "three", "four"}, args.Foo)
	assert.Equal(t, "zzz", args.Bar)
	assert.Equal(t, "aaa", args.Moo)
}

func TestSeparatePositionalInterweaved(t *testing.T) {
	var args struct {
		Foo  []string `arg:"--foo,-f,separate"`
		Bar  []string `arg:"--bar,-b,separate"`
		Pre  string   `arg:"positional"`
		Post []string `arg:"positional"`
	}

	err := parse("zzz -f foo1 -b=bar1 --foo=foo2 -b bar2 post1 -b bar3 post2 post3", &args)
	require.NoError(t, err)
	assert.Equal(t, []string{"foo1", "foo2"}, args.Foo)
	assert.Equal(t, []string{"bar1", "bar2", "bar3"}, args.Bar)
	assert.Equal(t, "zzz", args.Pre)
	assert.Equal(t, []string{"post1", "post2", "post3"}, args.Post)
}

func TestSpacesAllowedInTags(t *testing.T) {
	var args struct {
		Foo []string `arg:"--foo, -f, separate, required, help:quite nice really"`
	}

	err := parse("--foo one -f=two --foo=three -f four", &args)
	require.NoError(t, err)
	assert.Equal(t, []string{"one", "two", "three", "four"}, args.Foo)
}

func TestReuseParser(t *testing.T) {
	var args struct {
		Foo string `arg:"required"`
	}

	p, err := NewParser(Config{}, &args)
	require.NoError(t, err)

	err = p.Parse([]string{"--foo=abc"})
	require.NoError(t, err)
	assert.Equal(t, args.Foo, "abc")

	err = p.Parse([]string{})
	assert.Error(t, err)
}

func TestVersion(t *testing.T) {
	var args struct{}
	err := parse("--version", &args)
	assert.Equal(t, ErrVersion, err)

}

func TestMultipleTerminates(t *testing.T) {
	var args struct {
		X []string
		Y string `arg:"positional"`
	}

	err := parse("--x a b -- c", &args)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, args.X)
	assert.Equal(t, "c", args.Y)
}

func TestDefaultOptionValues(t *testing.T) {
	var args struct {
		A int      `default:"123"`
		B *int     `default:"123"`
		C string   `default:"abc"`
		D *string  `default:"abc"`
		E float64  `default:"1.23"`
		F *float64 `default:"1.23"`
		G bool     `default:"true"`
		H *bool    `default:"true"`
	}

	err := parse("--c=xyz --e=4.56", &args)
	require.NoError(t, err)

	assert.Equal(t, 123, args.A)
	assert.Equal(t, 123, *args.B)
	assert.Equal(t, "xyz", args.C)
	assert.Equal(t, "abc", *args.D)
	assert.Equal(t, 4.56, args.E)
	assert.Equal(t, 1.23, *args.F)
	assert.True(t, args.G)
	assert.True(t, args.G)
}

func TestDefaultUnparseable(t *testing.T) {
	var args struct {
		A int `default:"x"`
	}

	err := parse("", &args)
	assert.EqualError(t, err, `error processing default value for --a: strconv.ParseInt: parsing "x": invalid syntax`)
}

func TestDefaultPositionalValues(t *testing.T) {
	var args struct {
		A int      `arg:"positional" default:"123"`
		B *int     `arg:"positional" default:"123"`
		C string   `arg:"positional" default:"abc"`
		D *string  `arg:"positional" default:"abc"`
		E float64  `arg:"positional" default:"1.23"`
		F *float64 `arg:"positional" default:"1.23"`
		G bool     `arg:"positional" default:"true"`
		H *bool    `arg:"positional" default:"true"`
	}

	err := parse("456 789", &args)
	require.NoError(t, err)

	assert.Equal(t, 456, args.A)
	assert.Equal(t, 789, *args.B)
	assert.Equal(t, "abc", args.C)
	assert.Equal(t, "abc", *args.D)
	assert.Equal(t, 1.23, args.E)
	assert.Equal(t, 1.23, *args.F)
	assert.True(t, args.G)
	assert.True(t, args.G)
}

func TestDefaultValuesNotAllowedWithRequired(t *testing.T) {
	var args struct {
		A int `arg:"required" default:"123"` // required not allowed with default!
	}

	err := parse("", &args)
	assert.EqualError(t, err, ".A: 'required' cannot be used when a default value is specified")
}

func TestDefaultValuesNotAllowedWithSlice(t *testing.T) {
	var args struct {
		A []int `default:"123"` // required not allowed with default!
	}

	err := parse("", &args)
	assert.EqualError(t, err, ".A: default values are not supported for slice or map fields")
}

func TestUnexportedFieldsSkipped(t *testing.T) {
	var args struct {
		unexported struct{}
	}

	_, err := NewParser(Config{}, &args)
	require.NoError(t, err)
}

func TestMustParseInvalidParser(t *testing.T) {
	originalExit := osExit
	originalStdout := stdout
	defer func() {
		osExit = originalExit
		stdout = originalStdout
	}()

	var exitCode int
	osExit = func(code int) { exitCode = code }
	stdout = &bytes.Buffer{}

	var args struct {
		CannotParse struct{}
	}
	parser := MustParse(&args)
	assert.Nil(t, parser)
	assert.Equal(t, -1, exitCode)
}

func TestMustParsePrintsHelp(t *testing.T) {
	originalExit := osExit
	originalStdout := stdout
	originalArgs := os.Args
	defer func() {
		osExit = originalExit
		stdout = originalStdout
		os.Args = originalArgs
	}()

	var exitCode *int
	osExit = func(code int) { exitCode = &code }
	os.Args = []string{"someprogram", "--help"}
	stdout = &bytes.Buffer{}

	var args struct{}
	parser := MustParse(&args)
	assert.NotNil(t, parser)
	require.NotNil(t, exitCode)
	assert.Equal(t, 0, *exitCode)
}

func TestMustParsePrintsVersion(t *testing.T) {
	originalExit := osExit
	originalStdout := stdout
	originalArgs := os.Args
	defer func() {
		osExit = originalExit
		stdout = originalStdout
		os.Args = originalArgs
	}()

	var exitCode *int
	osExit = func(code int) { exitCode = &code }
	os.Args = []string{"someprogram", "--version"}

	var b bytes.Buffer
	stdout = &b

	var args versioned
	parser := MustParse(&args)
	require.NotNil(t, parser)
	require.NotNil(t, exitCode)
	assert.Equal(t, 0, *exitCode)
	assert.Equal(t, "example 3.2.1\n", b.String())
}
