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
	TypeFixedArray
	TypePointer
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

	// boundaries
	Min *float64
	Max *float64

	// structs
	Fields map[string]*TypeInfo

	// arrays
	Elem *TypeInfo // arrays and pointers
	Size int

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
	MODULE      ValueType = "module"
	NATIVE      ValueType = "native"
	POINTER     ValueType = "pointer"
)

type Value interface {
	Type() ValueType
	String() string
}

type Assignable interface {
	Set(*Interpreter, Value) error
}

type VariableTarget struct {
	Name string
	Var  *Variable
}

func (v VariableTarget) Set(i *Interpreter, val Value) error {
	if v.Var.isConst {
		return fmt.Errorf("cannot assign to const: %s", v.Name)
	}

	switch v.Var.Value.(type) {
	case UninitializedValue:
		v.Var.Value = val
		return nil
	}

	expectedTI := unwrapAlias(i.typeInfoFromValue(v.Var.Value))

	newVal, err := i.assignWithType(nil, val, expectedTI)
	if err != nil {
		return err
	}

	v.Var.Value = newVal
	return nil
}

type MemberTarget struct {
	Struct    *StructValue
	Field     string
	FieldType *TypeInfo
}

func (m MemberTarget) Set(i *Interpreter, val Value) error {
	valType := unwrapAlias(i.typeInfoFromValue(val))

	if !typesAssignable(valType, m.FieldType) {
		return fmt.Errorf(
			"field '%s' expects %s but got %s",
			m.Field,
			m.FieldType.Name,
			valType.Name,
		)
	}

	if err := validateRange(nil, val, m.FieldType); err != nil {
		return err
	}

	m.Struct.Fields[m.Field] = val
	return nil
}

type ArrayIndexTarget struct {
	Array    *ArrayValue
	Index    int
	ElemType *TypeInfo
}

func (t ArrayIndexTarget) Set(i *Interpreter, val Value) error {
	if t.Index < 0 || t.Index >= len(t.Array.Elements) {
		return fmt.Errorf("index %d out of bounds", t.Index)
	}

	valType := unwrapAlias(i.typeInfoFromValue(val))

	if t.ElemType.Kind != TypeAny {
		if !typesAssignable(valType, t.ElemType) {
			return fmt.Errorf(
				"array element expected %s but got %s",
				t.ElemType.Name,
				valType.Name,
			)
		}

		if err := validateRange(nil, val, t.ElemType); err != nil {
			return err
		}
	}

	t.Array.Elements[t.Index] = val
	return nil
}

type MapIndexTarget struct {
	Map       *MapValue
	Key       Value
	KeyType   *TypeInfo
	ValueType *TypeInfo
}

func (t MapIndexTarget) Set(i *Interpreter, val Value) error {
	keyType := unwrapAlias(i.typeInfoFromValue(t.Key))

	if t.KeyType.Kind == TypeAny {
		if !isComparableValue(t.Key) {
			return fmt.Errorf("value of this type cannot be used as map key")
		}
	} else {
		if !typesAssignable(keyType, t.KeyType) {
			return fmt.Errorf(
				"map index expected %s but got %s",
				t.KeyType.Name,
				keyType.Name,
			)
		}

		if err := validateRange(nil, t.Key, t.KeyType); err != nil {
			return err
		}
	}

	valType := unwrapAlias(i.typeInfoFromValue(val))

	if t.ValueType.Kind != TypeAny {
		if !typesAssignable(valType, t.ValueType) {
			return fmt.Errorf(
				"map value expected %s but got %s",
				t.ValueType.Name,
				valType.Name,
			)
		}

		if err := validateRange(nil, val, t.ValueType); err != nil {
			return err
		}
	}

	t.Map.Entries[t.Key] = val
	return nil
}

type PointerTarget struct {
	Ptr *PointerValue
}

func (p PointerTarget) Set(i *Interpreter, val Value) error {
	p.Ptr.Target.Value = val
	return nil
}

type ControlSignal any

type SignalNone struct{}
type SignalBreak struct{}
type SignalContinue struct{}

type SignalReturn struct {
	Values []Value
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
	Params   []*parser.Param
	Body     []parser.Statement
	Env      *Environment
	TypeName *TypeInfo
}

func (f Func) Type() ValueType {
	return FUNCTION
}

func (f Func) String() string {
	return f.TypeName.Name
}

type BoundMethodValue struct {
	Receiver Value
	Func     *Func
}

