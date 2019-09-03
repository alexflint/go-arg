package arg

import (
	"bytes"
	"fmt"
	"strings"
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

type optsA struct {
	Foo string
	Bar int
}

var helpA = `Usage: list [--type TYPE] [--foo FOO] [--bar BAR]

Options:
  --type TYPE [default: a]
  --foo FOO
  --bar BAR
  --help, -h             display this help and exit
`

type optsB struct {
	Baz bool
	Moo string
}

var helpB = `Usage: list [--type TYPE] [--baz] [--moo MOO]

Options:
  --type TYPE [default: b]
  --baz
  --moo MOO
  --help, -h             display this help and exit
`

type subCmdA struct {
	Type    string
	Options interface{} `arg:"-"`
}

func (c *subCmdA) UnmarshalText(b []byte) error {
	argsAll := strings.Split(string(b), " ")
	// remove command name
	args := argsAll[1:]
	// find type
	t := ""
	for i, a := range args {
		if a == "--type" {
			if len(a) == i+1 {
				break
			}
			t = args[i+1]
		}
		if strings.HasPrefix(a, "--type=") {
			t = strings.TrimPrefix(a, "--type=")
		}
	}
	if t == "" {
		return fmt.Errorf("no type provided")
	}

	c.Type = t

	// parse generic
	switch t {
	case "a":
		withAopts := &struct {
			Type string
			optsA
		}{}
		p, err := NewParser(Config{Program: argsAll[0]}, withAopts)
		if err != nil {
			return err
		}
		err = p.Parse(args)
		if err != nil {
			if err == ErrHelp {
				p.WriteHelp(buff)
				return nil
			}
			return err
		}
		c.Options = withAopts.optsA
	case "b":
		withBopts := &struct {
			Type string
			optsB
		}{}
		p, err := NewParser(Config{Program: argsAll[0]}, withBopts)
		if err != nil {
			return err
		}
		err = p.Parse(args)
		if err != nil {
			if err == ErrHelp {
				p.WriteHelp(buff)
				return nil
			}
			return err
		}
		c.Options = withBopts.optsB
	default:
		return fmt.Errorf("unknown type %s", t)
	}

	return nil
}

type subCmdB struct {
	Name string
}

var buff *bytes.Buffer

func TestCustomSubCommandParsing(t *testing.T) {
	type cmd struct {
		List *subCmdA `arg:"subcommand"`
		Get  *subCmdB `arg:"subcommand"`
	}

	{
		var args cmd
		err := parse("list --type a --foo FOO --bar 42", &args)
		require.NoError(t, err)
		assert.Equal(t, "a", args.List.Type)
		opts, ok := args.List.Options.(optsA)
		assert.True(t, ok)
		assert.Equal(t, "FOO", opts.Foo)
		assert.Equal(t, 42, opts.Bar)
	}

	{
		var args cmd
		err := parse("list --type b --baz --moo cow_says", &args)
		require.NoError(t, err)
		assert.Equal(t, "b", args.List.Type)
		opts, ok := args.List.Options.(optsB)
		assert.True(t, ok)
		assert.True(t, opts.Baz)
		assert.Equal(t, "cow_says", opts.Moo)
	}

	{
		var args cmd
		buff = &bytes.Buffer{}
		err := parse("list --type a -h", &args)
		require.NoError(t, err)
		assert.Equal(t, helpA, buff.String())
	}

	{
		var args cmd
		buff = &bytes.Buffer{}
		err := parse("list --type b -h", &args)
		require.NoError(t, err)
		assert.Equal(t, helpB, buff.String())
	}

	{
		var args cmd
		buff = &bytes.Buffer{}
		err := parse("get --name unknown", &args)
		require.NoError(t, err)
		assert.Equal(t, "unknown", args.Get.Name)
	}
}
