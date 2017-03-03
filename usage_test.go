package arg

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteUsage(t *testing.T) {
	expectedUsage := "usage: example [-v] [--name] [--value] [--dataset] [-O] [--ids] [--values] [-w] INPUT [OUTPUT [OUTPUT ...]] \n"

	expectedHelp := `usage: example [-v] [--name] [--value] [--dataset] [-O] [--ids] [--values] [-w] INPUT [OUTPUT [OUTPUT ...]] 

positional arguments:
  input                      
  output                     list of outputs

options:
      --name=Foo Bar         name to use [default: Foo Bar]
      --value=42             secret value [default: 42]
  -v, --verbose              verbosity level 
      --dataset=<s>          dataset to use 
  -O, --optimize=<n>         optimization level 
      --ids=[n]              Ids 
      --values=[3.14 42 256] Values [default: [3.14 42 256]]
  -w, --workers=<n>          number of workers to start 
  -h, --help                 display this help and exit 
`
	var args struct {
		Input    string    `arg:"positional"`
		Output   []string  `arg:"positional,help:list of outputs"`
		Name     string    `arg:"help:name to use"`
		Value    int       `arg:"help:secret value"`
		Verbose  bool      `arg:"-v,help:verbosity level"`
		Dataset  string    `arg:"help:dataset to use"`
		Optimize int       `arg:"-O,help:optimization level"`
		Ids      []int64   `arg:"help:Ids"`
		Values   []float64 `arg:"help:Values"`
		Workers  int       `arg:"-w,env:WORKERS,help:number of workers to start"`
	}
	args.Name = "Foo Bar"
	args.Value = 42
	args.Values = []float64{3.14, 42, 256}
	p, err := NewParser(Config{}, &args)
	require.NoError(t, err)

	os.Args[0] = "example"

	var usage bytes.Buffer
	p.WriteUsage(&usage)
	assert.Equal(t, expectedUsage, usage.String())

	var help bytes.Buffer
	p.WriteHelp(&help)
	assert.Equal(t, expectedHelp, help.String())
}

func TestUsageLongPositionalWithHelp(t *testing.T) {
	expectedHelp := `usage: example VERY-LONG-POSITIONAL-WITH-HELP 

positional arguments:
  very-long-positional-with-help
                             this positional argument is very long

options:
  -h, --help                 display this help and exit 
`
	var args struct {
		VeryLongPositionalWithHelp string `arg:"positional,help:this positional argument is very long"`
	}

	p, err := NewParser(Config{}, &args)
	require.NoError(t, err)

	os.Args[0] = "example"
	var help bytes.Buffer
	p.WriteHelp(&help)
	assert.Equal(t, expectedHelp, help.String())
}

func TestUsageWithProgramName(t *testing.T) {
	expectedHelp := `usage: myprogram 

options:
  -h, --help                 display this help and exit 
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
usage: example 

options:
  -h, --help                 display this help and exit 
      --version              display version and exit 
`
	os.Args[0] = "example"
	p, err := NewParser(Config{}, &versioned{})
	require.NoError(t, err)

	var help bytes.Buffer
	p.WriteHelp(&help)
	actual := help.String()
	t.Logf("Expected:\n%s", expectedHelp)
	t.Logf("Actual:\n%s", actual)
	if expectedHelp != actual {
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
usage: example 

options:
  -h, --help                 display this help and exit 
`
	os.Args[0] = "example"
	p, err := NewParser(Config{}, &described{})
	require.NoError(t, err)

	var help bytes.Buffer
	p.WriteHelp(&help)
	actual := help.String()
	t.Logf("Expected:\n%s", expectedHelp)
	t.Logf("Actual:\n%s", actual)
	if expectedHelp != actual {
		t.Fail()
	}
}
