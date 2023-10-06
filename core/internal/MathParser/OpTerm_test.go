package mathparser

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func opPtr(op Operator) *Operator {
	return &op
}

func createExpression(beTrue bool) *Expression {
	op := OpEq
	if beTrue == false {
		op = OpNeq
	}
	return &Expression{
		Left: &ComparisonExpression{
			Left: &Value{
				Number: floatPtr(1.0),
			},
			ComparisonOperator: opPtr(op),
			Right: &Value{
				Number: floatPtr(1.0),
			},
		},
	}
}

var trueExpr = createExpression(true)

var falseExpr = createExpression(false)

func TestOpTerm(t *testing.T) {
	cases := map[Operator][][]bool{
		OpAnd: {
			{true, true, true},
			{true, false, false},
			{false, false, false},
			{false, true, false},
		},
		OpOr: {
			{true, true, true},
			{true, false, true},
			{false, false, false},
			{false, true, true},
		},
	}
	for operator, table := range cases {
		for _, vals := range table {
			t.Run(fmt.Sprintf("%t %s %t", vals[0], operator, vals[1]), func(t *testing.T) {
				assert := assert.New(t)
				expr := trueExpr
				if vals[1] == false {
					expr = falseExpr
				}
				opTerm := &OpTerm{
					LogicalOperator: &operator,
					RightExpr:       expr,
				}
				res, err := opTerm.Evaluate(nil, vals[0])
				assert.NoError(err)
				assert.Equal(vals[2], res)
			})
		}
	}
	// assert := assert.New(t)
	// opEq := OpEq
	// opAnd := OpAnd
	// opTerm := &OpTerm{
	// 	LogicalOperator: &opAnd,
	// RightExpr: &Expression{
	// 	Left: &ComparisonExpression{
	// 		Left: &Value{
	// 			Number: floatPtr(1.0),
	// 		},
	// 		ComparisonOperator: &opEq,
	// 		Right: &Value{
	// 			Number: floatPtr(1.0),
	// 		},
	// 	},
	// },
	// }

	// res, err := opTerm.Evaluate(nil, true)
	// assert.NoError(err)
	// assert.True(res)
}
