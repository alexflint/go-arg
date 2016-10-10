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
	}
	err := parse("--foo bar", &args)
	require.NoError(t, err)
	assert.Equal(t, "bar", args.Foo)
}

func TestInt(t *testing.T) {
	var args struct {
		Foo int
	}
	err := parse("--foo 7", &args)
	require.NoError(t, err)
	assert.EqualValues(t, 7, args.Foo)
}

func TestUint(t *testing.T) {
	var args struct {
		Foo uint
	}
	err := parse("--foo 7", &args)
	require.NoError(t, err)
	assert.EqualValues(t, 7, args.Foo)
}

func TestFloat(t *testing.T) {
	var args struct {
		Foo float32
	}
	err := parse("--foo 3.4", &args)
	require.NoError(t, err)
	assert.EqualValues(t, 3.4, args.Foo)
}

func TestDuration(t *testing.T) {
	var args struct {
		Foo time.Duration
	}
	err := parse("--foo 3ms", &args)
	require.NoError(t, err)
	assert.Equal(t, 3*time.Millisecond, args.Foo)
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
