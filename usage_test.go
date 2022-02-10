package arg

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type NameDotName struct {
	Head, Tail string
}

func (n *NameDotName) UnmarshalText(b []byte) error {
	s := string(b)
	pos := strings.Index(s, ".")
	if pos == -1 {
		return fmt.Errorf("missing period in %s", s)
	}
	n.Head = s[:pos]
	n.Tail = s[pos+1:]
	return nil
}

func (n *NameDotName) MarshalText() (text []byte, err error) {
	text = []byte(fmt.Sprintf("%s.%s", n.Head, n.Tail))
	return
}

func TestWriteUsage(t *testing.T) {
	expectedUsage := "Usage: example [--name NAME] [--value VALUE] [--verbose] [--dataset DATASET] [--optimize OPTIMIZE] [--ids IDS] [--values VALUES] [--workers WORKERS] [--testenv TESTENV] [--file FILE] INPUT [OUTPUT [OUTPUT ...]]"

	expectedHelp := `
Usage: example [--name NAME] [--value VALUE] [--verbose] [--dataset DATASET] [--optimize OPTIMIZE] [--ids IDS] [--values VALUES] [--workers WORKERS] [--testenv TESTENV] [--file FILE] INPUT [OUTPUT [OUTPUT ...]]

Positional arguments:
  INPUT
  OUTPUT                 list of outputs

Options:
  --name NAME            name to use [default: Foo Bar]
  --value VALUE          secret value [default: 42]
  --verbose, -v          verbosity level
  --dataset DATASET      dataset to use
  --optimize OPTIMIZE, -O OPTIMIZE
                         optimization level
  --ids IDS              Ids
  --values VALUES        Values [default: [3.14 42 256]]
  --workers WORKERS, -w WORKERS
                         number of workers to start [default: 10, env: WORKERS]
  --testenv TESTENV, -a TESTENV [env: TEST_ENV]
  --file FILE, -f FILE   File with mandatory extension [default: scratch.txt]
  --help, -h             display this help and exit
`

	var args struct {
		Input    string       `arg:"positional,required"`
		Output   []string     `arg:"positional" help:"list of outputs"`
		Name     string       `help:"name to use"`
		Value    int          `help:"secret value"`
		Verbose  bool         `arg:"-v" help:"verbosity level"`
		Dataset  string       `help:"dataset to use"`
		Optimize int          `arg:"-O" help:"optimization level"`
		Ids      []int64      `help:"Ids"`
		Values   []float64    `help:"Values"`
		Workers  int          `arg:"-w,env:WORKERS" help:"number of workers to start" default:"10"`
		TestEnv  string       `arg:"-a,env:TEST_ENV"`
		File     *NameDotName `arg:"-f" help:"File with mandatory extension"`
	}
	args.Name = "Foo Bar"
	args.Value = 42
	args.Values = []float64{3.14, 42, 256}
	args.File = &NameDotName{"scratch", "txt"}
	p, err := NewParser(Config{Program: "example"}, &args)
	require.NoError(t, err)

	os.Args[0] = "example"

	var help bytes.Buffer
	p.WriteHelp(&help)
	assert.Equal(t, expectedHelp[1:], help.String())

	var usage bytes.Buffer
	p.WriteUsage(&usage)
	assert.Equal(t, expectedUsage, strings.TrimSpace(usage.String()))
}

type MyEnum int

func (n *MyEnum) UnmarshalText(b []byte) error {
	return nil
}

func (n *MyEnum) MarshalText() ([]byte, error) {
	return nil, errors.New("There was a problem")
}

func TestUsageWithDefaults(t *testing.T) {
	expectedUsage := "Usage: example [--label LABEL] [--content CONTENT]"

	expectedHelp := `
Usage: example [--label LABEL] [--content CONTENT]

Options:
  --label LABEL [default: cat]
  --content CONTENT [default: dog]
  --help, -h             display this help and exit
`
	var args struct {
		Label   string
		Content string `default:"dog"`
	}
	args.Label = "cat"
	p, err := NewParser(Config{Program: "example"}, &args)
	require.NoError(t, err)

	args.Label = "should_ignore_this"

	var help bytes.Buffer
	p.WriteHelp(&help)
	assert.Equal(t, expectedHelp[1:], help.String())

	var usage bytes.Buffer
	p.WriteUsage(&usage)
	assert.Equal(t, expectedUsage, strings.TrimSpace(usage.String()))
}

func TestUsageCannotMarshalToString(t *testing.T) {
	var args struct {
		Name *MyEnum
	}
	v := MyEnum(42)
	args.Name = &v
	_, err := NewParser(Config{Program: "example"}, &args)
	assert.EqualError(t, err, `args.Name: error marshaling default value to string: There was a problem`)
}

