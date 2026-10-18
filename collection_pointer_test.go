package arg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNilSlicePointer(t *testing.T) {
	for _, source := range []string{"flag", "environment"} {
		t.Run(source, func(t *testing.T) {
			var args struct {
				Values *[]string `arg:"env:GO_ARG_SLICE_POINTER"`
			}
			p, err := NewParser(Config{}, &args)
			require.NoError(t, err)
			require.NoError(t, p.Parse(nil))
			require.Nil(t, args.Values)
			commandLine := []string{"--values", "a", "b"}
			if source == "environment" {
				t.Setenv("GO_ARG_SLICE_POINTER", "a,b")
				commandLine = nil
			}
			require.NotPanics(t, func() { err = p.Parse(commandLine) })
			require.NoError(t, err)
			require.NotNil(t, args.Values)
			assert.Equal(t, []string{"a", "b"}, *args.Values)
		})
	}
}

func TestNilMapPointer(t *testing.T) {
	for _, source := range []string{"flag", "environment"} {
		t.Run(source, func(t *testing.T) {
			var args struct {
				Values *map[string]int `arg:"env:GO_ARG_MAP_POINTER"`
			}
			p, err := NewParser(Config{}, &args)
			require.NoError(t, err)
			require.NoError(t, p.Parse(nil))
			require.Nil(t, args.Values)
			commandLine := []string{"--values", "a=1", "b=2"}
			if source == "environment" {
				t.Setenv("GO_ARG_MAP_POINTER", "a=1,b=2")
				commandLine = nil
			}
			require.NotPanics(t, func() { err = p.Parse(commandLine) })
			require.NoError(t, err)
			require.NotNil(t, args.Values)
			assert.Equal(t, map[string]int{"a": 1, "b": 2}, *args.Values)
		})
	}
}
