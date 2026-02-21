package interpreter

import (
	"errors"
	"fmt"
	"sort"
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
	TypeArray
	TypeFunc
	TypeNil
	TypeStruct
	TypeMap
	TypeEnum
	TypeNamed
	TypeError
	TypeAny
)

type TypeInfo struct {
	Name       string
	Kind       TypeKind
	Underlying *TypeInfo
	Alias      bool

	// structs
	Fields map[string]*TypeInfo

	// arrays
	Elem *TypeInfo

	// maps
	Key          *TypeInfo
	Value        *TypeInfo
	IsComparable bool

	// enums
	Variants map[string]int

	// funcs
	Params  []*TypeInfo
	Returns []*TypeInfo
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
	STRUCT_TYPE ValueType = "struct_type"
	STRUCT      ValueType = "struct"
	TUPLE       ValueType = "tuple"
	MAP         ValueType = "map"
	FUNCTION    ValueType = "function"
	ENUM        ValueType = "enum"
	ERROR       ValueType = "error"
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

type Func struct {
	Params   []*parser.ParametersClause
	Body     []parser.Statement
	Env      *Environment
	TypeName *TypeInfo
}

func (f Func) Type() ValueType {
	return FUNCTION
}

func (f Func) String() string {
	return "fun()"
}

type BuiltinFunc struct {
	Name  string
	Arity int
	Fn    func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error)
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

type ErrorValue struct {
	V error
}

func (e ErrorValue) Type() ValueType {
	return ERROR
}

func (e ErrorValue) String() string {
	return e.V.Error()
}

type ArrayValue struct {
	Elements []Value
	ElemType *TypeInfo
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

type MapValue struct {
	Entries   map[Value]Value
	KeyType   *TypeInfo
	ValueType *TypeInfo
}

func (m MapValue) Type() ValueType {
	return MAP
}

func (m MapValue) String() string {
	keys := make([]string, 0, len(m.Entries))
	keyMap := map[string]Value{}

	for k := range m.Entries {
		ks := k.String()
		keys = append(keys, ks)
		keyMap[ks] = k
	}

	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, ks := range keys {
		k := keyMap[ks]
		v := m.Entries[k]
		parts = append(parts, fmt.Sprintf("%s: %s", k.String(), v.String()))
	}

	return fmt.Sprintf("map[%s]%s{%s}", m.KeyType.Name, m.ValueType.Name, strings.Join(parts, ", "))
}

type EnumValue struct {
	Enum    *TypeInfo
	Variant string
	Index   int
}

func (e EnumValue) Type() ValueType {
	return ENUM
}

func (e EnumValue) String() string {
	return fmt.Sprintf("%s.%s", e.Enum.Name, e.Variant)
}

type TypeValue struct {
	TypeName *TypeInfo
}

func (t TypeValue) String() string {
	if t.TypeName.Kind == TypeEnum {
		variants := make([]string, 0, len(t.TypeName.Variants))

		for name := range t.TypeName.Variants {
			variants = append(variants, name)
		}

		return fmt.Sprintf("%s: %s", t.TypeName.Name, strings.Join(variants, ", "))
	}

	return t.TypeName.Name
}

func (t TypeValue) Type() ValueType {
	return valueTypeOf(t.TypeName)
}

type NilValue struct{}

func (n NilValue) Type() ValueType {
	return NIL
}

func (n NilValue) String() string {
	return "nil"
}

type UninitializedValue struct{}

func (u UninitializedValue) Type() ValueType {
	return NIL
}

func (u UninitializedValue) String() string {
	return "nil"
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

	case *parser.ArrayType:
		elemTI, err := i.resolveTypeNode(tn.Elem)
		if err != nil {
			return nil, err
		}

		return &TypeInfo{
			Name: "[]" + elemTI.Name,
			Kind: TypeArray,
			Elem: elemTI,
		}, nil

	case *parser.MapType:
		keyTI, err := i.resolveTypeNode(tn.Key)
		if err != nil {
			return nil, err
		}

		valTI, err := i.resolveTypeNode(tn.Value)
		if err != nil {
			return nil, err
		}

		return &TypeInfo{
			Name:  fmt.Sprintf("map[%s]%s", keyTI.Name, valTI.Name),
			Kind:  TypeMap,
			Key:   keyTI,
			Value: valTI,
		}, nil

	case *parser.FuncType:
		paramsTI := make([]*TypeInfo, 0)
		paramsName := make([]string, 0)
		
		returnsTI := make([]*TypeInfo, 0)
		returnsName := make([]string, 0)

		for _, typ := range tn.Params {
			ti, err := i.resolveTypeNode(typ)
			if err != nil {
				return nil, err
			}

			ti = unwrapAlias(ti)
			paramsTI = append(paramsTI, ti)
			paramsName = append(paramsName, ti.Name)
		}

		for _, typ := range tn.Returns {
			ti, err := i.resolveTypeNode(typ)
			if err != nil {
				return nil, err
			}

			ti = unwrapAlias(ti)
			returnsTI = append(returnsTI, ti)
			returnsName = append(returnsName, ti.Name)
		}
		
		return &TypeInfo{
			Name: fmt.Sprintf("fun(%s) (%s)", strings.Join(paramsName, ", "), strings.Join(returnsName, ", ")),
			Kind: TypeFunc,
			Params: paramsTI,
			Returns: returnsTI,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported type node %T", t)
	}
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
	case TypeArray:
		return ARR
	case TypeEnum:
		return ENUM
	case TypeMap:
		return MAP
	default:
		return NIL
	}
}

func (i *Interpreter) typeInfoFromIdent(id *parser.Identifier) *TypeInfo {
	name := id.Value

	switch name {
	case
		"int",
		"float",
		"string",
		"bool",
		"error",
		"nil":
		return i.typeEnv[name]
	default:
		return i.typeEnv["nil"]
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
	case ErrorValue:
		return i.typeEnv["error"]
	case ArrayValue:
		if v.ElemType == nil {
			panic("ArrayValue ElemType is nil")
		}

		return &TypeInfo{
			Name: "[]" + v.ElemType.Name,
			Kind: TypeArray,
			Elem: v.ElemType,
		}

	case MapValue:
		if v.KeyType == nil || v.ValueType == nil {
			panic("MapValue KeyType or ValueType is nil")
		}

		return &TypeInfo{
			Name:  fmt.Sprintf("map[%s]%s", v.KeyType.Name, v.ValueType.Name),
			Kind:  TypeMap,
			Key:   v.KeyType,
			Value: v.ValueType,
		}

	case *StructValue:
		return v.TypeName
	case *Func:
		return v.TypeName
	case EnumValue:
		return v.Enum
	case NamedValue:
		return v.TypeName
	default:
		return i.typeEnv["nil"]
	}
}

func (i *Interpreter) defaultValueFromTypeInfo(node parser.Statement, ti *TypeInfo) (Value, error) {
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
	case TypeError:
		return ErrorValue{V: errors.New("")}, nil
	case TypeArray:
		if ti.Elem == nil {
			return NilValue{}, NewRuntimeError(node, "array type missing element type")
		}

		return ArrayValue{Elements: make([]Value, 0), ElemType: ti.Elem}, nil
	case TypeStruct:
		return &StructValue{
			TypeName: ti,
			Fields:   map[string]Value{},
		}, nil
	case TypeMap:
		if ti.Key == nil || ti.Value == nil {
			return NilValue{}, NewRuntimeError(node, "map type missing key type or value type")
		}

		return MapValue{
			Entries:   make(map[Value]Value),
			KeyType:   ti.Key,
			ValueType: ti.Value,
		}, nil
	case TypeFunc:
		return &Func{
			Params:   make([]*parser.ParametersClause, 0),
			Body:     make([]parser.Statement, 0),
			Env:      i.env,
			TypeName: ti,
		}, nil
	default:
		return NilValue{}, NewRuntimeError(node, "cannot create default value for "+ti.Name)
	}
}

func isComparableValue(v Value) bool {
	v = unwrapNamed(v)

	switch val := v.(type) {
	case IntValue, FloatValue, BoolValue, StringValue, NilValue:
		return true

	case *StructValue:
		for _, field := range val.Fields {
			if !isComparableValue(field) {
				return false
			}
		}
		return true

	case ArrayValue, MapValue:
		return false

	default:
		return false
	}
}
