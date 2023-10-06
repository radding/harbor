package mathparser

import (
	"fmt"
	"strings"

	"github.com/alecthomas/participle/v2"
)

type Boolean bool

func (b *Boolean) Capture(values []string) error {
	*b = values[0] == "true"
	return nil
}

type Operator int

func (o Operator) String() string {
	switch o {
	case OpAnd:
		return "&&"
	case OpEq:
		return "=="
	case OpNeq:
		return "!="
	case OpGte:
		return ">="
	case OpLte:
		return "<="
	case OpGt:
		return ">"
	case OpLt:
		return "<"
	case OpOr:
		return "||"
	default:
		return "not recognized"
	}
}

const (
	OpGte Operator = iota
	OpEq
	OpLte
	OpGt
	OpLt
	OpNeq
	OpAnd
	OpOr
	OpNot
)

var operatorMap = map[string]Operator{
	"<=": OpLte,
	"==": OpEq,
	">=": OpGte,
	">":  OpGt,
	"<":  OpLt,
	"!=": OpNeq,
	"&&": OpAnd,
	"||": OpOr,
	"!":  OpNot,
}

type ComparisonError struct {
	v1 *Value
	v2 *Value
}

func (c ComparisonError) Error() string {
	return fmt.Sprintf("Can't compare %s with %s", c.v1, c.v2)
}

func (o *Operator) Capture(s []string) error {
	val := strings.Join(s, "")
	*o = operatorMap[val]
	return nil
}

type OpTerm struct {
	LogicalOperator *Operator   `@( "&" "&" | "|" "|" )*`
	RightExpr       *Expression `@@*`
}

func (o *OpTerm) Evaluate(lookup VariableLookUp, leftValue bool) (bool, error) {
	if o.LogicalOperator == nil || o.RightExpr == nil {
		return leftValue, nil
	}
	rightValue, err := o.RightExpr.Evaluate(lookup)
	if err != nil {
		return false, err
	}
	if *o.LogicalOperator == OpAnd {
		return leftValue && rightValue, nil
	}
	return leftValue || rightValue, nil
}

type VariableLookUp interface {
	GetValue(variableName string) (*Value, error)
}

type ComparisonExpression struct {
	Left               *Value    `@@`
	ComparisonOperator *Operator `@( "<" "=" | "=" "=" | ">" "=" | ">" | "<" | "!" "=" )`
	Right              *Value    `@@`
}

func (c *ComparisonExpression) Evaluate(lookup VariableLookUp) (bool, error) {
	leftVal, err := c.Left.Evaluate(lookup)
	if err != nil {
		return false, err
	}
	rightVal, err := c.Right.Evaluate(lookup)
	if err != nil {
		return false, err
	}
	return leftVal.Compare(rightVal, *c.ComparisonOperator)
}

type Expression struct {
	Left   *ComparisonExpression `@@`
	OpTerm *OpTerm               `@@?`
}

func (e *Expression) Evaluate(lookup VariableLookUp) (bool, error) {
	leftVal, err := e.Left.Evaluate(lookup)
	if err != nil {
		return false, err
	}
	if e.OpTerm == nil {
		return leftVal, nil
	}
	return e.OpTerm.Evaluate(lookup, leftVal)

}

var parser = participle.MustBuild[Expression]()

func Parse(str string) (*Expression, error) {
	return parser.ParseString("", str)
}