func (b BoundMethodValue) Type() ValueType {
	return valueTypeOf(b.Func.TypeName)
}

func (b BoundMethodValue) String() string {
	parts := strings.Split(b.Func.TypeName.Name, "(")
	parts[0] += fmt.Sprintf("(%s)", b.Receiver.String())

	return strings.Join(parts, " ")
}

type BuiltinFunc struct {
	Name  string
	Arity int
	Fn    func(i *Interpreter, node *parser.FuncCall, args []Value) (Value, error)
}

func (b BuiltinFunc) Type() ValueType {
	return FUNCTION
}

func (b BuiltinFunc) String() string {
	return fmt.Sprintf("%s()", b.Name)
}

type PointerValue struct {
	Target   *Variable
	ElemType *TypeInfo
}

func (p *PointerValue) Type() ValueType {
	return POINTER
}

func (p *PointerValue) String() string {
	if p.Target == nil {
		return "ptr(nil)"
	}
	return fmt.Sprintf("ptr(%p -> %v)", p.Target, p.Target.Value)
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
	Capacity int
	Fixed    bool
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

	return fmt.Sprintf("map{%s}", strings.Join(parts, ", "))
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
	TypeInfo *TypeInfo
}

func (t TypeValue) String() string {
	if t.TypeInfo.Kind == TypeEnum {
		variants := make([]string, 0, len(t.TypeInfo.Variants))

		for name := range t.TypeInfo.Variants {
			variants = append(variants, name)
		}

		return fmt.Sprintf("%s: %s", t.TypeInfo.Name, strings.Join(variants, ", "))
	}

	return t.TypeInfo.Name
}

