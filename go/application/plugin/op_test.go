package plugin

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testInput struct {
	Name string `json:"name"`
}

type testOutput struct {
	Result string `json:"result"`
}

func TestRegisterOp(t *testing.T) {
	clearOpRegistry()

	RegisterOp[testInput, testOutput]("TestOp",
		Example[testInput, testOutput]{
			Name:           "basic",
			Input:          testInput{Name: "test"},
			ExpectedOutput: &testOutput{Result: "ok"},
		},
		Example[testInput, testOutput]{
			Name:          "error",
			Input:         testInput{Name: ""},
			ExpectedError: "name required",
		},
	)

	info, ok := getOpTypeInfo("TestOp")
	require.True(t, ok)

	assert.Equal(t, reflect.TypeOf(testInput{}), info.inputType)
	assert.Equal(t, reflect.TypeOf(testOutput{}), info.outputType)
	assert.Len(t, info.examples, 2)
}

func TestRegisterOp_NotRegistered(t *testing.T) {
	clearOpRegistry()

	_, ok := getOpTypeInfo("NonExistent")
	assert.False(t, ok)
}
