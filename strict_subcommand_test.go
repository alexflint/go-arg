package arg

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubcommandAncestorDefaults(t *testing.T) {
	for _, strict := range []bool{false, true} {
		for _, ignoreDefault := range []bool{false, true} {
			t.Run(fmt.Sprintf("strict=%v/ignoreDefault=%v", strict, ignoreDefault), func(t *testing.T) {
				type leaf struct {
					Value int `default:"30"`
				}
				type child struct {
					Value int   `default:"20"`
					Leaf  *leaf `arg:"subcommand:leaf|l"`
				}
				var args struct {
					Value *int   `default:"10"`
					Child *child `arg:"subcommand:child|c"`
					Other *struct {
						Required string `arg:"required"`
					} `arg:"subcommand"`
				}
				p, err := NewParser(Config{StrictSubcommands: strict, IgnoreDefault: ignoreDefault}, &args)
				require.NoError(t, err)
				require.NoError(t, p.Parse([]string{"c", "l"}))
				require.NotNil(t, args.Child)
				require.NotNil(t, args.Child.Leaf)
				assert.Nil(t, args.Other)
				if ignoreDefault {
					assert.Nil(t, args.Value)
					assert.Zero(t, args.Child.Value)
					assert.Zero(t, args.Child.Leaf.Value)
				} else {
					require.NotNil(t, args.Value)
					assert.Equal(t, 10, *args.Value)
					assert.Equal(t, 20, args.Child.Value)
					assert.Equal(t, 30, args.Child.Leaf.Value)
				}
			})
		}
	}
}

func TestSubcommandAncestorRequired(t *testing.T) {
	for _, strict := range []bool{false, true} {
		for _, tc := range []struct {
			name string
			argv []string
			err  string
		}{
			{"missing root", []string{"child", "--child", "ok"}, "ROOT is required"},
			{"missing parent", []string{"--root", "ok", "child", "leaf"}, "CHILD is required"},
			{"missing both", []string{"child", "leaf"}, "ROOT is required"},
			{"all present", []string{"--root", "ok", "child", "--child", "ok", "leaf"}, ""},
			{"aliases", []string{"--root", "ok", "c", "--child", "ok", "l"}, ""},
			{"root only", []string{"--root", "ok"}, ""},
		} {
			t.Run(fmt.Sprintf("strict=%v/%s", strict, tc.name), func(t *testing.T) {
				type child struct {
					Child string    `arg:"required"`
					Leaf  *struct{} `arg:"subcommand:leaf|l"`
				}
				var args struct {
					Root  string `arg:"required"`
					Child *child `arg:"subcommand:child|c"`
					Other *struct {
						Required string `arg:"required"`
					} `arg:"subcommand"`
				}
				p, err := NewParser(Config{StrictSubcommands: strict}, &args)
				require.NoError(t, err)
				err = p.Parse(tc.argv)
				if tc.err != "" {
					assert.EqualError(t, err, tc.err)
				} else {
					assert.NoError(t, err)
				}
				assert.Nil(t, args.Other)
			})
		}
	}
}

func TestStrictSubcommandAncestorValuePrecedence(t *testing.T) {
	t.Setenv("GO_ARG_TEST_ANCESTOR_ROOT", "env-root")
	t.Setenv("GO_ARG_TEST_ANCESTOR_CHILD", "env-child")
	for _, tc := range []struct {
		name      string
		argv      []string
		ignoreEnv bool
		root      string
		child     string
	}{
		{"environment", []string{"child", "leaf"}, false, "env-root", "env-child"},
		{"command line", []string{"--value", "cli-root", "child", "--value", "cli-child", "leaf"}, false, "cli-root", "cli-child"},
		{"defaults", []string{"child", "leaf"}, true, "default-root", "default-child"},
		{"local override", []string{"child", "--value", "cli-child", "leaf"}, true, "default-root", "cli-child"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			type child struct {
				Value string    `arg:"env:GO_ARG_TEST_ANCESTOR_CHILD" default:"default-child"`
				Leaf  *struct{} `arg:"subcommand"`
			}
			var args struct {
				Value string `arg:"env:GO_ARG_TEST_ANCESTOR_ROOT" default:"default-root"`
				Child *child `arg:"subcommand"`
			}
			p, err := NewParser(Config{StrictSubcommands: true, IgnoreEnv: tc.ignoreEnv}, &args)
			require.NoError(t, err)
			require.NoError(t, p.Parse(tc.argv))
			assert.Equal(t, tc.root, args.Value)
			assert.Equal(t, tc.child, args.Child.Value)
		})
	}
}
