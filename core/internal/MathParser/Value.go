package mathparser

import "fmt"

type VariableDef struct {
	Provider *string `@Ident`
	Value    *string `"." @Ident`
}

func (v *VariableDef) String() string {
	return fmt.Sprintf("%s.%s", *v.Provider, *v.Value)
}

type Value struct {
	Number      *float64     `  @(Float|Int)`
	StringValue *string      `| @(String)`
	BoolVal     *Boolean     `| @("true" | "false")`
	Variable    *VariableDef `| "$" "{" "{" @@ "}" "}"`
}

func (v *Value) String() string {
	if v.Number != nil {
		return fmt.Sprintf("N(%f)", *v.Number)
	}
	if v.StringValue != nil {
		return fmt.Sprintf("S(%s)", *v.StringValue)
	}
	if *v.BoolVal {
		if *v.BoolVal {
			return "B(true)"
		}
		return "B(false)"
	}
	return fmt.Sprintf("E(%s)", v.Variable.String())
}

func (v *Value) valType() string {
	if v.BoolVal != nil {
		return "bool"
	}
	if v.Number != nil {
		return "number"
	}
	if v.StringValue != nil {
		return "string"
	}
	return "env_var"
}

func (v *Value) Compare(v2 *Value, op Operator) (bool, error) {
	if v.valType() != v2.valType() {
		return false, ComparisonError{v1: v, v2: v2}
	}
	switch op {
	case OpNeq:
		switch v.valType() {
		case "number":
			return *v.Number != *v2.Number, nil
		case "bool":
			return bool(*v.BoolVal) != bool(*v2.BoolVal), nil
		default:
			return *v.StringValue != *v2.StringValue, nil
		}
	case OpEq:
		switch v.valType() {
		case "number":
			return *v.Number == *v2.Number, nil
		case "bool":
			return bool(*v.BoolVal) == bool(*v2.BoolVal), nil
		default:
			return *v.StringValue == *v2.StringValue, nil
		}

	case OpGt:
		switch v.valType() {
		case "number":
			return *v.Number > *v2.Number, nil
		case "bool":
			return false, nil
		default:
			return *v.StringValue > *v2.StringValue, nil
		}
	case OpLt:
		switch v.valType() {
		case "number":
			return *v.Number < *v2.Number, nil
		case "bool":
			return false, nil
		default:
			return *v.StringValue < *v2.StringValue, nil
		}
	case OpGte:
		switch v.valType() {
		case "number":
			return *v.Number >= *v2.Number, nil
		case "bool":
			return false, nil
		default:
			return *v.StringValue >= *v2.StringValue, nil
		}
	case OpLte:
		switch v.valType() {
		case "number":
			return *v.Number <= *v2.Number, nil
		case "bool":
			return false, nil
		default:
			return *v.StringValue <= *v2.StringValue, nil
		}
	}
	return false, fmt.Errorf("unrecognized operator")
}

func (v *Value) Evaluate(lookup VariableLookUp) (*Value, error) {
	if v.Variable != nil {
		return lookup.GetValue(*v.Variable.Provider, *v.Variable.Value)
	}
	return v, nil
}
