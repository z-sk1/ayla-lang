package interpreter

import (
	"fmt"
	"math"
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
	TypeInterface
	TypeNamed
)

type TypeInfo struct {
	Name       string
	Kind       TypeKind
	Underlying *TypeInfo
	Alias      bool
	Opaque     bool

	Methods     map[string]*Func
	MethodTypes map[string]*TypeInfo

	Min *float64
	Max *float64

	Fields map[string]*TypeInfo

	Elem *TypeInfo
	Size int

	Key          *TypeInfo
	Value        *TypeInfo
	IsComparable bool

	Variants map[string]int

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
	INTERFACE   ValueType = "interface"
)

type Value interface {
	Type() ValueType
	String() string
}

type Assignable interface {
	Set(*Interpreter, Value) error
	Get(*Interpreter) (Value, error)
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

	expectedTI := UnwrapAlias(i.TypeInfoFromValue(v.Var.Value))

	newVal, err := i.assignWithType(nil, val, expectedTI)
	if err != nil {
		return fmt.Errorf("%s", err.Error())
	}

	v.Var.Value = newVal
	return nil
}

func (v VariableTarget) Get(i *Interpreter) (Value, error) {
	val, ok, _ := i.Env.Get(v.Name)
	if !ok {
		return NilValue{}, fmt.Errorf("undefined variable: %s", v.Name)
	}

	return val, nil
}

type MemberTarget struct {
	Struct    *StructValue
	Field     string
	FieldType *TypeInfo
}

func (m MemberTarget) Set(i *Interpreter, val Value) error {
	newVal, err := i.assignToType(val, m.FieldType)
	if err != nil {
		return fmt.Errorf("field '%s' %s", m.Field, err)
	}
	m.Struct.Fields[m.Field] = newVal
	return nil
}

func (m MemberTarget) Get(i *Interpreter) (Value, error) {
	fieldVar, ok := m.Struct.Fields[m.Field]
	if !ok {
		return NilValue{}, fmt.Errorf("unknown field %s", m.Field)
	}

	return fieldVar, nil
}

type ArrayIndexTarget struct {
	Array    *ArrayValue
	Index    int
	ElemType *TypeInfo
}

func (a ArrayIndexTarget) Set(i *Interpreter, val Value) error {
	if a.Index < 0 || a.Index >= len(a.Array.Elements) {
		return fmt.Errorf("index %d out of bounds", a.Index)
	}

	newVal, err := i.assignToType(val, a.ElemType)
	if err != nil {
		return err
	}

	a.Array.Elements[a.Index] = newVal
	return nil
}

func (a ArrayIndexTarget) Get(i *Interpreter) (Value, error) {
	if a.Index < 0 || a.Index >= len(a.Array.Elements) {
		return NilValue{}, fmt.Errorf("index %d out of bounds", a.Index)
	}

	return a.Array.Elements[a.Index], nil
}

type MapIndexTarget struct {
	Map       *MapValue
	Key       Value
	KeyType   *TypeInfo
	ValueType *TypeInfo
}

func (m MapIndexTarget) Set(i *Interpreter, val Value) error {
	key, err := i.assignToType(m.Key, m.KeyType)
	if err != nil {
		return fmt.Errorf("map key %s", err)
	}

	newVal, err := i.assignToType(val, m.ValueType)
	if err != nil {
		return fmt.Errorf("map value %s", err)
	}

	m.Map.Entries[mapKey(key)] = newVal
	return nil
}

func (m MapIndexTarget) Get(i *Interpreter) (Value, error) {
	if val, ok := m.Map.Entries[mapKey(m.Key)]; ok {
		return val, nil
	}

	return NilValue{}, fmt.Errorf("unknown key: '%s'", m.Key.String())
}

type PointerTarget struct {
	Ptr *PointerValue
}

func (p PointerTarget) Set(i *Interpreter, val Value) error {
	p.Ptr.Target.Value = val
	return nil
}

func (p PointerTarget) Get(i *Interpreter) (Value, error) {
	return p.Ptr.Target.Value, nil
}