func TestUsageLongPositionalWithHelp_legacyForm(t *testing.T) {
	expectedUsage := "Usage: example [VERYLONGPOSITIONALWITHHELP]"

	expectedHelp := `
Usage: example [VERYLONGPOSITIONALWITHHELP]

Positional arguments:
  VERYLONGPOSITIONALWITHHELP
                         this positional argument is very long but cannot include commas

Options:
  --help, -h             display this help and exit
`
	var args struct {
		VeryLongPositionalWithHelp string `arg:"positional,help:this positional argument is very long but cannot include commas"`
	}

	p, err := NewParser(Config{Program: "example"}, &args)
	require.NoError(t, err)

	var help bytes.Buffer
	p.WriteHelp(&help)
	assert.Equal(t, expectedHelp[1:], help.String())

	var usage bytes.Buffer
	p.WriteUsage(&usage)
	assert.Equal(t, expectedUsage, strings.TrimSpace(usage.String()))
}

func TestUsageLongPositionalWithHelp_newForm(t *testing.T) {
	expectedUsage := "Usage: example [VERYLONGPOSITIONALWITHHELP]"

	expectedHelp := `
Usage: example [VERYLONGPOSITIONALWITHHELP]

Positional arguments:
  VERYLONGPOSITIONALWITHHELP
                         this positional argument is very long, and includes: commas, colons etc

Options:
  --help, -h             display this help and exit
`
	var args struct {
		VeryLongPositionalWithHelp string `arg:"positional" help:"this positional argument is very long, and includes: commas, colons etc"`
	}

	p, err := NewParser(Config{Program: "example"}, &args)
	require.NoError(t, err)

	var help bytes.Buffer
	p.WriteHelp(&help)
	assert.Equal(t, expectedHelp[1:], help.String())

	var usage bytes.Buffer
	p.WriteUsage(&usage)
	assert.Equal(t, expectedUsage, strings.TrimSpace(usage.String()))
}

func TestUsageWithProgramName(t *testing.T) {
	expectedUsage := "Usage: myprogram"

	expectedHelp := `
Usage: myprogram

Options:
  --help, -h             display this help and exit
`
	config := Config{
		Program: "myprogram",
	}
	p, err := NewParser(config, &struct{}{})
	require.NoError(t, err)

	os.Args[0] = "example"

	var help bytes.Buffer
	p.WriteHelp(&help)
	assert.Equal(t, expectedHelp[1:], help.String())

	var usage bytes.Buffer
	p.WriteUsage(&usage)
	assert.Equal(t, expectedUsage, strings.TrimSpace(usage.String()))
}

type versioned struct{}

// Version returns the version for this program
func (versioned) Version() string {
	return "example 3.2.1"
}

func TestUsageWithVersion(t *testing.T) {
	expectedUsage := "example 3.2.1\nUsage: example"

	expectedHelp := `
example 3.2.1
Usage: example

Options:
  --help, -h             display this help and exit
  --version              display version and exit
`
	os.Args[0] = "example"
	p, err := NewParser(Config{}, &versioned{})
	require.NoError(t, err)

	var help bytes.Buffer
	p.WriteHelp(&help)
	assert.Equal(t, expectedHelp[1:], help.String())

	var usage bytes.Buffer
	p.WriteUsage(&usage)
	assert.Equal(t, expectedUsage, strings.TrimSpace(usage.String()))
}

type described struct{}

// Described returns the description for this program
func (described) Description() string {
	return "this program does this and that"
}

func TestUsageWithDescription(t *testing.T) {
	expectedUsage := "Usage: example"

	expectedHelp := `
this program does this and that
Usage: example

Options:
  --help, -h             display this help and exit
`
	os.Args[0] = "example"
	p, err := NewParser(Config{}, &described{})
	require.NoError(t, err)

	var help bytes.Buffer
	p.WriteHelp(&help)
	assert.Equal(t, expectedHelp[1:], help.String())

	var usage bytes.Buffer
	p.WriteUsage(&usage)
	assert.Equal(t, expectedUsage, strings.TrimSpace(usage.String()))
}

func TestUsageForRequiredPositionals(t *testing.T) {
	expectedUsage := "Usage: example REQUIRED1 REQUIRED2\n"
	var args struct {
		Required1 string `arg:"positional,required"`
		Required2 string `arg:"positional,required"`
	}

	p, err := NewParser(Config{Program: "example"}, &args)
	require.NoError(t, err)

	var usage bytes.Buffer
	p.WriteUsage(&usage)
	assert.Equal(t, expectedUsage, usage.String())
}

