package arg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvalidTag(t *testing.T) {
	var args struct {
		Foo string `arg:"this_is_not_valid"`
	}
	_, err := NewParser(&args)
	assert.Error(t, err)
}

func TestUnexportedFieldsSkipped(t *testing.T) {
	var args struct {
		unexported struct{}
	}

	_, err := NewParser(&args)
	require.NoError(t, err)
}
