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

func TestNamedSubcommand(t *testing.T) {
	type listCmd struct {
	}
	var args struct {
		List *listCmd `arg:"subcommand:ls"`
	}
	err := parse("ls", &args)
	require.NoError(t, err)
	assert.NotNil(t, args.List)
}

func TestEmptySubcommand(t *testing.T) {
	type listCmd struct {
	}
	var args struct {
		List *listCmd `arg:"subcommand"`
	}
	err := parse("", &args)
	require.NoError(t, err)
	assert.Nil(t, args.List)
}

func TestTwoSubcommands(t *testing.T) {
	type getCmd struct {
	}
	type listCmd struct {
	}
	var args struct {
		Get  *getCmd  `arg:"subcommand"`
		List *listCmd `arg:"subcommand"`
	}
	err := parse("list", &args)
	require.NoError(t, err)
	assert.Nil(t, args.Get)
	assert.NotNil(t, args.List)
}

func TestSubcommandsWithOptions(t *testing.T) {
	type getCmd struct {
		Name string
	}
	type listCmd struct {
		Limit int
	}
	type cmd struct {
		Verbose bool
		Get     *getCmd  `arg:"subcommand"`
		List    *listCmd `arg:"subcommand"`
	}

	{
		var args cmd
		err := parse("list", &args)
		require.NoError(t, err)
		assert.Nil(t, args.Get)
		assert.NotNil(t, args.List)
	}

	{
		var args cmd
		err := parse("list --limit 3", &args)
		require.NoError(t, err)
		assert.Nil(t, args.Get)
		assert.NotNil(t, args.List)
		assert.Equal(t, args.List.Limit, 3)
	}

	{
		var args cmd
		err := parse("list --limit 3 --verbose", &args)
		require.NoError(t, err)
		assert.Nil(t, args.Get)
		assert.NotNil(t, args.List)
		assert.Equal(t, args.List.Limit, 3)
		assert.True(t, args.Verbose)
	}

	{
		var args cmd
		err := parse("list --verbose --limit 3", &args)
		require.NoError(t, err)
		assert.Nil(t, args.Get)
		assert.NotNil(t, args.List)
		assert.Equal(t, args.List.Limit, 3)
		assert.True(t, args.Verbose)
	}

	{
		var args cmd
		err := parse("--verbose list --limit 3", &args)
		require.NoError(t, err)
		assert.Nil(t, args.Get)
		assert.NotNil(t, args.List)
		assert.Equal(t, args.List.Limit, 3)
		assert.True(t, args.Verbose)
	}

	{
		var args cmd
		err := parse("get", &args)
		require.NoError(t, err)
		assert.NotNil(t, args.Get)
		assert.Nil(t, args.List)
	}

	{
		var args cmd
		err := parse("get --name test", &args)
		require.NoError(t, err)
		assert.NotNil(t, args.Get)
		assert.Nil(t, args.List)
		assert.Equal(t, args.Get.Name, "test")
	}
}
