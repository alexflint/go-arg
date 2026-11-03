package arg

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInlineStructArguments(t *testing.T) {
	var args struct {
		A struct {
			B string
			C string
		} `arg:"inline"`
		D string
	}
	p, err := NewParser(Config{}, &args)
	require.NoError(t, err)
	require.NoError(t, p.Parse([]string{"--b", "bee", "--c", "see", "--d", "dee"}))
	data, err := json.Marshal(args)
	require.NoError(t, err)
	require.JSONEq(t, `{"A":{"B":"bee","C":"see"},"D":"dee"}`, string(data))
	var help bytes.Buffer
	p.WriteHelp(&help)
	require.Contains(t, help.String(), "--b B")
	require.NotContains(t, help.String(), "--a ")
}

func TestNamedStructGroupsPreserveNestedPaths(t *testing.T) {
	type Inner struct {
		Value string `arg:"--inner"`
	}
	type Group struct {
		Nested Inner `arg:"inline"`
		Other  string
	}
	var args struct {
		First  Group                `arg:"inline"`
		Second struct{ Number int } `arg:"inline"`
	}
	p, err := NewParser(Config{}, &args)
	require.NoError(t, err)
	require.NoError(t, p.Parse([]string{"--inner", "nested", "--other", "sibling", "--number", "42"}))
	require.Equal(t, "nested", args.First.Nested.Value)
	require.Equal(t, "sibling", args.First.Other)
	require.Equal(t, 42, args.Second.Number)
}

func TestInlineGroupDefaultsEnvironmentAndRequired(t *testing.T) {
	var args struct {
		Group struct {
			Required string `arg:"required"`
			FromEnv  string `arg:"env:GO_ARG_INLINE_TEST_VALUE"`
			Default  int    `default:"7"`
			Existing int
		} `arg:"inline"`
		Ignored struct{ Unsupported func() } `arg:"-"`
	}
	args.Group.Existing = 9
	t.Setenv("GO_ARG_INLINE_TEST_VALUE", "from-env")
	p, err := NewParser(Config{}, &args)
	require.NoError(t, err)
	require.Error(t, p.Parse(nil))
	require.NoError(t, p.Parse([]string{"--required", "yes"}))
	require.Equal(t, "from-env", args.Group.FromEnv)
	require.Equal(t, 7, args.Group.Default)
	require.Equal(t, 9, args.Group.Existing)
}

func TestInlineGroupsInsideSubcommands(t *testing.T) {
	var args struct {
		Global struct{ Verbose bool } `arg:"inline"`
		Run    *struct {
			Options struct {
				Paths []string `arg:"--path"`
			} `arg:"inline"`
		} `arg:"subcommand:run"`
	}
	p, err := NewParser(Config{}, &args)
	require.NoError(t, err)
	require.NoError(t, p.Parse([]string{"--verbose", "run", "--path", "one", "two"}))
	require.True(t, args.Global.Verbose)
	require.Equal(t, []string{"one", "two"}, args.Run.Options.Paths)
}

func TestInlineGroupsKeepScalarStructParsing(t *testing.T) {
	var args struct {
		URL url.URL
	}
	p, err := NewParser(Config{}, &args)
	require.NoError(t, err)
	require.NoError(t, p.Parse([]string{"--url", "https://example.com/path"}))
	require.Equal(t, "example.com", args.URL.Host)
	require.Equal(t, "/path", args.URL.Path)
	var help bytes.Buffer
	p.WriteHelp(&help)
	require.False(t, strings.Contains(help.String(), "--host"))
}

func TestInlineGroupRejectsInvalidDefinitions(t *testing.T) {
	for _, tc := range []struct {
		name string
		dest interface{}
		want string
	}{
		{"scalar", &struct {
			Value string `arg:"inline"`
		}{}, "inline fields must be structs"},
		{"pointer", &struct {
			Value *struct{} `arg:"inline"`
		}{}, "inline fields must be structs"},
		{"default", &struct {
			Value struct{} `arg:"inline" default:""`
		}{}, "default belongs on an argument"},
		{"help", &struct {
			Value struct{} `arg:"inline" help:"group"`
		}{}, "help belongs on an argument"},
		{"placeholder", &struct {
			Value struct{} `arg:"inline" placeholder:"GROUP"`
		}{}, "placeholder belongs on an argument"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewParser(Config{}, tc.dest)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestInlineGroupsAreOptIn(t *testing.T) {
	var args struct{ Group struct{ Value string } }
	_, err := NewParser(Config{}, &args)
	require.Error(t, err)
	require.Contains(t, err.Error(), "fields are not supported")
}