func (i *Interpreter) assignToType(val Value, expected *TypeInfo) (Value, error) {
	valType := UnwrapAlias(i.TypeInfoFromValue(val))
	expected = UnwrapAlias(expected)

	if expected.Kind == TypePointer && valType.Kind != TypePointer {
		return nil, fmt.Errorf(
			"type mismatch: expected %s but got %s",
			expected.Name,
			valType.Name,
		)
	}

	if expected.Kind != TypePointer && valType.Kind == TypePointer {
		ptr := val.(*PointerValue)
		if typesAssignable(ptr.ElemType, expected) {
			val = ptr.Target.Value
			valType = UnwrapAlias(i.TypeInfoFromValue(val))
		}
	}

	if !typesAssignable(valType, expected) {
		return nil, fmt.Errorf(
			"type mismatch: expected %s but got %s",
			expected.Name,
			valType.Name,
		)
	}

	if err := validateRange(nil, val, expected); err != nil {
		return nil, err
	}

	return val, nil
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

type UntypedValue struct {
	Value Value
}

func (u UntypedValue) Type() ValueType {
	return u.Value.Type()
}

func (u UntypedValue) String() string {
	return u.Value.String()
}

type IntValue struct {
	V        int
	TypeInfo *TypeInfo
}

func (i IntValue) Type() ValueType {
	return INT
}

func (i IntValue) String() string {
	return fmt.Sprintf("%d", i.V)
}

type FloatValue struct {
	V        float64
	TypeInfo *TypeInfo
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

type InterfaceValue struct {
	TypeInfo *TypeInfo
	Value    Value
}

func (i InterfaceValue) Type() ValueType {
	return INTERFACE
}

func (i InterfaceValue) String() string {
	return i.TypeInfo.Name
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
	Native   any
}

func (s *StructValue) Type() ValueType {
	return STRUCT
}

func (s *StructValue) String() string {
	if s.TypeName == nil {
		return fmt.Sprintf("struct{%v}", s.Fields)
	}

	return fmt.Sprintf("%s{%v}", s.TypeName.Name, s.Fields)
}

type MapValue struct {
	KeyType   *TypeInfo
	ValueType *TypeInfo

	Entries map[string]Value
	Keys    map[string]Value
}

func (m MapValue) Type() ValueType {
	return MAP
}

func (m MapValue) String() string {
	keys := make([]Value, 0, len(m.Entries))

	for _, k := range m.Keys {
		keys = append(keys, k)
	}

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		v := m.Entries[mapKey(k)]
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
	TypeEnv map[string]TypeValue
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

func (i *Interpreter) rangedIntType(min, max *float64) *TypeInfo {
	base := i.TypeEnv["int"].TypeInfo
	return &TypeInfo{
		Name:       fmt.Sprintf("int<%v..%v>", *min, *max),
		Kind:       TypeInt,
		Underlying: base,
		Min:        min,
		Max:        max,
	}
}

func (i *Interpreter) rangedFloatType(min, max *float64) *TypeInfo {
	base := i.TypeEnv["float"].TypeInfo
	return &TypeInfo{
		Name:       fmt.Sprintf("float<%v..%v>", *min, *max),
		Kind:       TypeFloat,
		Underlying: base,
		Min:        min,
		Max:        max,
	}
}

func (i *Interpreter) resolveTypeNode(t parser.TypeNode) (*TypeInfo, error) {
	switch tn := t.(type) {

	case *parser.IdentType:
		// int, string, Person, etc.
		tv, ok := i.TypeEnv[tn.Name.Value]
		if !ok {
			return nil, NewRuntimeError(tn, fmt.Sprintf("unknown type '%s'", tn.Name.Value))
		}
		return tv.TypeInfo, nil

	case *parser.PointerType:
		base, err := i.resolveTypeNode(tn.Base)
		if err != nil {
			return nil, err
		}

		return i.pointerTo(base), nil

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

		minVal = UnwrapUntyped(minVal)
		maxVal = UnwrapUntyped(maxVal)

		var minPtr *float64
		var maxPtr *float64

		var minNum float64
		switch v := minVal.(type) {
		case IntValue:
			minNum = float64(v.V)
		case FloatValue:
			minNum = v.V
		default:
			return nil, NewRuntimeError(tn.Min, fmt.Sprintf("range minimum must be a numeric type, got '%s'", i.TypeInfoFromValue(minVal).Name))
		}
		minPtr = &minNum

		var maxNum float64
		switch v := maxVal.(type) {
		case IntValue:
			maxNum = float64(v.V)
		case FloatValue:
			maxNum = v.V
		default:
			return nil, NewRuntimeError(tn.Max, fmt.Sprintf("range maximum must be a numeric type, got '%s'", i.TypeInfoFromValue(maxVal).Name))
		}
		maxPtr = &maxNum

		if minNum > maxNum {
			return nil, NewRuntimeError(tn, "range minimum cannot be greater than maximum")
		}

		name := fmt.Sprintf("%s<%v..%v>", baseTI.Name, minNum, maxNum)

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

		tv, ok := mod.TypeEnv[tn.Name.Value]
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

		fieldTypes := make([]string, 0)
		for _, f := range fields {
			fieldTypes = append(fieldTypes, f.Name)
		}

		name := fmt.Sprintf("struct{ %s }", strings.Join(fieldTypes, ", "))

		return &TypeInfo{
			Name:   name,
			Kind:   TypeStruct,
			Fields: fields,
		}, nil

	case *parser.InterfaceType:
		methods := make(map[string]*TypeInfo)
		methodNames := make([]string, 0)

		for _, m := range tn.Methods {
			fnType, err := i.resolveTypeNode(m)
			if err != nil {
				return nil, err
			}

			methods[m.Name.Value] = fnType
			methodParams := make([]string, 0)
			methodReturns := make([]string, 0)

			for _, p := range m.Params {
				pType, err := i.resolveTypeNode(p)
				if err != nil {
					return nil, err
				}

				methodParams = append(methodParams, pType.Name)
			}

			if len(m.Returns) > 0 {
				for _, r := range m.Returns {
					rType, err := i.resolveTypeNode(r)
					if err != nil {
						return nil, err
					}

					methodReturns = append(methodReturns, rType.Name)
				}
			}

			var name string

			if len(methodReturns) > 0 {
				name = fmt.Sprintf("%s(%s) (%s)", m.Name.Value, strings.Join(methodParams, ", "), strings.Join(methodReturns, ", "))
			} else {
				name = fmt.Sprintf("%s(%s)", m.Name.Value, strings.Join(methodParams, ", "))
			}

			methodNames = append(methodNames, name)
		}

		name := fmt.Sprintf("interface{ %s }", strings.Join(methodNames, ", "))

		if len(methodNames) == 0 {
			name = "interface{}"
		}

		return &TypeInfo{
			Name:        name,
			Kind:        TypeInterface,
			MethodTypes: methods,
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

		sizeVal = UnwrapFully(sizeVal)

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

			ti = UnwrapAlias(ti)
			paramsTI = append(paramsTI, ti)
			paramsName = append(paramsName, ti.Name)
		}

		for _, typ := range tn.Returns {
			ti, err := i.resolveTypeNode(typ)
			if err != nil {
				return nil, err
			}

			ti = UnwrapAlias(ti)
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

func (i *Interpreter) TypeInfoFromValue(v Value) *TypeInfo {
	switch v := v.(type) {
	case UntypedValue:
		return i.TypeInfoFromValue(v.Value)
	case IntValue:
		if v.TypeInfo != nil {
			return v.TypeInfo
		}

		return i.TypeEnv["int"].TypeInfo
	case FloatValue:
		if v.TypeInfo != nil {
			return v.TypeInfo
		}

		return i.TypeEnv["float"].TypeInfo
	case StringValue:
		return i.TypeEnv["string"].TypeInfo
	case BoolValue:
		return i.TypeEnv["bool"].TypeInfo
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
	case InterfaceValue:
		return v.TypeInfo
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
		return i.TypeEnv["nil"].TypeInfo
	}
}

func (i *Interpreter) defaultValueFromTypeInfo(node parser.Node, ti *TypeInfo) (Value, error) {
	ti = UnwrapAlias(ti)

	switch ti.Kind {
	case TypeInt:
		if ti.Min != nil {
			return IntValue{V: int(*ti.Min)}, nil
		}

		return IntValue{V: 0}, nil
	case TypeFloat:
		if ti.Min != nil {
			return FloatValue{V: *ti.Min}, nil
		}

		return FloatValue{V: 0}, nil
	case TypeString:
		return StringValue{V: ""}, nil
	case TypeBool:
		return BoolValue{V: false}, nil
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
		fields := make(map[string]Value)
		for k, t := range ti.Fields {
			zero, err := i.defaultValueFromTypeInfo(node, t)
			if err != nil {
				return NilValue{}, err
			}

			fields[k] = zero
		}

		return &StructValue{
			TypeName: ti,
			Fields:   fields,
		}, nil
	case TypeMap:
		if ti.Key == nil || ti.Value == nil {
			return NilValue{}, NewRuntimeError(node, "map type missing key type or value type")
		}

		return MapValue{
			Entries:   make(map[string]Value),
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
	case TypeEnum:
		if len(ti.Variants) == 0 {
			return NilValue{}, NewRuntimeError(node, "enum has no variants")
		}

		var (
			name string
			idx  = math.MaxInt
		)

		for n, i := range ti.Variants {
			if i < idx {
				name = n
				idx = i
			}
		}

		return EnumValue{
			Enum:    ti,
			Variant: name,
			Index:   idx,
		}, nil
	case TypePointer:
		return NilValue{}, nil
	case TypeInterface:
		return NilValue{}, nil
	case TypeNamed:
		return i.defaultValueFromTypeInfo(node, ti.Underlying)
	default:
		return NilValue{}, NewRuntimeError(node, "cannot create default value for "+ti.Name)
	}
}

func isComparableValue(v Value) bool {
	v = UnwrapFully(v)

	switch val := v.(type) {
	case IntValue, FloatValue, BoolValue, StringValue, NilValue, *PointerValue:
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
