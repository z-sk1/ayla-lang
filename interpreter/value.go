package interpreter

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/z-sk1/ayla-lang/parser"
)

type TypeKind int

const (
	TypeInt TypeKind = iota
	TypeFloat
	TypeString
	TypeBool
	TypeNil
	TypeStruct
	TypeNamed
)

type TypeInfo struct {
	Name       string
	Kind       TypeKind
	Underlying *TypeInfo
	Alias      bool
	Fields     map[string]*TypeInfo // for structs
}

type NamedValue struct {
	TypeName *TypeInfo
	Value    Value
}

func (n NamedValue) Type() ValueType {
	return valueTypeOf(n.TypeName)
}

func (n NamedValue) String() string {
	return n.Value.String()
}

type ValueType string

const (
	INT         ValueType = "int"
	FLOAT       ValueType = "float"
	STRING      ValueType = "string"
	BOOL        ValueType = "bool"
	ARR         ValueType = "arr"
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
	return ARR
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
	TypeName *TypeInfo
	Fields   map[string]Value
}

func (s *StructValue) Type() ValueType {
	return STRUCT
}

func (s *StructValue) String() string {
	if s.TypeName == nil {
		return "struct"
	}

	return "struct " + s.TypeName.Name
}

type NilValue struct{}

func (n NilValue) Type() ValueType {
	return NIL
}

func (n NilValue) String() string {
	return "nil"
}

func (i *Interpreter) resolveType(expr parser.Expression) (*TypeInfo, error) {
	switch e := expr.(type) {

	case *parser.IntLiteral:
		return i.typeEnv["int"], nil

	case *parser.FloatLiteral:
		return i.typeEnv["float"], nil

	case *parser.StringLiteral, *parser.InterpolatedString:
		return i.typeEnv["string"], nil

	case *parser.BoolLiteral:
		return i.typeEnv["bool"], nil

	case *parser.NilLiteral:
		return i.typeEnv["nil"], nil

	case *parser.Identifier:
		v, ok := i.env.Get(e.Value)
		if !ok {
			return nil, NewRuntimeError(e, "unknown identifier "+e.Value)
		}
		return i.typeInfoFromValue(v), nil

	case *parser.StructLiteral:
		ti, ok := i.typeEnv[e.TypeName.Value]
		if !ok {
			return nil, NewRuntimeError(e, "unknown struct "+e.TypeName.Value)
		}
		return ti, nil

	case *parser.MemberExpression:
		leftTI, err := i.resolveType(e.Left)
		if err != nil {
			return nil, err
		}

		if leftTI.Kind != TypeStruct {
			return nil, NewRuntimeError(e, "not a struct")
		}

		ft, ok := leftTI.Fields[e.Field.Value]
		if !ok {
			return nil, NewRuntimeError(e, "unknown field "+e.Field.Value)
		}

		return ft, nil

	default:
		return nil, NewRuntimeError(expr, "cannot resolve type")
	}
}

func (i *Interpreter) resolveTypeNode(t parser.TypeNode) (*TypeInfo, error) {
	switch tn := t.(type) {

	case *parser.IdentType:
		// int, string, Person, etc.
		ti, ok := i.typeEnv[tn.Name]
		if !ok {
			return nil, fmt.Errorf("unknown type '%s'", tn.Name)
		}
		return ti, nil

	case *parser.StructType:
		// anonymous struct type
		fields := make(map[string]*TypeInfo)

		for _, f := range tn.Fields {
			ft, err := i.resolveTypeNode(f.Type)
			if err != nil {
				return nil, err
			}
			fields[f.Name.Value] = ft
		}

		return &TypeInfo{
			Name:   "<anon>",
			Kind:   TypeStruct,
			Fields: fields,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported type node %T", t)
	}
}

func (i Interpreter) resolveTypeFromName(node parser.Statement, name string) (ValueType, error) {
	switch name {
	case "int":
		return INT, nil
	case "float":
		return FLOAT, nil
	case "string":
		return STRING, nil
	case "bool":
		return BOOL, nil
	case "arr":
		return ARR, nil
	case "struct":
		return STRUCT, nil

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

func defaultValueFromType(node parser.Statement, typ ValueType) (Value, error) {
	switch typ {
	case INT:
		return IntValue{V: 0}, nil
	case FLOAT:
		return FloatValue{V: 0}, nil
	case STRING:
		return StringValue{V: ""}, nil
	case BOOL:
		return BoolValue{V: false}, nil
	case ARR:
		return ArrayValue{Elements: make([]Value, 0)}, nil
	default:
		return NilValue{}, NewRuntimeError(
			node,
			fmt.Sprintf("unknown type: %s", string(typ)),
		)
	}
}

func defaultValueFromString(node parser.Statement, name string) (Value, error) {
	switch name {
	case "int":
		return IntValue{V: 0}, nil
	case "float":
		return FloatValue{V: 0}, nil
	case "string":
		return StringValue{V: ""}, nil
	case "bool":
		return BoolValue{V: false}, nil
	case "arr":
		return ArrayValue{Elements: make([]Value, 0)}, nil
	default:
		return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown type: %s", name))
	}
}

func valueTypeOf(ti *TypeInfo) ValueType {
	for ti.Kind == TypeNamed {
		ti = ti.Underlying
	}

	switch ti.Kind {
	case TypeInt:
		return INT
	case TypeFloat:
		return FLOAT
	case TypeString:
		return STRING
	case TypeBool:
		return BOOL
	case TypeStruct:
		return STRUCT
	default:
		return NIL
	}
}

func typeKindOf(vt ValueType) TypeKind {
	switch vt {
	case INT:
		return TypeInt
	case FLOAT:
		return TypeFloat
	case STRING:
		return TypeString
	case BOOL:
		return TypeBool
	case STRUCT:
		return TypeStruct
	default:
		return TypeNil
	}
}

func (i *Interpreter) typeInfoFromValue(v Value) *TypeInfo {
	switch v := v.(type) {
	case IntValue:
		return i.typeEnv["int"]
	case FloatValue:
		return i.typeEnv["float"]
	case StringValue:
		return i.typeEnv["string"]
	case BoolValue:
		return i.typeEnv["bool"]
	case *StructValue:
		return v.TypeName
	case NamedValue:
		return v.TypeName

	default:
		return i.typeEnv["nil"]
	}
}

func (i *Interpreter) typeInfoFromIdent(name string) (*TypeInfo, bool) {
	// builtins
	switch name {
	case "int":
		return i.typeEnv["int"], true
	case "float":
		return i.typeEnv["float"], true
	case "string":
		return i.typeEnv["string"], true
	case "bool":
		return i.typeEnv["bool"], true
	}

	// user-defined
	ti, ok := i.typeEnv[name]
	if ok {
		return ti, true
	}

	return nil, false
}

func defaultValueFromTypeInfo(node parser.Statement, ti *TypeInfo) (Value, error) {
	ti = unwrapAlias(ti)

	switch ti.Kind {
	case TypeInt:
		return IntValue{V: 0}, nil
	case TypeFloat:
		return FloatValue{V: 0}, nil
	case TypeString:
		return StringValue{V: ""}, nil
	case TypeBool:
		return BoolValue{V: false}, nil
	case TypeStruct:
		return &StructValue{
			TypeName: ti,
			Fields:   map[string]Value{},
		}, nil
	default:
		return NilValue{}, NewRuntimeError(node, "cannot create default value for "+ti.Name)
	}
}
