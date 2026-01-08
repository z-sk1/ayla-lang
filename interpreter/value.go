package interpreter

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/z-sk1/ayla-lang/parser"
)

type ValueType string

const (
	INT         ValueType = "int"
	FLOAT       ValueType = "float"
	STRING      ValueType = "string"
	BOOL        ValueType = "bool"
	ARRAY       ValueType = "array"
	FUNCTION    ValueType = "function"
	STRUCT_TYPE ValueType = "struct_type"
	STRUCT      ValueType = "struct"
	TUPLE       ValueType = "tuple"
	NIL         ValueType = "nil"
)

type Value interface {
	Type() ValueType
	String() string
}

type ControlSignal interface{}

type SignalNone struct{}
type SignalBreak struct{}
type SignalContinue struct{}

type SignalReturn struct {
	Values []Value
}

type ConstValue struct {
	Value Value
}

func (c ConstValue) Type() ValueType {
	return c.Value.Type()
}

func (c ConstValue) String() string {
	return c.Value.String()
}

type TupleValue struct {
	Values []Value
}

func (t TupleValue) Type() ValueType {
	return TUPLE
}

func (t TupleValue) String() string {
	parts := []string{}
	for _, v := range t.Values {
		parts = append(parts, v.String())
	}
	return fmt.Sprintf("(%s)", strings.Join(parts, ", "))
}

type IntValue struct {
	V int
}

func (i IntValue) Type() ValueType {
	return INT
}

func (i IntValue) String() string {
	return fmt.Sprintf("%d", i.V)
}

type FloatValue struct {
	V float64
}

func (f FloatValue) Type() ValueType {
	return FLOAT
}

func (f FloatValue) String() string {
	return strconv.FormatFloat(f.V, 'f', -1, 64)
}

type StringValue struct {
	V string
}

func (s StringValue) Type() ValueType {
	return STRING
}

func (s StringValue) String() string {
	return s.V
}

type BoolValue struct {
	V bool
}

func (b BoolValue) Type() ValueType {
	return BOOL
}

func (b BoolValue) String() string {
	if b.V {
		return "yes"
	}

	return "no"
}

type ArrayValue struct {
	Elements []Value
}

func (a ArrayValue) Type() ValueType {
	return ARRAY
}

func (a ArrayValue) String() string {
	out := "["
	for i, el := range a.Elements {
		if i > 0 {
			out += ", "
		}
		out += el.String()
	}
	out += "]"
	return out
}

type StructType struct {
	Name   string
	Fields map[string]ValueType
}

func (st StructType) Type() ValueType {
	return STRUCT_TYPE
}

func (st StructType) String() string {
	return "struct type " + st.Name
}

type StructValue struct {
	TypeName *StructType
	Fields   map[string]Value
}

func (s *StructValue) Type() ValueType {
	return STRUCT
}

func (s *StructValue) String() string {
	if s.TypeName == nil {
		return "struct"
	}

	return "struct " + s.TypeName.String()
}

type NilValue struct{}

func (n NilValue) Type() ValueType {
	return NIL
}

func (n NilValue) String() string {
	return "nil"
}

func (i Interpreter) resolveType(expr parser.Expression) (ValueType, error) {
	switch e := expr.(type) {
	case *parser.IntLiteral:
		return INT, nil

	case *parser.FloatLiteral:
		return FLOAT, nil

	case *parser.StringLiteral, *parser.InterpolatedString:
		return STRING, nil

	case *parser.BoolLiteral:
		return BOOL, nil

	case *parser.NilLiteral:
		return NIL, nil

	case *parser.Identifier:
		val, ok := i.env.Get(e.Value)
		if !ok {
			return "", NewRuntimeError(e, fmt.Sprintf("unknown identifier: %s", e.Value))
		}

		return val.Type(), nil

	case *parser.StructLiteral:
		_, ok := i.typeEnv[e.TypeName.Value]
		if !ok {
			return "", NewRuntimeError(e, fmt.Sprintf("unknown struct type: %s", e.TypeName.Value))
		}
		return STRUCT, nil

	case *parser.MemberExpression:
		leftType, err := i.resolveType(e.Left)
		if err != nil {
			return "", err
		}

		if leftType != STRUCT {
			return "", NewRuntimeError(e, fmt.Sprintf("cannot access field type on %s", leftType))
		}

		leftVal, _ := i.EvalExpression(e.Left)
		sv := leftVal.(*StructValue)

		fieldType, ok := sv.TypeName.Fields[e.Field.Value]
		if !ok {
			return "", NewRuntimeError(e, fmt.Sprintf("unknown field: %s", e.Field.Value))
		}

		return fieldType, nil

	case *parser.InfixExpression:
		return i.resolveType(e.Left)

	case *parser.FuncCall:
		return NIL, nil

	default:
		return "", NewRuntimeError(e, fmt.Sprintf("cannot resolve type for %T", expr))
	}
}

func (i Interpreter) resolveTypeFromName(node *parser.StructStatement, name string) (ValueType, error) {
	switch name {
	case "int":
		return INT, nil
	case "float":
		return FLOAT, nil
	case "string":
		return STRING, nil
	case "bool":
		return BOOL, nil
	case "array":
		return ARRAY, nil
	}

	// user-defined struct type
	val, ok := i.env.Get(name)
	if ok {
		if _, ok := val.(StructType); ok {
			return STRUCT_TYPE, nil
		}
	}

	return "", NewRuntimeError(node, fmt.Sprintf("unknown type: %s", name))
}

func valuesEqual(a, b Value) bool {
	if a.Type() != b.Type() {
		return false
	}

	switch av := a.(type) {
	case IntValue:
		return av.V == b.(IntValue).V
	case FloatValue:
		return av.V == b.(FloatValue).V
	case StringValue:
		return av.V == b.(StringValue).V
	case BoolValue:
		return av.V == b.(BoolValue).V
	case NilValue:
		return true
	default:
		return false
	}
}
