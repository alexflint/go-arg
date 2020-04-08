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
	expectedUsage := "Usage: example [--name NAME] [--value VALUE] [--verbose] [--dataset DATASET] [--optimize OPTIMIZE] [--ids IDS] [--values VALUES] [--workers WORKERS] [--file FILE] INPUT [OUTPUT [OUTPUT ...]]\n"

	expectedHelp := `Usage: example [--name NAME] [--value VALUE] [--verbose] [--dataset DATASET] [--optimize OPTIMIZE] [--ids IDS] [--values VALUES] [--workers WORKERS] [--file FILE] INPUT [OUTPUT [OUTPUT ...]]

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
                         number of workers to start
  --file FILE, -f FILE   File with mandatory extension [default: scratch.txt]
  --help, -h             display this help and exit
`
	var args struct {
		Input    string       `arg:"positional"`
		Output   []string     `arg:"positional" help:"list of outputs"`
		Name     string       `help:"name to use"`
		Value    int          `help:"secret value"`
		Verbose  bool         `arg:"-v" help:"verbosity level"`
		Dataset  string       `help:"dataset to use"`
		Optimize int          `arg:"-O" help:"optimization level"`
		Ids      []int64      `help:"Ids"`
		Values   []float64    `help:"Values"`
		Workers  int          `arg:"-w,env:WORKERS" help:"number of workers to start"`
		File     *NameDotName `arg:"-f" help:"File with mandatory extension"`
	}
	args.Name = "Foo Bar"
	args.Value = 42
	args.Values = []float64{3.14, 42, 256}
	args.File = &NameDotName{"scratch", "txt"}
	p, err := NewParser(Config{Program: "example"}, &args)
	require.NoError(t, err)

	os.Args[0] = "example"

	var usage bytes.Buffer
	p.WriteUsage(&usage)
	assert.Equal(t, expectedUsage, usage.String())

	var help bytes.Buffer
	p.WriteHelp(&help)
	assert.Equal(t, expectedHelp, help.String())
}

type MyEnum int

func (n *MyEnum) UnmarshalText(b []byte) error {
	return nil
}

func (n *MyEnum) MarshalText() ([]byte, error) {
	return nil, errors.New("There was a problem")
}

func TestUsageWithDefaults(t *testing.T) {
	expectedHelp := `Usage: example [--label LABEL] [--content CONTENT]

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
	assert.Equal(t, expectedHelp, help.String())
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
	expectedHelp := `Usage: example VERYLONGPOSITIONALWITHHELP

Positional arguments:
  VERYLONGPOSITIONALWITHHELP
                         this positional argument is very long but cannot include commas

Options:
  --help, -h             display this help and exit
`
	var args struct {
		VeryLongPositionalWithHelp string `arg:"positional,help:this positional argument is very long but cannot include commas"`
	}

	p, err := NewParser(Config{}, &args)
	require.NoError(t, err)

	os.Args[0] = "example"
	var help bytes.Buffer
	p.WriteHelp(&help)
	assert.Equal(t, expectedHelp, help.String())
}

func TestUsageLongPositionalWithHelp_newForm(t *testing.T) {
	expectedHelp := `Usage: example VERYLONGPOSITIONALWITHHELP

Positional arguments:
  VERYLONGPOSITIONALWITHHELP
                         this positional argument is very long, and includes: commas, colons etc

Options:
  --help, -h             display this help and exit
`
	var args struct {
		VeryLongPositionalWithHelp string `arg:"positional" help:"this positional argument is very long, and includes: commas, colons etc"`
	}

	p, err := NewParser(Config{}, &args)
	require.NoError(t, err)

	os.Args[0] = "example"
	var help bytes.Buffer
	p.WriteHelp(&help)
	assert.Equal(t, expectedHelp, help.String())
}

func TestUsageWithProgramName(t *testing.T) {
	expectedHelp := `Usage: myprogram

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
	assert.Equal(t, expectedHelp, help.String())
}

type versioned struct{}

// Version returns the version for this program
func (versioned) Version() string {
	return "example 3.2.1"
}

func TestUsageWithVersion(t *testing.T) {
	expectedHelp := `example 3.2.1
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
	actual := help.String()
	if expectedHelp != actual {
		t.Logf("Expected:\n%s", expectedHelp)
		t.Logf("Actual:\n%s", actual)
		t.Fail()
	}
}

type described struct{}

// Described returns the description for this program
func (described) Description() string {
	return "this program does this and that"
}

func TestUsageWithDescription(t *testing.T) {
	expectedHelp := `this program does this and that
Usage: example

Options:
  --help, -h             display this help and exit
`
	os.Args[0] = "example"
	p, err := NewParser(Config{}, &described{})
	require.NoError(t, err)

	var help bytes.Buffer
	p.WriteHelp(&help)
	actual := help.String()
	if expectedHelp != actual {
		t.Logf("Expected:\n%s", expectedHelp)
		t.Logf("Actual:\n%s", actual)
		t.Fail()
	}
}

func TestRequiredMultiplePositionals(t *testing.T) {
	expectedHelp := `Usage: example [--makerequired MAKEREQUIRED] [--requiredvariable REQUIREDVARIABLE] REQUIREDMULTIPLE [REQUIREDMULTIPLE ...]

Required arguments:
  REQUIREDMULTIPLE       required multiple positional

Conditionally required arguments:
  REQUIREDVARIABLE       required if: makerequired has be set

Positional arguments:
  REQUIREDMULTIPLE       required multiple positional

Options:
  --makerequired MAKEREQUIRED [default: dog]
  --requiredvariable REQUIREDVARIABLE
  --help, -h             display this help and exit
`
	var args struct {
		RequiredMultiple []string `arg:"positional,required" help:"required multiple positional"`
		MakeRequired     string   `default:"dog"`
		RequiredVariable string   `arg:"required-if:makerequired|"`
	}

	p, err := NewParser(Config{}, &args)
	require.NoError(t, err)

	var help bytes.Buffer
	p.WriteHelp(&help)
	assert.Equal(t, expectedHelp, help.String())
}

func TestUsagWithNestedSubcommands(t *testing.T) {
	expectedHelp := `Usage: example child nested [--enable] OUTPUT

Required arguments:
  OUTPUT

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
	assert.Equal(t, expectedHelp, help.String())
}
