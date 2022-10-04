package arg

import (
	"reflect"
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
		A string    `arg:"positional"`
		B *struct{} `arg:"subcommand"`
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
	p, err := pparse("list", &args)
	require.NoError(t, err)
	assert.NotNil(t, args.List)
	assert.Equal(t, args.List, p.Subcommand())
	assert.Equal(t, []string{"list"}, p.SubcommandNames())
}

func TestSubcommandNamesBeforeParsing(t *testing.T) {
	type listCmd struct{}
	var args struct {
		List *listCmd `arg:"subcommand"`
	}
	p, err := NewParser(Config{}, &args)
	require.NoError(t, err)
	assert.Nil(t, p.Subcommand())
	assert.Nil(t, p.SubcommandNames())
}

func TestNoSuchSubcommand(t *testing.T) {
	type listCmd struct {
	}
	var args struct {
		List *listCmd `arg:"subcommand"`
	}
	_, err := pparse("invalid", &args)
	assert.Error(t, err)
}

func TestNamedSubcommand(t *testing.T) {
	type listCmd struct {
	}
	var args struct {
		List *listCmd `arg:"subcommand:ls"`
	}
	p, err := pparse("ls", &args)
	require.NoError(t, err)
	assert.NotNil(t, args.List)
	assert.Equal(t, args.List, p.Subcommand())
	assert.Equal(t, []string{"ls"}, p.SubcommandNames())
}

