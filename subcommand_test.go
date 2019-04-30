package arg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file contains tests for parse.go but I decided to put them here
// since that file is getting large

func TestSubcommandNotAPointer(t *testing.T) {
	var args struct {
		A string `arg:"subcommand"`
	}
	_, err := NewParser(Config{}, &args)
	assert.Error(t, err)
}

func TestSubcommandNotAPointerToStruct(t *testing.T) {
	var args struct {
		A struct{} `arg:"subcommand"`
	}
	_, err := NewParser(Config{}, &args)
	assert.Error(t, err)
}

func TestPositionalAndSubcommandNotAllowed(t *testing.T) {
	var args struct {
		A string   `arg:"positional"`
		B struct{} `arg:"subcommand"`
	}
	_, err := NewParser(Config{}, &args)
	assert.Error(t, err)
}

func TestMinimalSubcommand(t *testing.T) {
	type listCmd struct {
	}
	var args struct {
		List *listCmd `arg:"subcommand"`
	}
	err := parse("list", &args)
	require.NoError(t, err)
	assert.NotNil(t, args.List)
}
