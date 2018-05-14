package arg

import (
	"net"
	"net/mail"
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
	p, err := NewParser(Config{}, dest)
	if err != nil {
		return err
	}
	var parts []string
	if len(cmdline) > 0 {
		parts = strings.Split(cmdline, " ")
	}
	return p.Parse(parts)
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
	setenv(t, "FOO", "bar")
	os.Args = []string{"example"}
	MustParse(&args)
	assert.Equal(t, "bar", args.Foo)
}

func TestEnvironmentVariableOverrideName(t *testing.T) {
	var args struct {
		Foo string `arg:"env:BAZ"`
	}
	setenv(t, "BAZ", "bar")
	os.Args = []string{"example"}
	MustParse(&args)
	assert.Equal(t, "bar", args.Foo)
}

func TestEnvironmentVariableOverrideArgument(t *testing.T) {
	var args struct {
		Foo string `arg:"env"`
	}
	setenv(t, "FOO", "bar")
	os.Args = []string{"example", "--foo", "baz"}
	MustParse(&args)
	assert.Equal(t, "baz", args.Foo)
}

func TestEnvironmentVariableError(t *testing.T) {
	var args struct {
		Foo int `arg:"env"`
	}
	setenv(t, "FOO", "bar")
	os.Args = []string{"example"}
	err := Parse(&args)
	assert.Error(t, err)
}

func TestEnvironmentVariableRequired(t *testing.T) {
	var args struct {
		Foo string `arg:"env,required"`
	}
	setenv(t, "FOO", "bar")
	os.Args = []string{"example"}
	MustParse(&args)
	assert.Equal(t, "bar", args.Foo)
}

func TestEnvironmentVariableSliceArgumentString(t *testing.T)  {
	var args struct {
		Foo []string `arg:"env"`
	}
	setenv(t, "FOO", "bar,\"baz, qux\"")
	MustParse(&args)
	assert.Equal(t, []string{"bar", "baz, qux"}, args.Foo)
}

func TestEnvironmentVariableSliceArgumentInteger(t *testing.T)  {
	var args struct {
		Foo []int `arg:"env"`
	}
	setenv(t, "FOO", "1,99")
	MustParse(&args)
	assert.Equal(t, []int{1, 99}, args.Foo)
}

func TestEnvironmentVariableSliceArgumentFloat(t *testing.T)  {
	var args struct {
		Foo []float32 `arg:"env"`
	}
	setenv(t, "FOO", "1.1,99.9")
	MustParse(&args)
	assert.Equal(t, []float32{1.1, 99.9}, args.Foo)
}

func TestEnvironmentVariableSliceArgumentBool(t *testing.T)  {
	var args struct {
		Foo []bool `arg:"env"`
	}
	setenv(t, "FOO", "true,false,0,1")
	MustParse(&args)
	assert.Equal(t, []bool{true, false, false, true}, args.Foo)
}

func TestEnvironmentVariableSliceArgumentWrongCsv(t *testing.T)  {
	var args struct {
		Foo []int `arg:"env"`
	}
	setenv(t, "FOO", "1,99\"")
	err := Parse(&args)
	assert.Error(t, err)
}

func TestEnvironmentVariableSliceArgumentWrongType(t *testing.T)  {
	var args struct {
		Foo []bool `arg:"env"`
	}
	setenv(t, "FOO", "one,two")
	err := Parse(&args)
	assert.Error(t, err)
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
		Foo *textUnmarshaler
	}
	err := parse("--foo abc", &args)
	require.NoError(t, err)
	assert.Equal(t, 3, args.Foo.val)
}

func TestRepeatedTextUnmarshaler(t *testing.T) {
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
