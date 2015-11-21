package arg

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteUsage(t *testing.T) {
	expectedUsage := "usage: example [--verbose] [--dataset DATASET] [--optimize OPTIMIZE] INPUT [OUTPUT [OUTPUT ...]]\n"

	expectedHelp := `usage: example [--verbose] [--dataset DATASET] [--optimize OPTIMIZE] INPUT [OUTPUT [OUTPUT ...]]

positional arguments:
  input
  output                 positional output

options:
  --verbose, -v          verbosity level
  --dataset DATASET      dataset to use
  --optimize OPTIMIZE, -O OPTIMIZE
                         optimization level
  --help, -h             display this help and exit
`
	var args struct {
		Input    string   `arg:"positional"`
		Output   []string `arg:"positional,help:positional output"`
		Verbose  bool     `arg:"-v,help:verbosity level"`
		Dataset  string   `arg:"help:dataset to use"`
		Optimize int      `arg:"-O,help:optimization level"`
	}
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

func TestUsageLongPositionalWithHelp(t *testing.T) {
	expectedHelp := `usage: example VERYLONGPOSITIONALWITHHELP

positional arguments:
  verylongpositionalwithhelp
                         this positional argument is very long

options:
  --help, -h             display this help and exit
`
	var args struct {
		VeryLongPositionalWithHelp string `arg:"positional,help:this positional argument is very long"`
	}

	p, err := NewParser(&args)
	require.NoError(t, err)

	os.Args[0] = "example"
	var help bytes.Buffer
	p.WriteHelp(&help)
	assert.Equal(t, expectedHelp, help.String())
}
