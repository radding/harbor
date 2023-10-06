package mathparser

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCase struct {
	valueA  *Value
	valueB  *Value
	results map[Operator]bool
}

var tests = []testCase{
	{
		valueA: &Value{Number: floatPtr(1.0)},
		valueB: &Value{Number: floatPtr(1.0)},
		results: map[Operator]bool{
			OpEq:  true,
			OpGt:  false,
			OpLt:  false,
			OpNeq: false,
			OpGte: true,
			OpLte: true,
		},
	},
	{
		valueA: &Value{Number: floatPtr(1.0)},
		valueB: &Value{Number: floatPtr(2.0)},
		results: map[Operator]bool{
			OpEq:  false,
			OpGt:  false,
			OpLt:  true,
			OpNeq: true,
			OpGte: false,
			OpLte: true,
		},
	},
	{
		valueA: &Value{Number: floatPtr(2.0)},
		valueB: &Value{Number: floatPtr(1.0)},
		results: map[Operator]bool{
			OpEq:  false,
			OpGt:  true,
			OpLt:  false,
			OpNeq: true,
			OpGte: true,
			OpLte: false,
		},
	},

	{
		valueA: &Value{StringValue: stringPtr("ABC")},
		valueB: &Value{StringValue: stringPtr("ABC")},
		results: map[Operator]bool{
			OpEq:  true,
			OpGt:  false,
			OpLt:  false,
			OpNeq: false,
			OpGte: true,
			OpLte: true,
		},
	},
	{
		valueA: &Value{StringValue: stringPtr("ABC")},
		valueB: &Value{StringValue: stringPtr("XYZ")},
		results: map[Operator]bool{
			OpEq:  false,
			OpGt:  false,
			OpLt:  true,
			OpNeq: true,
			OpGte: false,
			OpLte: true,
		},
	},
	{
		valueA: &Value{StringValue: stringPtr("XYZ")},
		valueB: &Value{StringValue: stringPtr("ABC")},
		results: map[Operator]bool{
			OpEq:  false,
			OpGt:  true,
			OpLt:  false,
			OpNeq: true,
			OpGte: true,
			OpLte: false,
		},
	},

	{
		valueA: &Value{BoolVal: (*Boolean)(boolPtr(true))},
		valueB: &Value{BoolVal: (*Boolean)(boolPtr(true))},
		results: map[Operator]bool{
			OpEq:  true,
			OpGt:  false,
			OpLt:  false,
			OpNeq: false,
			OpGte: false,
			OpLte: false,
		},
	},
	{
		valueA: &Value{BoolVal: (*Boolean)(boolPtr(true))},
		valueB: &Value{BoolVal: (*Boolean)(boolPtr(false))},
		results: map[Operator]bool{
			OpEq:  false,
			OpGt:  false,
			OpLt:  false,
			OpNeq: true,
			OpGte: false,
			OpLte: false,
		},
	},
}

func TestValueComparison(t *testing.T) {
	for _, cse := range tests {
		for op, value := range cse.results {
			t.Run(fmt.Sprintf("%s %s %s", cse.valueA, op, cse.valueB), func(t *testing.T) {
				assert := assert.New(t)
				res, err := cse.valueA.Compare(cse.valueB, op)
				assert.NoError(err)
				assert.Equal(value, res)
			})

		}
	}
}

type ValueLookup struct {
	returns *Value
	err     error
}

func (v *ValueLookup) GetValue(variableName string) (*Value, error) {
	return v.returns, v.err
}

func TestLookup(t *testing.T) {
	assert := assert.New(t)
	v := &ValueLookup{
		returns: &Value{
			Number: floatPtr(1.0),
		},
	}
	underTest := &Value{
		EnvVar: stringPtr("testEnv"),
	}
	v2, err := underTest.Evaluate(v)
	assert.NoError(err)
	assert.Equal(1.0, *v2.Number)
}