func TestUsageForMixedPositionals(t *testing.T) {
	expectedUsage := "Usage: example REQUIRED1 REQUIRED2 [OPTIONAL1 [OPTIONAL2]]\n"
	var args struct {
		Required1 string `arg:"positional,required"`
		Required2 string `arg:"positional,required"`
		Optional1 string `arg:"positional"`
		Optional2 string `arg:"positional"`
	}

	p, err := NewParser(Config{Program: "example"}, &args)
	require.NoError(t, err)

	var usage bytes.Buffer
	p.WriteUsage(&usage)
	assert.Equal(t, expectedUsage, usage.String())
}

func TestUsageForRepeatedPositionals(t *testing.T) {
	expectedUsage := "Usage: example REQUIRED1 REQUIRED2 REPEATED [REPEATED ...]\n"
	var args struct {
		Required1 string   `arg:"positional,required"`
		Required2 string   `arg:"positional,required"`
		Repeated  []string `arg:"positional,required"`
	}

	p, err := NewParser(Config{Program: "example"}, &args)
	require.NoError(t, err)

	var usage bytes.Buffer
	p.WriteUsage(&usage)
	assert.Equal(t, expectedUsage, usage.String())
}

func TestUsageForMixedAndRepeatedPositionals(t *testing.T) {
	expectedUsage := "Usage: example REQUIRED1 REQUIRED2 [OPTIONAL1 [OPTIONAL2 [REPEATED [REPEATED ...]]]]\n"
	var args struct {
		Required1 string   `arg:"positional,required"`
		Required2 string   `arg:"positional,required"`
		Optional1 string   `arg:"positional"`
		Optional2 string   `arg:"positional"`
		Repeated  []string `arg:"positional"`
	}

	p, err := NewParser(Config{Program: "example"}, &args)
	require.NoError(t, err)

	var usage bytes.Buffer
	p.WriteUsage(&usage)
	assert.Equal(t, expectedUsage, usage.String())
}

func TestRequiredMultiplePositionals(t *testing.T) {
	expectedUsage := "Usage: example REQUIREDMULTIPLE [REQUIREDMULTIPLE ...]\n"

	expectedHelp := `
Usage: example REQUIREDMULTIPLE [REQUIREDMULTIPLE ...]

Positional arguments:
  REQUIREDMULTIPLE       required multiple positional

Options:
  --help, -h             display this help and exit
`
	var args struct {
		RequiredMultiple []string `arg:"positional,required" help:"required multiple positional"`
	}

	p, err := NewParser(Config{Program: "example"}, &args)
	require.NoError(t, err)

	var help bytes.Buffer
	p.WriteHelp(&help)
	assert.Equal(t, expectedHelp[1:], help.String())

	var usage bytes.Buffer
	p.WriteUsage(&usage)
	assert.Equal(t, expectedUsage, usage.String())
}

func TestUsageWithNestedSubcommands(t *testing.T) {
	expectedUsage := "Usage: example child nested [--enable] OUTPUT"

	expectedHelp := `
Usage: example child nested [--enable] OUTPUT

Positional arguments:
  OUTPUT

Options:
  --enable

Global options:
  --values VALUES        Values
  --verbose, -v          verbosity level
  --help, -h             display this help and exit
`

	var args struct {
		Verbose bool `arg:"-v" help:"verbosity level"`
		Child   *struct {
			Values []float64 `help:"Values"`
			Nested *struct {
				Enable bool
				Output string `arg:"positional,required"`
			} `arg:"subcommand:nested"`
		} `arg:"subcommand:child"`
	}

	os.Args[0] = "example"
	p, err := NewParser(Config{}, &args)
	require.NoError(t, err)

	_ = p.Parse([]string{"child", "nested", "value"})

	var help bytes.Buffer
	p.WriteHelp(&help)
	assert.Equal(t, expectedHelp[1:], help.String())

	var help2 bytes.Buffer
	p.WriteHelpForSubcommand(&help2, "child", "nested")
	assert.Equal(t, expectedHelp[1:], help2.String())

	var usage bytes.Buffer
	p.WriteUsage(&usage)
	assert.Equal(t, expectedUsage, strings.TrimSpace(usage.String()))

	var usage2 bytes.Buffer
	p.WriteUsageForSubcommand(&usage2, "child", "nested")
	assert.Equal(t, expectedUsage, strings.TrimSpace(usage2.String()))
}