func TestEmptySubcommand(t *testing.T) {
	type listCmd struct {
	}
	var args struct {
		List *listCmd `arg:"subcommand"`
	}
	p, err := pparse("", &args)
	require.NoError(t, err)
	assert.Nil(t, args.List)
	assert.Nil(t, p.Subcommand())
	assert.Empty(t, p.SubcommandNames())
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
	p, err := pparse("list", &args)
	require.NoError(t, err)
	assert.Nil(t, args.Get)
	assert.NotNil(t, args.List)
	assert.Equal(t, args.List, p.Subcommand())
	assert.Equal(t, []string{"list"}, p.SubcommandNames())
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

func TestSubcommandsWithEnvVars(t *testing.T) {
	type getCmd struct {
		Name string `arg:"env"`
	}
	type listCmd struct {
		Limit int `arg:"env"`
	}
	type cmd struct {
		Verbose bool
		Get     *getCmd  `arg:"subcommand"`
		List    *listCmd `arg:"subcommand"`
	}

	{
		var args cmd
		setenv(t, "LIMIT", "123")
		err := parse("list", &args)
		require.NoError(t, err)
		require.NotNil(t, args.List)
		assert.Equal(t, 123, args.List.Limit)
	}

	{
		var args cmd
		setenv(t, "LIMIT", "not_an_integer")
		err := parse("list", &args)
		assert.Error(t, err)
	}
}

func TestNestedSubcommands(t *testing.T) {
	type child struct{}
	type parent struct {
		Child *child `arg:"subcommand"`
	}
	type grandparent struct {
		Parent *parent `arg:"subcommand"`
	}
	type root struct {
		Grandparent *grandparent `arg:"subcommand"`
	}

	{
		var args root
		p, err := pparse("grandparent parent child", &args)
		require.NoError(t, err)
		require.NotNil(t, args.Grandparent)
		require.NotNil(t, args.Grandparent.Parent)
		require.NotNil(t, args.Grandparent.Parent.Child)
		assert.Equal(t, args.Grandparent.Parent.Child, p.Subcommand())
		assert.Equal(t, []string{"grandparent", "parent", "child"}, p.SubcommandNames())
	}

	{
		var args root
		p, err := pparse("grandparent parent", &args)
		require.NoError(t, err)
		require.NotNil(t, args.Grandparent)
		require.NotNil(t, args.Grandparent.Parent)
		require.Nil(t, args.Grandparent.Parent.Child)
		assert.Equal(t, args.Grandparent.Parent, p.Subcommand())
		assert.Equal(t, []string{"grandparent", "parent"}, p.SubcommandNames())
	}

	{
		var args root
		p, err := pparse("grandparent", &args)
		require.NoError(t, err)
		require.NotNil(t, args.Grandparent)
		require.Nil(t, args.Grandparent.Parent)
		assert.Equal(t, args.Grandparent, p.Subcommand())
		assert.Equal(t, []string{"grandparent"}, p.SubcommandNames())
	}

	{
		var args root
		p, err := pparse("", &args)
		require.NoError(t, err)
		require.Nil(t, args.Grandparent)
		assert.Nil(t, p.Subcommand())
		assert.Empty(t, p.SubcommandNames())
	}
}

func TestSubcommandsWithPositionals(t *testing.T) {
	type listCmd struct {
		Pattern string `arg:"positional"`
	}
	type cmd struct {
		Format string
		List   *listCmd `arg:"subcommand"`
	}

	{
		var args cmd
		err := parse("list", &args)
		require.NoError(t, err)
		assert.NotNil(t, args.List)
		assert.Equal(t, "", args.List.Pattern)
	}

	{
		var args cmd
		err := parse("list --format json", &args)
		require.NoError(t, err)
		assert.NotNil(t, args.List)
		assert.Equal(t, "", args.List.Pattern)
		assert.Equal(t, "json", args.Format)
	}

	{
		var args cmd
		err := parse("list somepattern", &args)
		require.NoError(t, err)
		assert.NotNil(t, args.List)
		assert.Equal(t, "somepattern", args.List.Pattern)
	}

	{
		var args cmd
		err := parse("list somepattern --format json", &args)
		require.NoError(t, err)
		assert.NotNil(t, args.List)
		assert.Equal(t, "somepattern", args.List.Pattern)
		assert.Equal(t, "json", args.Format)
	}

	{
		var args cmd
		err := parse("list --format json somepattern", &args)
		require.NoError(t, err)
		assert.NotNil(t, args.List)
		assert.Equal(t, "somepattern", args.List.Pattern)
		assert.Equal(t, "json", args.Format)
	}

	{
		var args cmd
		err := parse("--format json list somepattern", &args)
		require.NoError(t, err)
		assert.NotNil(t, args.List)
		assert.Equal(t, "somepattern", args.List.Pattern)
		assert.Equal(t, "json", args.Format)
	}

	{
		var args cmd
		err := parse("--format json", &args)
		require.NoError(t, err)
		assert.Nil(t, args.List)
		assert.Equal(t, "json", args.Format)
	}
}
func TestSubcommandsWithMultiplePositionals(t *testing.T) {
	type getCmd struct {
		Items []string `arg:"positional"`
	}
	type cmd struct {
		Limit int
		Get   *getCmd `arg:"subcommand"`
	}

	{
		var args cmd
		err := parse("get", &args)
		require.NoError(t, err)
		assert.NotNil(t, args.Get)
		assert.Empty(t, args.Get.Items)
	}

	{
		var args cmd
		err := parse("get --limit 5", &args)
		require.NoError(t, err)
		assert.NotNil(t, args.Get)
		assert.Empty(t, args.Get.Items)
		assert.Equal(t, 5, args.Limit)
	}

	{
		var args cmd
		err := parse("get item1", &args)
		require.NoError(t, err)
		assert.NotNil(t, args.Get)
		assert.Equal(t, []string{"item1"}, args.Get.Items)
	}

	{
		var args cmd
		err := parse("get item1 item2 item3", &args)
		require.NoError(t, err)
		assert.NotNil(t, args.Get)
		assert.Equal(t, []string{"item1", "item2", "item3"}, args.Get.Items)
	}

	{
		var args cmd
		err := parse("get item1 --limit 5 item2", &args)
		require.NoError(t, err)
		assert.NotNil(t, args.Get)
		assert.Equal(t, []string{"item1", "item2"}, args.Get.Items)
		assert.Equal(t, 5, args.Limit)
	}
}

func TestValForNilStruct(t *testing.T) {
	type subcmd struct{}
	var cmd struct {
		Sub *subcmd `arg:"subcommand"`
	}

	p, err := NewParser(Config{}, &cmd)
	require.NoError(t, err)

	typ := reflect.TypeOf(cmd)
	subField, _ := typ.FieldByName("Sub")

	v := p.val(path{fields: []reflect.StructField{subField, subField}})
	assert.False(t, v.IsValid())
}
