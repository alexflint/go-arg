package arg

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteUsage(t *testing.T) {
	expectedUsage := "usage: example [--name NAME] [--value VALUE] [--verbose] [--dataset DATASET] [--optimize OPTIMIZE] [--ids IDS] INPUT [OUTPUT [OUTPUT ...]]\n"

	expectedHelp := `usage: example [--name NAME] [--value VALUE] [--verbose] [--dataset DATASET] [--optimize OPTIMIZE] [--ids IDS] INPUT [OUTPUT [OUTPUT ...]]

positional arguments:
  input
  output

options:
  --name NAME            name to use [default: Foo Bar]
  --value VALUE          secret value [default: 42]
  --verbose, -v          verbosity level
  --dataset DATASET      dataset to use
  --optimize OPTIMIZE, -O OPTIMIZE
                         optimization level
  --ids IDS              Ids
  --help, -h             display this help and exit
`
	var args struct {
		Input    string   `arg:"positional"`
		Output   []string `arg:"positional"`
		Name     string   `arg:"help:name to use"`
		Value    int      `arg:"help:secret value"`
		Verbose  bool     `arg:"-v,help:verbosity level"`
		Dataset  string   `arg:"help:dataset to use"`
		Optimize int      `arg:"-O,help:optimization level"`
		Ids      []int64  `arg:"help:Ids"`
	}
	args.Name = "Foo Bar"
	args.Value = 42
	p, err := NewParser(&args)
	require.NoError(t, err)

	os.Args[0] = "example"

	var usage bytes.Buffer
	p.WriteUsage(&usage)
	assert.Equal(t, expectedUsage, usage.String())

	var help bytes.Buffer
	p.WriteHelp(&help)
	assert.Equal(t, expectedHelp, help.String())
}

func TestVersionInHelp(t *testing.T) {
	expectedHelp := `usage: example [--verbose]

options:
  --verbose, -v          verbosity level
  --help, -h             display this help and exit
  --version              output version information and exit
`
	var args struct {
		Verbose bool `arg:"-v,help:verbosity level"`
	}

	SetVersion("1.2.3")
	p, err := NewParser(&args)
	require.NoError(t, err)

	os.Args[0] = "example"

	var help bytes.Buffer
	p.WriteHelp(&help)
	assert.Equal(t, expectedHelp, help.String())
}

func TestWriteVersion(t *testing.T) {
	expectedVersion := "example 1.0.0\n"

	var args struct {
	}

	SetVersion("1.0.0")
	p, err := NewParser(&args)
	require.NoError(t, err)

	os.Args[0] = "example"

	var version bytes.Buffer
	p.WriteVersion(&version)
	assert.Equal(t, expectedVersion, version.String())
}