func TestNonexistentSubcommand(t *testing.T) {
	var args struct {
		sub *struct{} `arg:"subcommand"`
	}
	p, err := NewParser(Config{}, &args)
	require.NoError(t, err)

	var b bytes.Buffer

	err = p.WriteUsageForSubcommand(&b, "does_not_exist")
	assert.Error(t, err)

	err = p.WriteHelpForSubcommand(&b, "does_not_exist")
	assert.Error(t, err)

	err = p.FailSubcommand("something went wrong", "does_not_exist")
	assert.Error(t, err)

	err = p.WriteUsageForSubcommand(&b, "sub", "does_not_exist")
	assert.Error(t, err)

	err = p.WriteHelpForSubcommand(&b, "sub", "does_not_exist")
	assert.Error(t, err)

	err = p.FailSubcommand("something went wrong", "sub", "does_not_exist")
	assert.Error(t, err)
}

func TestUsageWithoutLongNames(t *testing.T) {
	expectedUsage := "Usage: example [-a PLACEHOLDER] -b SHORTONLY2"

	expectedHelp := `
Usage: example [-a PLACEHOLDER] -b SHORTONLY2

Options:
  -a PLACEHOLDER         some help [default: some val]
  -b SHORTONLY2          some help2
  --help, -h             display this help and exit
`
	var args struct {
		ShortOnly  string `arg:"-a,--" help:"some help" default:"some val" placeholder:"PLACEHOLDER"`
		ShortOnly2 string `arg:"-b,--,required" help:"some help2"`
	}
	p, err := NewParser(Config{Program: "example"}, &args)
	assert.NoError(t, err)

	var help bytes.Buffer
	p.WriteHelp(&help)
	assert.Equal(t, expectedHelp[1:], help.String())

	var usage bytes.Buffer
	p.WriteUsage(&usage)
	assert.Equal(t, expectedUsage, strings.TrimSpace(usage.String()))
}

func TestUsageWithShortFirst(t *testing.T) {
	expectedUsage := "Usage: example [-c CAT] [--dog DOG]"

	expectedHelp := `
Usage: example [-c CAT] [--dog DOG]

Options:
  -c CAT
  --dog DOG
  --help, -h             display this help and exit
`
	var args struct {
		Dog string
		Cat string `arg:"-c,--"`
	}
	p, err := NewParser(Config{Program: "example"}, &args)
	assert.NoError(t, err)

	var help bytes.Buffer
	p.WriteHelp(&help)
	assert.Equal(t, expectedHelp[1:], help.String())

	var usage bytes.Buffer
	p.WriteUsage(&usage)
	assert.Equal(t, expectedUsage, strings.TrimSpace(usage.String()))
}

func TestUsageWithEnvOptions(t *testing.T) {
	expectedUsage := "Usage: example [-s SHORT]"

	expectedHelp := `
Usage: example [-s SHORT]

Options:
  -s SHORT [env: SHORT]
  --help, -h             display this help and exit
`
	var args struct {
		Short            string `arg:"--,-s,env"`
		EnvOnly          string `arg:"--,env"`
		EnvOnlyOverriden string `arg:"--,env:CUSTOM"`
	}

	p, err := NewParser(Config{Program: "example"}, &args)
	assert.NoError(t, err)

	var help bytes.Buffer
	p.WriteHelp(&help)
	assert.Equal(t, expectedHelp[1:], help.String())

	var usage bytes.Buffer
	p.WriteUsage(&usage)
	assert.Equal(t, expectedUsage, strings.TrimSpace(usage.String()))
}

func TestFail(t *testing.T) {
	originalStderr := stderr
	originalExit := osExit
	defer func() {
		stderr = originalStderr
		osExit = originalExit
	}()

	var b bytes.Buffer
	stderr = &b

	var exitCode int
	osExit = func(code int) { exitCode = code }

	expectedStdout := `
Usage: example [--foo FOO]
error: something went wrong
`

	var args struct {
		Foo int
	}
	p, err := NewParser(Config{Program: "example"}, &args)
	require.NoError(t, err)
	p.Fail("something went wrong")

	assert.Equal(t, expectedStdout[1:], b.String())
	assert.Equal(t, -1, exitCode)
}

func TestFailSubcommand(t *testing.T) {
	originalStderr := stderr
	originalExit := osExit
	defer func() {
		stderr = originalStderr
		osExit = originalExit
	}()

	var b bytes.Buffer
	stderr = &b

	var exitCode int
	osExit = func(code int) { exitCode = code }

	expectedStdout := `
Usage: example sub
error: something went wrong
`

	var args struct {
		Sub *struct{} `arg:"subcommand"`
	}
	p, err := NewParser(Config{Program: "example"}, &args)
	require.NoError(t, err)

	err = p.FailSubcommand("something went wrong", "sub")
	require.NoError(t, err)

	assert.Equal(t, expectedStdout[1:], b.String())
	assert.Equal(t, -1, exitCode)
}
