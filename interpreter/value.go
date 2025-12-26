package interpreter

import (
	"fmt"
)

type ValueType string

const (
	INT      ValueType = "int"
	FLOAT    ValueType = "float"
	STRING   ValueType = "string"
	BOOL     ValueType = "bool"
	ARRAY    ValueType = "array"
	FUNCTION ValueType = "function"
	NIL      ValueType = "nil"
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
	Value Value
}

type ConstValue struct {
	Value Value
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
	return fmt.Sprintf("%f", f.V)
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

type NilValue struct{}

func (n NilValue) Type() ValueType {
	return NIL
}

func (n NilValue) String() string {
	return "nil"
}
