package arg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// this file contains tests related to the precedence rules for:
//   ProcessCommandLine
//   ProcessOptions
//   ProcessPositionals
//   ProcessEnvironment
//   ProcessMap
//   ProcessSingle
//   ProcessSequence
//   OverwriteWithCommandLine
//   OverwriteWithOptions
//   OverwriteWithPositionals
//   OverwriteWithEnvironment
//   OverwriteWithMap
//
// The Process* functions should not overwrite fields that have
// been previously populated, whereas the OverwriteWith* functions
// should overwrite fields that have been previously populated.

// check that we can accumulate "separate" args across env, cmdline, map, and defaults

// check what happens if we have a required arg with a default value

// add more tests for combinations of separate and cardinality

// check what happens if we call ProcessCommandLine multiple times with different subcommands

func TestProcessOptions(t *testing.T) {
	var args struct {
		Arg string
	}

	p, err := NewParser(&args)
	require.NoError(t, err)

	_, err = p.ProcessOptions([]string{"program", "--arg=hello"})
	require.NoError(t, err)
	assert.Equal(t, "hello", args.Arg)
}

func TestProcessOptionsDoesNotOverwrite(t *testing.T) {
	var args struct {
		Arg string `arg:"env"`
	}

	p, err := NewParser(&args)
	require.NoError(t, err)

	err = p.ProcessEnvironment([]string{"ARG=123"})
	require.NoError(t, err)

	_, err = p.ProcessOptions([]string{"--arg=hello"})
	require.NoError(t, err)

	assert.EqualValues(t, "123", args.Arg)
}

func TestOverwriteWithOptions(t *testing.T) {
	var args struct {
		Arg string `arg:"env"`
	}

	p, err := NewParser(&args)
	require.NoError(t, err)

	err = p.ProcessEnvironment([]string{"ARG=123"})
	require.NoError(t, err)

	_, err = p.OverwriteWithOptions([]string{"--arg=hello"})
	require.NoError(t, err)

	assert.EqualValues(t, "hello", args.Arg)
}

func TestProcessPositionals(t *testing.T) {
	var args struct {
		Arg string `arg:"positional"`
	}

	p, err := NewParser(&args)
	require.NoError(t, err)

	err = p.ProcessPositionals([]string{"hello"})
	require.NoError(t, err)
	assert.Equal(t, "hello", args.Arg)
}

func TestProcessPositionalsDoesNotOverwrite(t *testing.T) {
	var args struct {
		Arg string `arg:"env,positional"`
	}

	p, err := NewParser(&args)
	require.NoError(t, err)

	err = p.ProcessEnvironment([]string{"ARG=123"})
	require.NoError(t, err)

	err = p.ProcessPositionals([]string{"hello"})
	require.NoError(t, err)

	assert.EqualValues(t, "123", args.Arg)
}

func TestOverwriteWithPositionals(t *testing.T) {
	var args struct {
		Arg string `arg:"env,positional"`
	}

	p, err := NewParser(&args)
	require.NoError(t, err)

	err = p.ProcessEnvironment([]string{"ARG=123"})
	require.NoError(t, err)

	err = p.OverwriteWithPositionals([]string{"hello"})
	require.NoError(t, err)

	assert.EqualValues(t, "hello", args.Arg)
}

func TestProcessCommandLine(t *testing.T) {
	var args struct {
		Arg string
	}

	p, err := NewParser(&args)
	require.NoError(t, err)

	err = p.ProcessCommandLine([]string{"program", "--arg=hello"})
	require.NoError(t, err)
	assert.Equal(t, "hello", args.Arg)
}

func TestProcessCommandLineDoesNotOverwrite(t *testing.T) {
	var args struct {
		Arg string `arg:"env"`
	}

	p, err := NewParser(&args)
	require.NoError(t, err)

	err = p.ProcessEnvironment([]string{"ARG=123"})
	require.NoError(t, err)

	err = p.ProcessCommandLine([]string{"program", "--arg=hello"})
	require.NoError(t, err)

	assert.EqualValues(t, "123", args.Arg)
}

func TestOverwriteWithCommandLine(t *testing.T) {
	var args struct {
		Arg string `arg:"env"`
	}

	p, err := NewParser(&args)
	require.NoError(t, err)

	err = p.ProcessEnvironment([]string{"ARG=123"})
	require.NoError(t, err)

	err = p.OverwriteWithCommandLine([]string{"program", "--arg=hello"})
	require.NoError(t, err)

	assert.EqualValues(t, "hello", args.Arg)
}

func TestProcessEnvironment(t *testing.T) {
	var args struct {
		Arg string `arg:"env"`
	}

	p, err := NewParser(&args)
	require.NoError(t, err)

	err = p.ProcessEnvironment([]string{"ARG=hello"})
	require.NoError(t, err)

	assert.EqualValues(t, "hello", args.Arg)
}

func TestProcessEnvironmentDoesNotOverwrite(t *testing.T) {
	var args struct {
		Arg string `arg:"env"`
	}

	p, err := NewParser(&args)
	require.NoError(t, err)

	_, err = p.ProcessOptions([]string{"--arg=123"})
	require.NoError(t, err)

	err = p.ProcessEnvironment([]string{"ARG=hello"})
	require.NoError(t, err)

	assert.EqualValues(t, "123", args.Arg)
}

func TestOverwriteWithEnvironment(t *testing.T) {
	var args struct {
		Arg string `arg:"env"`
	}

	p, err := NewParser(&args)
	require.NoError(t, err)

	_, err = p.ProcessOptions([]string{"--arg=123"})
	require.NoError(t, err)

	err = p.OverwriteWithEnvironment([]string{"ARG=hello"})
	require.NoError(t, err)

	assert.EqualValues(t, "hello", args.Arg)
}

func TestProcessDefaults(t *testing.T) {
	var args struct {
		Arg string `default:"hello"`
	}

	p, err := NewParser(&args)
	require.NoError(t, err)

	err = p.ProcessDefaults()
	require.NoError(t, err)

	assert.EqualValues(t, "hello", args.Arg)
}

func TestProcessDefaultsDoesNotOverwrite(t *testing.T) {
	var args struct {
		Arg string `default:"hello"`
	}

	p, err := NewParser(&args)
	require.NoError(t, err)

	_, err = p.ProcessOptions([]string{"--arg=123"})
	require.NoError(t, err)

	err = p.ProcessDefaults()
	require.NoError(t, err)

	assert.EqualValues(t, "123", args.Arg)
}

func TestOverwriteWithDefaults(t *testing.T) {
	var args struct {
		Arg string `default:"hello"`
	}

	p, err := NewParser(&args)
	require.NoError(t, err)

	_, err = p.ProcessOptions([]string{"--arg=123"})
	require.NoError(t, err)

	err = p.OverwriteWithDefaults()
	require.NoError(t, err)

	assert.EqualValues(t, "hello", args.Arg)
}
