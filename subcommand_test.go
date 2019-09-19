package arg

import (
	"bytes"
	"errors"
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

var helpA = `Usage: subparser list [--type TYPE] [--name NAME] [--foo FOO] [--bar BAR]

Options:
  --type TYPE [default: a]
  --name NAME
  --foo FOO
  --bar BAR
  --help, -h             display this help and exit
`

type optsB struct {
	Baz bool
	Moo string
}

var helpB = `Usage: subparser list [--type TYPE] [--name NAME] [--baz] [--moo MOO]

Options:
  --type TYPE [default: b]
  --name NAME
  --baz
  --moo MOO
  --help, -h             display this help and exit
`

type subCmdA struct {
	Type string
	Name string
	// it can be any interface
	Options interface{} `arg:"-"`
}

func (c *subCmdA) SubcommandParse(p *Parser, args []string) error {

	var t string
	var help bool

	// check for type and help params
	for i, a := range args {
		if a == "--help" || a == "-h" {
			help = true
		}
		if strings.HasPrefix(a, "--type=") {
			arr := strings.Split(a, "=")
			t = arr[1]
		}
		if a == "--type" && i != len(args)-1 {
			t = args[i+1]
		}
	}

	// no type provided, checkfor help request
	if t == "" {
		if help {
			return ErrHelp
		}
		return errors.New("--type required")
	}

	var opts interface{}

	// find options struct
	switch t {
	case "a":
		opts = &optsA{}
	case "b":
		opts = &optsB{}
	default:
		return fmt.Errorf("unknown type %s", t)
	}

	// add new destinations to the parser
	if err := p.AddDestinations(opts); err != nil {
		return err
	}

	// parse will marshal values to c & opts
	if err := p.Parse(args); err != nil {
		return err
	}
	c.Options = opts

	return nil
}

type subCmdB struct {
	Name string
}

var buff *bytes.Buffer

func TestCustomSubCommandParsing(t *testing.T) {
	type cmd struct {
		List     *subCmdA `arg:"subcommand"`
		Get      *subCmdB `arg:"subcommand"`
		GlobFlag string
	}

	{
		var args cmd
		err := parse("list --globflag before --type a --foo FOO --bar 42", &args)
		require.NoError(t, err)
		assert.Equal(t, "a", args.List.Type)
		opts, ok := args.List.Options.(*optsA)
		assert.True(t, ok)
		assert.Equal(t, "before", args.GlobFlag)
		assert.Equal(t, "", args.List.Name)
		assert.Equal(t, "FOO", opts.Foo)
		assert.Equal(t, 42, opts.Bar)
	}

	{
		var args cmd
		err := parse("list --type a --name john --foo FOO --bar 42 --globflag after", &args)
		require.NoError(t, err)
		assert.Equal(t, "a", args.List.Type)
		opts, ok := args.List.Options.(*optsA)
		assert.True(t, ok)
		assert.Equal(t, "after", args.GlobFlag)
		assert.Equal(t, "john", args.List.Name)
		assert.Equal(t, "FOO", opts.Foo)
		assert.Equal(t, 42, opts.Bar)
	}

	{
		var args cmd
		err := parse("list --type b --baz --globflag inbetween --moo cow_says", &args)
		require.NoError(t, err)
		assert.Equal(t, "b", args.List.Type)
		opts, ok := args.List.Options.(*optsB)
		assert.True(t, ok)
		assert.Equal(t, "inbetween", args.GlobFlag)
		assert.True(t, opts.Baz)
		assert.Equal(t, "cow_says", opts.Moo)
	}

	{
		var args cmd
		buff = &bytes.Buffer{}
		p, err := pparsename("subparser", "list --type a -h", &args)
		require.Equal(t, err, ErrHelp)
		p.WriteHelp(buff)
		assert.Equal(t, helpA, buff.String())
	}

	{
		var args cmd
		buff = &bytes.Buffer{}
		p, err := pparsename("subparser", "list --type b -h", &args)
		require.Equal(t, err, ErrHelp)
		p.WriteHelp(buff)
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