func (t TypeValue) Type() ValueType {
	return valueTypeOf(t.TypeInfo)
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

type ModuleValue struct {
	Name    string
	Env     *Environment
	typeEnv map[string]TypeValue
}

func (m ModuleValue) Type() ValueType {
	return MODULE
}

func (m ModuleValue) String() string {
	return fmt.Sprintf("<module %s>", m.Name)
}

type NativeValue struct {
	V any
}

func (n NativeValue) Type() ValueType {
	return NATIVE
}

func (n NativeValue) String() string {
	return "native"
}

func (i *Interpreter) resolveTypeNode(t parser.TypeNode) (*TypeInfo, error) {
	switch tn := t.(type) {

	case *parser.IdentType:
		// int, string, Person, etc.
		tv, ok := i.typeEnv[tn.Name.Value]
		if !ok {
			return nil, NewRuntimeError(tn, fmt.Sprintf("unknown type '%s'", tn.Name.Value))
		}
		return tv.TypeInfo, nil

	case *parser.PointerType:
		base, err := i.resolveTypeNode(tn.Base)
		if err != nil {
			return nil, err
		}

		return &TypeInfo{
			Name: fmt.Sprintf("*%s", base.Name),
			Kind: TypePointer,
			Elem: base,
		}, nil

	case *parser.RangeType:
		baseTI, err := i.resolveTypeNode(tn.Base)
		if err != nil {
			return nil, err
		}

		minVal, err := i.EvalExpression(tn.Min)
		if err != nil {
			return nil, err
		}

		maxVal, err := i.EvalExpression(tn.Max)
		if err != nil {
			return nil, err
		}

		var minPtr *float64
		var maxPtr *float64

		var minNum float64
		switch v := minVal.(type) {
		case IntValue:
			minNum = float64(v.V)
		case FloatValue:
			minNum = v.V
		default:
			return nil, NewRuntimeError(tn.Min, "range minimum must be numeric")
		}
		minPtr = &minNum

		var maxNum float64
		switch v := maxVal.(type) {
		case IntValue:
			maxNum = float64(v.V)
		case FloatValue:
			maxNum = v.V
		default:
			return nil, NewRuntimeError(tn.Max, "range maximum must be numeric")
		}
		maxPtr = &maxNum

		if minNum > maxNum {
			return nil, NewRuntimeError(tn, "range minimum cannot be greater than maximum")
		}

		name := fmt.Sprintf("%s[%v..%v]", baseTI.Name, minNum, maxNum)

		return &TypeInfo{
			Name:       name,
			Kind:       baseTI.Kind,
			Underlying: baseTI,
			Min:        minPtr,
			Max:        maxPtr,
		}, nil

	case *parser.QualifiedType:
		modVal, ok, _ := i.Env.Get(tn.Module.Value)
		if !ok {
			return nil, NewRuntimeError(tn, fmt.Sprintf("unknown module '%s'", tn.Module.Value))
		}

		mod, ok := modVal.(ModuleValue)
		if !ok {
			return nil, NewRuntimeError(tn, fmt.Sprintf("'%s' is not a module", tn.Module.Value))
		}

		tv, ok := mod.typeEnv[tn.Name.Value]
		if !ok {
			return nil, NewRuntimeError(tn,
				fmt.Sprintf("module '%s' has no type '%s'", tn.Module.Value, tn.Name.Value))
		}

		return tv.TypeInfo, nil

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

		if tn.Size == nil {
			return &TypeInfo{
				Name: "[]" + elemTI.Name,
				Kind: TypeArray,
				Elem: elemTI,
			}, nil
		}

		sizeVal, err := i.EvalExpression(tn.Size)
		if err != nil {
			return nil, err
		}

		intSize, ok := sizeVal.(IntValue)
		if !ok {
			return nil, NewRuntimeError(tn, "array size must be int")
		}

		return &TypeInfo{
			Name: fmt.Sprintf("[%d]%s", intSize.V, elemTI.Name),
			Kind: TypeFixedArray,
			Elem: elemTI,
			Size: intSize.V,
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
			Name:    fmt.Sprintf("fun(%s) (%s)", strings.Join(paramsName, ", "), strings.Join(returnsName, ", ")),
			Kind:    TypeFunc,
			Params:  paramsTI,
			Returns: returnsTI,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported type node %T", t)
	}
}

func valuesEqual(a, b Value) bool {
	switch av := a.(type) {

	case IntValue:
		bv, ok := b.(IntValue)
		return ok && av.V == bv.V

	case FloatValue:
		bv, ok := b.(FloatValue)
		return ok && av.V == bv.V

	case StringValue:
		bv, ok := b.(StringValue)
		return ok && av.V == bv.V

	case BoolValue:
		bv, ok := b.(BoolValue)
		return ok && av.V == bv.V

	case EnumValue:
		switch bv := b.(type) {
		case EnumValue:
			return av.Index == bv.Index
		case IntValue:
			return av.Index == bv.V
		default:
			return false
		}

	case *PointerValue:
		bv, ok := b.(*PointerValue)
		return ok && av.Target == bv.Target

	case NilValue:
		_, ok := b.(NilValue)
		return ok

	default:
		return false
	}
}

func runtimeKind(ti *TypeInfo) TypeKind {
	if ti.Kind == TypeNamed {
		return ti.Underlying.Kind
	}
	return ti.Kind
}

func valueTypeOf(ti *TypeInfo) ValueType {
	switch runtimeKind(ti) {
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
	case TypeFunc:
		return FUNCTION
	default:
		return NIL
	}
}

func (i *Interpreter) typeInfoFromValue(v Value) *TypeInfo {
	switch v := v.(type) {
	case IntValue:
		return i.typeEnv["int"].TypeInfo
	case FloatValue:
		return i.typeEnv["float"].TypeInfo
	case StringValue:
		return i.typeEnv["string"].TypeInfo
	case BoolValue:
		return i.typeEnv["bool"].TypeInfo
	case ErrorValue:
		return i.typeEnv["error"].TypeInfo
	case ArrayValue:
		if v.Fixed {
			return &TypeInfo{
				Name: fmt.Sprintf("[%d]%s", v.Capacity, v.ElemType.Name),
				Kind: TypeFixedArray,
				Elem: v.ElemType,
				Size: v.Capacity,
			}
		}

		return &TypeInfo{
			Name: fmt.Sprintf("[]%s", v.ElemType.Name),
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
	case *PointerValue:
		if v.ElemType == nil {
			panic("PointerValue ElemType is nil")
		}
		return i.pointerTo(v.ElemType)
	default:
		return i.typeEnv["nil"].TypeInfo
	}
}

func (i *Interpreter) defaultValueFromTypeInfo(node parser.Node, ti *TypeInfo) (Value, error) {
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
	case TypeFixedArray:
		if ti.Elem == nil {
			return NilValue{}, NewRuntimeError(node, "array type missing element type")
		}

		return ArrayValue{Elements: make([]Value, ti.Size), ElemType: ti.Elem, Capacity: ti.Size, Fixed: true}, nil
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
			Params:   make([]*parser.Param, 0),
			Body:     make([]parser.Statement, 0),
			Env:      i.Env,
			TypeName: ti,
		}, nil
	case TypePointer:
		return &PointerValue{
			Target:   nil,
			ElemType: ti,
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
