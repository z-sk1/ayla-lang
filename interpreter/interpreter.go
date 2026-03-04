package interpreter

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"os"
	"sync"

	"github.com/z-sk1/ayla-lang/lexer"
	"github.com/z-sk1/ayla-lang/parser"
)

type Environment struct {
	store    map[string]Variable
	methods  map[string]map[string]*Func
	builtins map[string]*BuiltinFunc
	defers   []*parser.FuncCall

	mu     sync.RWMutex
	parent *Environment
}

type Interpreter struct {
	Env           *Environment
	typeEnv       map[string]*TypeInfo
	modules       map[string]ModuleValue
	nativeModules map[string]NativeLoader
	modulePaths   []string
	currentDir    string
	projectRoot   string
}

type RuntimeError struct {
	Message string
	Line    int
	Column  int
}

func (e RuntimeError) Error() string {
	return fmt.Sprintf("runtime error at %d:%d: %s\n", e.Line, e.Column, e.Message)
}

type Variable struct {
	Value    Value
	Lifetime int
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (i *Interpreter) resolveModule(name string) (string, error) {
	exts := []string{".ayla", ".ayl"}

	wd, _ := os.Getwd()

	searchPaths := []string{
		i.currentDir,
		filepath.Join(i.currentDir, name),
		filepath.Join(i.currentDir, "lib"),
		filepath.Join(i.currentDir, "lib", name),
		filepath.Join(wd, "lib", name),
	}

	searchPaths = append(searchPaths, i.modulePaths...)

	for _, base := range searchPaths {
		for _, ext := range exts {
			path := filepath.Join(base, name+ext)
			if fileExists(path) {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("module '%s' not found", name)
}

func (i *Interpreter) loadModule(name string) (Value, error) {
	if mod, ok := i.modules[name]; ok {
		return mod, nil
	}

	if loader, ok := i.nativeModules[name]; ok {
		mod, err := loader(i)
		if err != nil {
			return NilValue{}, err
		}

		i.modules[name] = mod
		i.Env.Define(name, mod)
		return mod, nil
	}

	path, err := i.resolveModule(name)
	if err != nil {
		return NilValue{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return NilValue{}, err
	}
	src := string(data)

	l := lexer.New(src)
	p := parser.New(l)
	program := p.ParseProgram()

	Env := NewEnvironment(i.Env)

	modInterp := NewWithEnv(Env, path)
	modInterp.currentDir = filepath.Dir(path)

	_, err = modInterp.EvalStatements(program)
	if err != nil {
		return NilValue{}, err
	}

	module := ModuleValue{
		Name: name,
		Env:  Env,
	}

	i.Env.Define(name, module)

	return module, nil
}

func (i *Interpreter) EvalStatements(stmts []parser.Statement) (ControlSignal, error) {
	for _, s := range stmts {
		sig, err := i.EvalStatement(s)
		if err != nil {
			return SignalNone{}, err
		}

		switch sig.(type) {
		case SignalReturn, SignalBreak, SignalContinue, ErrorValue:
			return sig, nil
		}

		i.tickLifetimes()
	}
	return SignalNone{}, nil
}

func (i *Interpreter) EvalBlock(stmts []parser.Statement, newScope bool) (ControlSignal, error) {
	blockEnv := NewEnvironment(i.Env)
	oldEnv := i.Env

	if newScope {
		i.Env = blockEnv
	}

	sig, err := i.EvalStatements(stmts)

	i.Env = oldEnv
	return sig, err
}

func (i *Interpreter) EvalStatement(s parser.Statement) (ControlSignal, error) {
	if s == nil {
		return SignalNone{}, nil
	}

	switch stmt := s.(type) {
	case *parser.VarStatement:
		var val Value
		var err error

		var expectedTI *TypeInfo
		if stmt.Type != nil {
			expectedTI, err = i.resolveTypeNode(stmt.Type)
			if err != nil {
				return SignalNone{}, err
			}
			expectedTI = unwrapAlias(expectedTI)
		}

		if stmt.Value != nil {
			val, err = i.EvalExpression(stmt.Value)

			if tup, ok := val.(TupleValue); ok {
				if len(tup.Values) > 1 {
					return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("expected 1 value but got %d", len(tup.Values)))
				}
			}
		} else if expectedTI != nil {
			val, err = i.defaultValueFromTypeInfo(stmt, expectedTI)
			if err != nil {
				return SignalNone{}, err
			}
		} else {
			val = UninitializedValue{}
		}

		if err != nil {
			return SignalNone{}, err
		}
		if v, ok := val.(ErrorValue); ok {
			return v, nil
		}

		val, err = i.assignWithType(stmt, val, expectedTI)
		if err != nil {
			return SignalNone{}, err
		}

		if stmt.Lifetime != nil {
			lifetime, err := i.EvalExpression(stmt.Lifetime)
			if err != nil {
				return SignalNone{}, err
			}

			if lifetime.(IntValue).V > 0 {
				i.Env.DefineWithLifetime(stmt.Name.Value, val, lifetime.(IntValue).V+1) // +1 because the var statement itself also decrements it
				return SignalNone{}, nil
			}
		}

		if stmt.Name.Value == "_" {
			return SignalNone{}, nil
		}

		i.Env.Define(stmt.Name.Value, val)
		return SignalNone{}, nil

	case *parser.VarStatementBlock:
		for _, decl := range stmt.Decls {
			i.EvalStatement(decl)
		}

		return SignalNone{}, nil

	case *parser.VarStatementNoKeyword:
		val, err := i.EvalExpression(stmt.Value)
		if err != nil {
			return NilValue{}, err
		}

		if tup, ok := val.(TupleValue); ok {
			if len(tup.Values) > 1 {
				return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("expected 1 value but got %d", len(tup.Values)))
			}
		}

		// variable must not exist
		if _, ok := i.Env.Get(stmt.Name.Value); ok {
			return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("cant redeclare var: %s", stmt.Name.Value))
		}

		if stmt.Lifetime != nil {
			lifetime, err := i.EvalExpression(stmt.Lifetime)
			if err != nil {
				return SignalNone{}, err
			}

			if lifetime.(IntValue).V > 0 {
				i.Env.DefineWithLifetime(stmt.Name.Value, val, lifetime.(IntValue).V+1) // +1 because the var statement itself also decrements it
				return SignalNone{}, nil
			}
		}

		if stmt.Name.Value == "_" {
			return SignalNone{}, nil
		}

		i.Env.Define(stmt.Name.Value, val)
		return SignalNone{}, nil

	case *parser.MultiVarStatement:
		if stmt.Values == nil {
			var expectedTI *TypeInfo
			var err error

			if stmt.Type != nil {
				expectedTI, err = i.resolveTypeNode(stmt.Type)
				if err != nil {
					return SignalNone{}, err
				}
			}

			for _, name := range stmt.Names {

				if _, ok := i.Env.Get(name.Value); ok {
					return SignalNone{}, NewRuntimeError(stmt,
						fmt.Sprintf("cannot redeclare var: %s", name.Value))
				}

				var v Value
				if expectedTI != nil {
					v, err = i.defaultValueFromTypeInfo(stmt, expectedTI)
					if err != nil {
						return SignalNone{}, err
					}
				} else {
					v = UninitializedValue{}
				}

				if stmt.Lifetime != nil {
					lifetime, err := i.EvalExpression(stmt.Lifetime)
					if err != nil {
						return SignalNone{}, err
					}

					if lifetime.(IntValue).V > 0 {
						i.Env.DefineWithLifetime(name.Value, v, lifetime.(IntValue).V+1) // +1 because the var statement itself also decrements it
					}
				} else {
					i.Env.Define(name.Value, v)
				}
			}

			return SignalNone{}, nil
		}

		var values []Value

		if len(stmt.Values) == 1 {
			val, err := i.EvalExpression(stmt.Values[0])
			if err != nil {
				return SignalNone{}, err
			}
			if v, ok := val.(ErrorValue); ok {
				return v, nil
			}

			if tup, ok := val.(TupleValue); ok {
				values = tup.Values
			} else {
				return SignalNone{}, NewRuntimeError(stmt, "multi assign expected multiple values")
			}
		} else {
			values = make([]Value, 0, len(stmt.Values))

			for idx, expr := range stmt.Values {
				v, err := i.EvalExpression(expr)
				if err != nil {
					return SignalNone{}, err
				}
				if v, ok := v.(ErrorValue); ok {
					return v, nil
				}

				values[idx] = v
			}
		}

		if len(values) != len(stmt.Names) {
			return SignalNone{}, NewRuntimeError(stmt,
				fmt.Sprintf("expected %d values, got %d",
					len(stmt.Names), len(stmt.Values)))
		}

		var expectedTI *TypeInfo
		var err error
		if stmt.Type != nil {
			expectedTI, err = i.resolveTypeNode(stmt.Type)
			if err != nil {
				return SignalNone{}, err
			}
		}

		for idx, name := range stmt.Names {
			if name.Value == "_" {
				continue
			}

			if _, ok := i.Env.Get(name.Value); ok {
				return SignalNone{}, NewRuntimeError(stmt,
					fmt.Sprintf("cannot redeclare var: %s", name.Value))
			}

			v, err := i.assignWithType(stmt, values[idx], expectedTI)
			if err != nil {
				return SignalNone{}, err
			}

			if stmt.Lifetime != nil {
				lifetimeVal, err := i.EvalExpression(stmt.Lifetime)
				if err != nil {
					return SignalNone{}, err
				}

				lifetime := lifetimeVal.(IntValue).V
				if lifetime > 0 {
					i.Env.DefineWithLifetime(name.Value, v, lifetime+1)
				}
			} else {
				i.Env.Define(name.Value, v)
			}
		}

	case *parser.MultiVarStatementNoKeyword:
		var values []Value

		if len(stmt.Values) == 1 {

			val, err := i.EvalExpression(stmt.Values[0])
			if err != nil {
				return SignalNone{}, err
			}

			if v, ok := val.(ErrorValue); ok {
				return v, nil
			}

			if tup, ok := val.(TupleValue); ok {
				values = tup.Values
			} else {
				values = []Value{val}
			}

		} else {

			values = make([]Value, len(stmt.Values))

			for idx, expr := range stmt.Values {
				v, err := i.EvalExpression(expr)
				if err != nil {
					return SignalNone{}, err
				}

				if v, ok := v.(ErrorValue); ok {
					return v, nil
				}

				values[idx] = v
			}
		}

		if len(values) != len(stmt.Names) {
			return SignalNone{}, NewRuntimeError(
				stmt,
				fmt.Sprintf("expected %d values, got %d",
					len(stmt.Names), len(values)),
			)
		}
		for idx, name := range stmt.Names {
			if name.Value == "_" {
				continue
			}

			if _, ok := i.Env.Get(name.Value); ok {
				return SignalNone{}, NewRuntimeError(stmt,
					fmt.Sprintf("cannot redeclare var: %s", name.Value))
			}

			if stmt.Lifetime != nil {
				lifetimeVal, err := i.EvalExpression(stmt.Lifetime)
				if err != nil {
					return SignalNone{}, err
				}

				lifetime := lifetimeVal.(IntValue).V
				if lifetime > 0 {
					i.Env.DefineWithLifetime(name.Value, values[idx], lifetime+1)
				}
			} else {
				i.Env.Define(name.Value, values[idx])
			}
		}

		return SignalNone{}, nil

	case *parser.ConstStatementBlock:
		for _, decl := range stmt.Decls {
			i.EvalStatement(decl)
		}

		return SignalNone{}, nil

	case *parser.ConstStatement:
		var val Value
		var err error

		var expectedTI *TypeInfo
		if stmt.Type != nil {
			expectedTI, err = i.resolveTypeNode(stmt.Type)
			if err != nil {
				return SignalNone{}, err
			}
		}

		if stmt.Value != nil {
			val, err = i.EvalExpression(stmt.Value)

			if tup, ok := val.(TupleValue); ok {
				if len(tup.Values) > 1 {
					return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("expected 1 value but got %d", len(tup.Values)))
				}
			}
		} else if expectedTI != nil {
			val, err = i.defaultValueFromTypeInfo(stmt, expectedTI)
			if err != nil {
				return SignalNone{}, err
			}
		} else {
			return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("const %s must be initalised with a value", stmt.Name.Value))
		}

		// check if variable already exist
		if _, ok := i.Env.Get(stmt.Name.Value); ok {
			return SignalNone{}, NewRuntimeError(s, fmt.Sprintf("cant redeclare const: %s", stmt.Name.Value))
		}

		val, err = i.assignWithType(stmt, val, expectedTI)
		if err != nil {
			return SignalNone{}, err
		}

		if stmt.Lifetime != nil {
			lifetime, err := i.EvalExpression(stmt.Lifetime)
			if err != nil {
				return SignalNone{}, err
			}

			if lifetime.(IntValue).V > 0 {
				i.Env.DefineWithLifetime(stmt.Name.Value, ConstValue{Value: val}, lifetime.(IntValue).V+1) // +1 because the var statement itself also decrements it
				return SignalNone{}, nil
			}
		}

		// store const val
		i.Env.Define(stmt.Name.Value, ConstValue{Value: val})
		return SignalNone{}, nil

	case *parser.MultiConstStatement:
		if stmt.Values == nil {
			var names string

			for _, name := range stmt.Names {
				if name == stmt.Names[len(stmt.Names)-1] {
					names = names + name.Value
				} else {
					names = names + (name.Value + ", ")
				}
			}

			return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("constants, %s, must be initialised", names))
		}

		if stmt.Values == nil {
			var expectedTI *TypeInfo
			var err error

			if stmt.Type != nil {
				expectedTI, err = i.resolveTypeNode(stmt.Type)
				if err != nil {
					return SignalNone{}, err
				}
			}

			for _, name := range stmt.Names {

				if _, ok := i.Env.Get(name.Value); ok {
					return SignalNone{}, NewRuntimeError(stmt,
						fmt.Sprintf("cannot redeclare var: %s", name.Value))
				}

				var v Value
				if expectedTI != nil {
					v, err = i.defaultValueFromTypeInfo(stmt, expectedTI)
					if err != nil {
						return SignalNone{}, err
					}
				} else {
					v = UninitializedValue{}
				}

				if stmt.Lifetime != nil {
					lifetime, err := i.EvalExpression(stmt.Lifetime)
					if err != nil {
						return SignalNone{}, err
					}

					if lifetime.(IntValue).V > 0 {
						i.Env.DefineWithLifetime(name.Value, v, lifetime.(IntValue).V+1) // +1 because the var statement itself also decrements it
					}
				} else {
					i.Env.Define(name.Value, v)
				}
			}

			return SignalNone{}, nil
		}

		var values []Value

		if len(stmt.Values) == 1 {
			val, err := i.EvalExpression(stmt.Values[0])
			if err != nil {
				return SignalNone{}, err
			}
			if v, ok := val.(ErrorValue); ok {
				return v, nil
			}

			if tup, ok := val.(TupleValue); ok {
				values = tup.Values
			} else {
				return SignalNone{}, NewRuntimeError(stmt, "multi assign expected multiple values")
			}
		} else {
			values = make([]Value, 0, len(stmt.Values))

			for idx, expr := range stmt.Values {
				v, err := i.EvalExpression(expr)
				if err != nil {
					return SignalNone{}, err
				}
				if v, ok := v.(ErrorValue); ok {
					return v, nil
				}

				values[idx] = v
			}
		}

		if len(values) != len(stmt.Names) {
			return SignalNone{}, NewRuntimeError(stmt,
				fmt.Sprintf("expected %d values, got %d",
					len(stmt.Names), len(stmt.Values)))
		}

		var expectedTI *TypeInfo
		var err error
		if stmt.Type != nil {
			expectedTI, err = i.resolveTypeNode(stmt.Type)
			if err != nil {
				return SignalNone{}, err
			}
		}

		for idx, name := range stmt.Names {
			if name.Value == "_" {
				continue
			}

			if _, ok := i.Env.Get(name.Value); ok {
				return SignalNone{}, NewRuntimeError(stmt,
					fmt.Sprintf("cannot redeclare var: %s", name.Value))
			}

			v, err := i.assignWithType(stmt, values[idx], expectedTI)
			if err != nil {
				return SignalNone{}, err
			}

			if stmt.Lifetime != nil {
				lifetimeVal, err := i.EvalExpression(stmt.Lifetime)
				if err != nil {
					return SignalNone{}, err
				}

				lifetime := lifetimeVal.(IntValue).V
				if lifetime > 0 {
					i.Env.DefineWithLifetime(name.Value, ConstValue{Value: v}, lifetime+1)
				}
			} else {
				i.Env.Define(name.Value, ConstValue{Value: v})
			}
		}

		return SignalNone{}, nil

	case *parser.ImportStatement:
		val, err := i.loadModule(stmt.Name)
		if err != nil {
			return NilValue{}, NewRuntimeError(stmt, err.Error())
		}

		return val, nil

	case *parser.TypeStatement:
		underlying, err := i.resolveTypeNode(stmt.Type)
		if err != nil {
			return SignalNone{}, NewRuntimeError(stmt, err.Error())
		}

		ti := &TypeInfo{
			Name:       stmt.Name.Value,
			Kind:       TypeNamed,
			Underlying: underlying,
		}

		if stmt.Alias {
			ti.Alias = true
			ti.Kind = underlying.Kind
		}

		i.typeEnv[stmt.Name.Value] = ti

		return SignalNone{}, nil
	case *parser.AssignmentStatement:
		val, err := i.EvalExpression(stmt.Value)
		if err != nil {
			return SignalNone{}, err
		}
		if v, ok := val.(ErrorValue); ok {
			return v, nil
		}

		existingVal, ok := i.Env.Get(stmt.Name.Value)
		if !ok {
			return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("assignment to undefined variable: %s", stmt.Name.Value))
		}

		if _, isConst := existingVal.(ConstValue); isConst {
			return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("cannot reassign to const: %s", stmt.Name.Value))
		}

		switch existingVal.(type) {
		case UninitializedValue:
			i.Env.Set(stmt.Name.Value, val)
			return SignalNone{}, nil
		}

		expectedTI := unwrapAlias(i.typeInfoFromValue(existingVal))

		v, err := i.assignWithType(stmt, val, expectedTI)
		if err != nil {
			return SignalNone{}, err
		}

		i.Env.Set(stmt.Name.Value, v)
		return SignalNone{}, nil

	case *parser.MultiAssignmentStatement:

		var values []Value

		if len(stmt.Values) == 1 {
			val, err := i.EvalExpression(stmt.Values[0])
			if err != nil {
				return SignalNone{}, err
			}
			if v, ok := val.(ErrorValue); ok {
				return v, nil
			}

			if tup, ok := val.(TupleValue); ok {
				values = tup.Values
			} else {
				return SignalNone{}, NewRuntimeError(stmt, "multi assign expected multiple values")
			}
		} else {
			values = make([]Value, 0, len(stmt.Values))

			for idx, expr := range stmt.Values {
				v, err := i.EvalExpression(expr)
				if err != nil {
					return SignalNone{}, err
				}
				if v, ok := v.(ErrorValue); ok {
					return v, nil
				}

				values[idx] = v
			}
		}

		if len(stmt.Values) != len(stmt.Names) {
			return SignalNone{}, NewRuntimeError(stmt,
				fmt.Sprintf("expected %d values, got %d",
					len(stmt.Names), len(stmt.Values)))
		}

		var expectedTI *TypeInfo

		for idx, name := range stmt.Names {
			if _, ok := i.Env.Get(name.Value); ok {
				return SignalNone{}, NewRuntimeError(stmt,
					fmt.Sprintf("cannot redeclare var: %s", name.Value))
			}

			v, err := i.assignWithType(stmt, values[idx], expectedTI)
			if err != nil {
				return SignalNone{}, err
			}

			i.Env.Set(name.Value, v)
		}

	case *parser.IndexAssignmentStatement:
		leftVal, err := i.EvalExpression(stmt.Left)
		if err != nil {
			return NilValue{}, err
		}

		arrVal, ok := leftVal.(ArrayValue)
		if !ok {
			return SignalNone{}, NewRuntimeError(s, fmt.Sprintf("assignment to non-array: %v", leftVal.String()))
		}

		index, err := i.EvalExpression(stmt.Index)
		if err != nil {
			return SignalNone{}, err
		}

		idxVal, ok := index.(IntValue)
		if !ok {
			return SignalNone{}, NewRuntimeError(s, fmt.Sprintf("array index: %v, must be int", idxVal.V))
		}

		idx := idxVal.V

		if idx < 0 || idx >= len(arrVal.Elements) {
			return SignalNone{}, NewRuntimeError(s, fmt.Sprintf("array index: %d, out of bounds", idx))
		}

		newVal, err := i.EvalExpression(stmt.Value)
		if err != nil {
			return SignalNone{}, err
		}

		if newVal == nil {
			return SignalNone{}, nil
		}

		arrVal.Elements[idx] = newVal
		return SignalNone{}, nil

	case *parser.MemberAssignmentStatement:
		objVal, err := i.EvalExpression(stmt.Object)
		if err != nil {
			return SignalNone{}, nil
		}

		structVal, ok := objVal.(*StructValue)
		if !ok {
			return SignalNone{}, NewRuntimeError(s, "cannot assign field on non-struct")
		}

		val, err := i.EvalExpression(stmt.Value)
		if err != nil {
			return SignalNone{}, err
		}

		if _, ok := structVal.Fields[stmt.Field.Value]; !ok {
			return SignalNone{}, NewRuntimeError(s, fmt.Sprintf("unknown struct field: %s", stmt.Field.Value))
		}

		expectedType := structVal.TypeName.Fields[stmt.Field.Value]

		actualTI := unwrapAlias(i.typeInfoFromValue(val))
		expectedTI := unwrapAlias(expectedType)

		if !typesAssignable(actualTI, expectedTI) {
			return NilValue{}, NewRuntimeError(stmt, fmt.Sprintf("field '%s' expects %v but got %v", stmt.Field.Value, expectedType.Name, actualTI.Name))
		}

		structVal.Fields[stmt.Field.Value] = val
		return SignalNone{}, nil

	case *parser.EnumStatement:
		if _, ok := i.Env.Get(stmt.Name.Value); ok {
			return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("cannot redeclare enum: %s", stmt.Name.Value))
		}

		enumType := &TypeInfo{
			Name:     stmt.Name.Value,
			Kind:     TypeEnum,
			Variants: make(map[string]int),
		}

		for idx, ident := range stmt.Variants {
			name := ident.Value

			if _, exists := enumType.Variants[name]; exists {
				return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("duplicate enum variant: %s", name))
			}

			enumType.Variants[name] = idx
		}

		i.typeEnv[stmt.Name.Value] = enumType

		i.Env.Define(stmt.Name.Value, TypeValue{
			TypeName: enumType,
		})

		return SignalNone{}, nil

	case *parser.FuncStatement:
		paramTypes := make([]*TypeInfo, 0)
		paramNames := make([]string, 0)

		returnTypes := make([]*TypeInfo, 0)
		returnNames := make([]string, 0)

		err := i.checkFuncStatement(stmt)
		if err != nil {
			return SignalNone{}, err
		}

		for _, typ := range stmt.ReturnTypes {
			ti, err := i.resolveTypeNode(typ)
			if err != nil {
				return SignalNone{}, err
			}

			ti = unwrapAlias(ti)

			returnTypes = append(returnTypes, ti)
			paramNames = append(paramNames, ti.Name)
		}

		for _, param := range stmt.Params {
			ti, err := i.resolveTypeNode(param.Type)
			if err != nil {
				return SignalNone{}, err
			}

			ti = unwrapAlias(ti)

			paramTypes = append(paramTypes, ti)
			paramNames = append(paramNames, ti.Name)
		}

		typeInfo := &TypeInfo{
			Name:    fmt.Sprintf("fun(%s) (%s)", strings.Join(paramNames, ", "), strings.Join(returnNames, ", ")),
			Kind:    TypeFunc,
			Returns: returnTypes,
			Params:  paramTypes,
		}

		i.Env.Define(stmt.Name.Value, &Func{Params: stmt.Params, Body: stmt.Body, TypeName: typeInfo, Env: i.Env})
		return SignalNone{}, nil

	case *parser.MethodStatement:
		recvType, err := i.resolveTypeNode(stmt.Receiver.Type)
		if err != nil {
			return ErrorValue{
				V: NewRuntimeError(stmt, err.Error()),
			}, nil
		}

		params := append(
			[]*parser.Param{
				{
					Name: stmt.Receiver.Name,
					Type: stmt.Receiver.Type,
				},
			},
			stmt.Params...,
		)

		paramTypes := make([]*TypeInfo, 0)
		paramNames := make([]string, 0)

		returnTypes := make([]*TypeInfo, 0)
		returnNames := make([]string, 0)

		err = i.checkMethodStatement(stmt)
		if err != nil {
			return SignalNone{}, err
		}

		for _, typ := range stmt.ReturnTypes {
			ti, err := i.resolveTypeNode(typ)
			if err != nil {
				return SignalNone{}, err
			}

			ti = unwrapAlias(ti)

			returnTypes = append(returnTypes, ti)
			paramNames = append(paramNames, ti.Name)
		}

		for _, param := range stmt.Params {
			ti, err := i.resolveTypeNode(param.Type)
			if err != nil {
				return SignalNone{}, err
			}

			ti = unwrapAlias(ti)

			paramTypes = append(paramTypes, ti)
			paramNames = append(paramNames, ti.Name)
		}

		typeInfo := &TypeInfo{
			Name:    fmt.Sprintf("fun(%s) (%s) (%s)", recvType.Name, strings.Join(paramNames, ", "), strings.Join(returnNames, ", ")),
			Kind:    TypeFunc,
			Returns: returnTypes,
			Params:  paramTypes,
		}

		i.Env.SetMethod(recvType, stmt.Name.Value, &Func{
			Params:   params,
			Body:     stmt.Body,
			Env:      i.Env,
			TypeName: typeInfo,
		})

		return SignalNone{}, nil

	case *parser.ReturnStatement:
		values := []Value{}

		for _, expr := range stmt.Values {
			v, err := i.EvalExpression(expr)
			if err != nil {
				return SignalNone{}, err
			}
			values = append(values, v)
		}

		return SignalReturn{Values: values}, nil

	case *parser.ExpressionStatement:
		_, err := i.EvalExpression(stmt.Expression)
		if err != nil {
			return SignalNone{}, err
		}

		return SignalNone{}, nil

	case *parser.IfStatement:
		if stmt.Condition == nil {
			return SignalNone{}, NewRuntimeError(s, "if statement missing condition")
		}
		if stmt.Consequence == nil {
			return SignalNone{}, NewRuntimeError(s, "if statement missing consequence")
		}
		cond, err := i.EvalExpression(stmt.Condition)
		if err != nil {
			return SignalNone{}, err
		}

		truthy, err := isTruthy(cond)
		if err != nil {
			return SignalNone{}, NewRuntimeError(stmt, err.Error())
		}
		if truthy {
			if stmt.Consequence != nil {
				return i.EvalBlock(stmt.Consequence, true)
			}
		} else {
			if stmt.Alternative != nil {
				return i.EvalBlock(stmt.Alternative, true)
			}
		}
		return SignalNone{}, nil

	case *parser.SpawnStatement:
		go func() {
			defer func() {
				if r := recover(); r != nil {

				}
			}()

			i.EvalBlock(stmt.Body, true)
		}()

		return SignalNone{}, nil

	case *parser.SwitchStatement:
		switchVal, err := i.EvalExpression(stmt.Value)
		if err != nil {
			return SignalNone{}, err
		}

		for _, c := range stmt.Cases {
			caseVal, err := i.EvalExpression(c.Expr)
			if err != nil {
				return SignalNone{}, err
			}

			if !valuesEqual(switchVal, caseVal) {
				continue
			}

			sig, err := i.EvalBlock(c.Body, true)
			if err != nil {
				return SignalNone{}, err
			}

			if _, ok := sig.(SignalNone); !ok {
				return sig, nil
			}

			return SignalNone{}, nil
		}

		if stmt.Default != nil {
			sig, err := i.EvalBlock(stmt.Default.Body, true)
			if err != nil {
				return SignalNone{}, err
			}
			if _, ok := sig.(SignalNone); !ok {
				return sig, nil
			}
		}

		return SignalNone{}, nil

	case *parser.WithStatement:
		val, err := i.EvalExpression(stmt.Expr)
		if err != nil {
			return SignalNone{}, err
		}

		oldEnv := i.Env
		i.Env = NewEnvironment(oldEnv)

		i.Env.Define("it", ConstValue{Value: val})

		sig, err := i.EvalStatements(stmt.Body)

		i.Env = oldEnv

		return sig, err

	case *parser.ForStatement:
		loopEnv := NewEnvironment(i.Env)
		oldEnv := i.Env
		i.Env = loopEnv

		i.EvalStatement(stmt.Init)
		for {
			cond, err := i.EvalExpression(stmt.Condition)
			if err != nil {
				return SignalNone{}, err
			}

			truthy, err := isTruthy(cond)
			if err != nil {
				return SignalNone{}, NewRuntimeError(stmt, err.Error())
			}

			if !truthy {
				break
			}

			sig, err := i.EvalBlock(stmt.Body, false)
			if err != nil {
				return SignalNone{}, err
			}

			switch sig.(type) {
			case SignalBreak:
				return SignalNone{}, nil
			case SignalContinue:
				i.EvalStatement(stmt.Post)
				continue
			case SignalReturn:
				return sig, nil
			}
			i.EvalStatement(stmt.Post)
		}

		i.Env = oldEnv
		return SignalNone{}, nil

	case *parser.ForRangeStatement:
		iterable, err := i.EvalExpression(stmt.Expr)
		if err != nil {
			return SignalNone{}, err
		}

		iterable = unwrapNamed(iterable)

		switch v := iterable.(type) {
		case ArrayValue:
			for idx, elem := range v.Elements {
				oldEnv := i.Env
				i.Env = NewEnvironment(oldEnv)

				if stmt.Key != nil && stmt.Key.Value != "_" {
					i.Env.Define(stmt.Key.Value, IntValue{V: idx})
				}

				if stmt.Value != nil && stmt.Value.Value != "_" {
					i.Env.Define(stmt.Value.Value, elem)
				}

				sig, err := i.EvalBlock(stmt.Body, false)

				i.Env = oldEnv

				if err != nil {
					return SignalNone{}, err
				}

				switch sig.(type) {
				case SignalBreak:
					return SignalNone{}, nil
				case SignalContinue:
					continue
				case SignalReturn:
					return sig, nil
				}
			}
		case MapValue:
			for k, val := range v.Entries {
				oldEnv := i.Env
				i.Env = NewEnvironment(oldEnv)

				if stmt.Key != nil && stmt.Key.Value != "_" {
					i.Env.Define(stmt.Key.Value, k)
				}

				if stmt.Value != nil && stmt.Value.Value != "_" {
					i.Env.Define(stmt.Value.Value, val)
				}

				sig, err := i.EvalBlock(stmt.Body, false)

				i.Env = oldEnv

				if err != nil {
					return SignalNone{}, err
				}

				switch sig.(type) {
				case SignalBreak:
					return SignalNone{}, nil
				case SignalContinue:
					continue
				case SignalReturn:
					return sig, nil
				}
			}
		case StringValue:
			for idx, s := range v.V {
				oldEnv := i.Env
				i.Env = NewEnvironment(oldEnv)

				if stmt.Key != nil && stmt.Key.Value != "_" {
					i.Env.Define(stmt.Key.Value, IntValue{V: idx})
				}

				if stmt.Value != nil && stmt.Value.Value != "_" {
					i.Env.Define(stmt.Value.Value, StringValue{V: string(s)})
				}

				sig, err := i.EvalBlock(stmt.Body, false)

				i.Env = oldEnv

				if err != nil {
					return SignalNone{}, err
				}

				switch sig.(type) {
				case SignalBreak:
					return SignalNone{}, nil
				case SignalContinue:
					continue
				case SignalReturn:
					return sig, nil
				}
			}
		case IntValue:
			for idx := range v.V {
				oldEnv := i.Env
				i.Env = NewEnvironment(oldEnv)

				if stmt.Key != nil && stmt.Key.Value != "_" {
					i.Env.Define(stmt.Key.Value, IntValue{V: idx})
				}

				if stmt.Value != nil && stmt.Value.Value != "_" {
					return SignalNone{}, NewRuntimeError(stmt, "integer range expects 1 variable")
				}

				sig, err := i.EvalBlock(stmt.Body, false)

				i.Env = oldEnv

				if err != nil {
					return SignalNone{}, err
				}

				switch sig.(type) {
				case SignalBreak:
					return SignalNone{}, nil
				case SignalContinue:
					continue
				case SignalReturn:
					return sig, nil
				}
			}
		default:
			return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("range expects (array|map|int|string), but got %s", unwrapAlias(i.typeInfoFromValue(iterable)).Name))
		}

		return SignalNone{}, nil

	case *parser.WhileStatement:
		for {
			cond, err := i.EvalExpression(stmt.Condition)
			if err != nil {
				return SignalNone{}, err
			}

			truthy, err := isTruthy(cond)
			if err != nil {
				return SignalNone{}, NewRuntimeError(stmt, err.Error())
			}

			if !truthy {
				break
			}

			sig, err := i.EvalBlock(stmt.Body, true)
			if err != nil {
				return SignalNone{}, err
			}

			switch sig.(type) {
			case SignalBreak:
				return SignalNone{}, nil
			case SignalContinue:
				continue
			case SignalReturn:
				return sig, nil
			}
		}
		return SignalNone{}, nil

	case *parser.DeferStatement:
		i.Env.AddDefer(stmt.Call)
		return SignalNone{}, nil

	case *parser.BreakStatement:
		return SignalBreak{}, nil

	case *parser.ContinueStatement:
		return SignalContinue{}, nil
	}

	return SignalNone{}, nil
}

func (i *Interpreter) EvalExpression(e parser.Expression) (Value, error) {
	if e == nil {
		return NilValue{}, nil
	}

	switch expr := e.(type) {
	case *parser.IntLiteral:
		return IntValue{V: expr.Value}, nil

	case *parser.FloatLiteral:
		return FloatValue{V: expr.Value}, nil

	case *parser.StringLiteral:
		return StringValue{V: expr.Value}, nil

	case *parser.BoolLiteral:
		return BoolValue{V: expr.Value}, nil

	case *parser.NilLiteral:
		return NilValue{}, nil

	case parser.TypeNode:
		ti, err := i.resolveTypeNode(expr)
		if err != nil {
			return NilValue{}, err
		}

		return TypeValue{
			TypeName: ti,
		}, nil

	case *parser.MemberExpression:
		leftVal, err := i.EvalExpression(expr.Left)
		if err != nil {
			return NilValue{}, err
		}
		if v, ok := leftVal.(ErrorValue); ok {
			return v, nil
		}

		return i.evalMemberExpression(expr, leftVal, expr.Field.Value)

	case *parser.Identifier:
		if expr.Value == "_" {
			return ErrorValue{
				V: NewRuntimeError(expr, "cannot use '_' as a value"),
			}, nil
		}

		v, ok := i.Env.Get(expr.Value)
		if !ok {
			return ErrorValue{
				V: NewRuntimeError(expr, fmt.Sprintf("undefined variable: %s", expr.Value)),
			}, nil
		}

		if c, isConst := v.(ConstValue); isConst {
			return c.Value, nil
		}

		return v, nil

	case *parser.CompositeLiteral:
		ti, err := i.resolveTypeNode(expr.Type)
		if err != nil {
			return ErrorValue{
				V: err,
			}, nil
		}
		return i.evalCompositeLiteral(expr, ti)
	case *parser.AnonymousStructLiteral:
		fields := make(map[string]Value)
		fieldTypes := make(map[string]*TypeInfo)

		for name, e := range expr.Fields {
			v, err := i.EvalExpression(e)
			if err != nil {
				return NilValue{}, err
			}

			if expected, ok := fieldTypes[name]; ok {
				actualTI := unwrapAlias(i.typeInfoFromValue(v))
				expectedTI := unwrapAlias(expected)

				if !(typesAssignable(actualTI, expectedTI)) {
					return ErrorValue{
						V: NewRuntimeError(
							expr,
							fmt.Sprintf(
								"field '%s' expects '%s' but got '%s'",
								name,
								expected.Name,
								actualTI.Name,
							),
						),
					}, nil
				}
			}

			fields[name] = v
			fieldTypes[name] = i.typeInfoFromValue(v)
		}

		return &StructValue{
			TypeName: &TypeInfo{
				Name:   "<anon>",
				Fields: fieldTypes,
			},
			Fields: fields,
		}, nil

	case *parser.FuncLiteral:
		paramTypes := make([]*TypeInfo, 0)
		paramNames := make([]string, 0)

		returnTypes := make([]*TypeInfo, 0)
		returnNames := make([]string, 0)

		err := i.checkFuncLiteral(expr)
		if err != nil {
			return ErrorValue{
				V: err,
			}, nil
		}

		for _, typ := range expr.ReturnTypes {
			ti, err := i.resolveTypeNode(typ)
			if err != nil {
				return NilValue{}, err
			}

			ti = unwrapAlias(ti)

			returnTypes = append(returnTypes, ti)
			paramNames = append(paramNames, ti.Name)
		}

		for _, param := range expr.Params {
			ti, err := i.resolveTypeNode(param.Type)
			if err != nil {
				return NilValue{}, err
			}

			ti = unwrapAlias(ti)

			paramTypes = append(paramTypes, ti)
			returnNames = append(returnNames, ti.Name)
		}

		typeInfo := &TypeInfo{
			Name:    fmt.Sprintf("fun(%s) (%s)", strings.Join(paramNames, ", "), strings.Join(returnNames, ", ")),
			Kind:    TypeFunc,
			Returns: returnTypes,
			Params:  paramTypes,
		}

		return &Func{
			Params:   expr.Params,
			Body:     expr.Body,
			TypeName: typeInfo,
			Env:      i.Env,
		}, nil

	case *parser.FuncCall:
		return i.evalCall(expr)

	case *parser.IndexExpression:
		left, err := i.EvalExpression(expr.Left)
		if err != nil {
			return NilValue{}, err
		}
		if v, ok := left.(ErrorValue); ok {
			return v, nil
		}

		index, err := i.EvalExpression(expr.Index)
		if err != nil {
			return NilValue{}, err
		}
		if v, ok := index.(ErrorValue); ok {
			return v, nil
		}

		val, err := i.evalIndexExpression(expr, left, index)
		if err != nil {
			return NilValue{}, err
		}
		if v, ok := val.(ErrorValue); ok {
			return v, nil
		}

		return val, nil

	case *parser.SliceExpression:
		left, err := i.EvalExpression(expr.Left)
		if err != nil {
			return NilValue{}, err
		}
		if v, ok := left.(ErrorValue); ok {
			return v, nil
		}

		start, err := i.EvalExpression(expr.Start)
		if err != nil {
			return NilValue{}, err
		}
		if v, ok := start.(ErrorValue); ok {
			return v, nil
		}

		end, err := i.EvalExpression(expr.End)
		if err != nil {
			return NilValue{}, err
		}
		if v, ok := end.(ErrorValue); ok {
			return v, nil
		}

		val, err := i.evalSliceExpression(expr, left, start, end)
		if err != nil {
			return NilValue{}, err
		}
		if v, ok := val.(ErrorValue); ok {
			return v, nil
		}

		return val, nil

	case *parser.TypeAssertExpression:
		val, err := i.EvalExpression(expr.Expr)
		if err != nil {
			return NilValue{}, err
		}

		staticTI := unwrapAlias(i.typeInfoFromValue(val))
		if staticTI.Kind != TypeAny {
			return NilValue{}, NewRuntimeError(expr, "type assertion only allowed on 'thing'")
		}

		targetTI, err := i.resolveTypeNode(expr.Type)
		if err != nil {
			return NilValue{}, err
		}

		inner := unwrapNamed(val)
		actualTI := unwrapAlias(i.typeInfoFromValue(inner))

		if !typesAssignable(actualTI, targetTI) {
			return ErrorValue{
				V: NewRuntimeError(expr, fmt.Sprintf("type mismatch: '%s' asserted as '%s'", actualTI.Name, targetTI.Name)),
			}, nil
		}

		return i.promoteValueToType(inner, targetTI), nil

	case *parser.InfixExpression:
		if expr.Operator == "&&" {
			left, err := i.EvalExpression(expr.Left)
			if err != nil {
				return NilValue{}, err
			}

			lTruthy, err := isTruthy(left)
			if err != nil {
				return NilValue{}, NewRuntimeError(expr, err.Error())
			}

			if !lTruthy {
				return BoolValue{V: false}, nil
			}

			right, err := i.EvalExpression(expr.Right)
			if err != nil {
				return NilValue{}, err
			}

			rTruthy, err := isTruthy(right)
			if err != nil {
				return NilValue{}, NewRuntimeError(expr, err.Error())
			}

			return BoolValue{V: rTruthy}, nil
		}

		if expr.Operator == "||" {
			left, err := i.EvalExpression(expr.Left)
			if err != nil {
				return NilValue{}, err
			}

			lTruthy, err := isTruthy(left)
			if err != nil {
				return NilValue{}, NewRuntimeError(expr, err.Error())
			}

			if lTruthy {
				return BoolValue{V: true}, nil
			}

			right, err := i.EvalExpression(expr.Right)
			if err != nil {
				return NilValue{}, err
			}

			rTruthy, err := isTruthy(right)
			if err != nil {
				return NilValue{}, NewRuntimeError(expr, err.Error())
			}

			return BoolValue{V: rTruthy}, nil
		}

		left, err := i.EvalExpression(expr.Left)
		if err != nil {
			return NilValue{}, err
		}

		right, err := i.EvalExpression(expr.Right)
		if err != nil {
			return NilValue{}, err
		}

		return i.evalInfix(expr, left, expr.Operator, right)

	case *parser.PrefixExpression:
		right, err := i.EvalExpression(expr.Right)
		if err != nil {
			return NilValue{}, err
		}

		return evalPrefix(expr, expr.Operator, right)

	case *parser.GroupedExpression:
		return i.EvalExpression(expr.Expression)

	case *parser.InExpression:
		elem, err := i.EvalExpression(expr.Left)
		if err != nil {
			return NilValue{}, err
		}
		if v, ok := elem.(ErrorValue); ok {
			return v, nil
		}

		set, err := i.EvalExpression(expr.Right)
		if err != nil {
			return NilValue{}, err
		}
		if v, ok := set.(ErrorValue); ok {
			return v, nil
		}

		switch s := set.(type) {
		case MapValue:
			_, ok := s.Entries[elem]
			return BoolValue{V: ok}, nil

		case ArrayValue:
			for _, v := range s.Elements {
				if valuesEqual(v, elem) {
					return BoolValue{V: true}, nil
				}
			}
			return BoolValue{V: false}, nil
		case StringValue:
			if strings.Contains(s.V, elem.(StringValue).V) {
				return BoolValue{V: true}, nil
			}
			return BoolValue{V: false}, nil
		}

		return ErrorValue{
			V: NewRuntimeError(expr, "in expects map or array or string"),
		}, nil

	case *parser.InterpolatedString:
		is := expr
		var out strings.Builder

		for _, part := range is.Parts {
			val, err := i.EvalExpression(part)
			if err != nil {
				return NilValue{}, err
			}
			out.WriteString(val.String())
		}

		return StringValue{V: out.String()}, nil

	default:
		return ErrorValue{
			V: NewRuntimeError(expr, fmt.Sprintf("unhandled expression type: %T", e)),
		}, nil
	}
}

func (i *Interpreter) assignWithType(node parser.Node, v Value, expected *TypeInfo) (Value, error) {
	if expected == nil {
		return v, nil
	}

	expected = unwrapAlias(expected)
	actual := unwrapAlias(i.typeInfoFromValue(v))

	// special case: []thing absorbs array elem types
	if expected.Kind == TypeArray && expected.Elem.Kind == TypeAny {
		if arr, ok := v.(ArrayValue); ok {
			arr.ElemType = expected.Elem
			v = arr
			actual = expected
		}
	}

	if !typesAssignable(actual, expected) {
		return NilValue{}, NewRuntimeError(
			node,
			fmt.Sprintf(
				"type mismatch: expected '%s' but got '%s'",
				expected.Name,
				actual.Name,
			),
		)
	}

	v = i.promoteValueToType(v, expected)

	if expected.Kind == TypeAny {
		v = NamedValue{
			TypeName: expected,
			Value:    v,
		}
	}

	return v, nil
}

func (i *Interpreter) evalCompositeLiteral(expr *parser.CompositeLiteral, ti *TypeInfo) (Value, error) {
	switch ti.Kind {
	case TypeArray, TypeFixedArray:
		return i.evalArrayLiteral(expr, ti)
	case TypeMap:
		return i.evalMapLiteral(expr, ti)
	case TypeStruct:
		return i.evalStructLiteral(expr, ti)
	case TypeNamed:
		if ti.Underlying.Kind == TypeStruct {
			return i.evalStructLiteral(expr, ti)
		}
		return i.evalCompositeLiteral(expr, ti.Underlying)
	default:
		return nil, NewRuntimeError(expr, fmt.Sprintf("composite literals do not support '%s'", ti.Name))
	}
}

func (i *Interpreter) evalStructLiteral(expr *parser.CompositeLiteral, typeInfo *TypeInfo) (Value, error) {
	var structType *TypeInfo

	switch typeInfo.Kind {
	case TypeStruct:
		structType = typeInfo

	case TypeNamed:
		if typeInfo.Underlying.Kind != TypeStruct {
			return ErrorValue{
				V: NewRuntimeError(expr,
					fmt.Sprintf("%s is not a struct type", typeInfo.Name)),
			}, nil
		}
		structType = typeInfo.Underlying

	default:
		return ErrorValue{
			V: NewRuntimeError(expr,
				fmt.Sprintf("%s is not a struct type", typeInfo.Name)),
		}, nil
	}

	fields := make(map[string]Value)

	for name, e := range expr.Fields {
		expectedType, ok := structType.Fields[name]
		if !ok {
			return ErrorValue{
				V: NewRuntimeError(
					expr,
					fmt.Sprintf("unknown field '%s' in struct %s",
						name, typeInfo.Name),
				),
			}, nil
		}

		v, err := i.EvalExpression(e)
		if err != nil {
			return NilValue{}, err
		}

		actualTI := unwrapAlias(i.typeInfoFromValue(v))
		expectedTI := unwrapAlias(expectedType)

		if !typesAssignable(actualTI, expectedTI) {
			return ErrorValue{
				V: NewRuntimeError(
					expr,
					fmt.Sprintf(
						"field '%s' expects '%s' but got '%s'",
						name,
						expectedType.Name,
						actualTI.Name,
					),
				),
			}, nil
		}

		fields[name] = v
	}

	return &StructValue{
		TypeName: typeInfo,
		Fields:   fields,
	}, nil
}

func (i *Interpreter) evalArrayLiteral(expr *parser.CompositeLiteral, ti *TypeInfo) (Value, error) {
	if ti.Kind != TypeArray && ti.Kind != TypeFixedArray {
		return nil, NewRuntimeError(expr, "composite literal is not an array type")
	}

	elemType := ti.Elem

	elements := make([]Value, 0, len(expr.Elements))

	for idx, el := range expr.Elements {
		val, err := i.EvalExpression(el)
		if err != nil {
			return NilValue{}, err
		}

		valType := unwrapAlias(i.typeInfoFromValue(val))

		if !typesAssignable(valType, elemType) {
			return nil, NewRuntimeError(
				expr,
				fmt.Sprintf(
					"array element %d expected %s but got %s",
					idx,
					elemType.Name,
					valType.Name,
				),
			)
		}

		val = i.promoteValueToType(val, elemType)
		elements = append(elements, val)
	}

	if ti.Kind == TypeFixedArray {
		if len(elements) > ti.Size {
			return NilValue{}, NewRuntimeError(
				expr,
				fmt.Sprintf(
					"too many elements: capacity is %d but got %d",
					ti.Size,
					len(elements),
				),
			)
		}

		for len(elements) < ti.Size {
			def, err := i.defaultValueFromTypeInfo(expr, elemType)
			if err != nil {
				return NilValue{}, err
			}
			elements = append(elements, def)
		}
	}

	return ArrayValue{
		Elements: elements,
		ElemType: elemType,
		Capacity: capacityFromType(ti, elements),
		Fixed:    ti.Kind == TypeFixedArray,
	}, nil
}

func (i *Interpreter) evalMapLiteral(expr *parser.CompositeLiteral, expected *TypeInfo) (Value, error) {
	if len(expr.Pairs) == 0 {
		if expected == nil || expected.Kind != TypeMap {
			return ErrorValue{
				V: NewRuntimeError(expr, "cannot infer type of empty map"),
			}, nil
		}

		return MapValue{
			Entries:   map[Value]Value{},
			KeyType:   expected.Key,
			ValueType: expected.Value,
		}, nil
	}

	k0, err := i.EvalExpression(expr.Pairs[0].Key)
	if err != nil {
		return NilValue{}, err
	}
	if v, ok := k0.(ErrorValue); ok {
		return v, nil
	}

	v0, err := i.EvalExpression(expr.Pairs[0].Value)
	if err != nil {
		return NilValue{}, err
	}
	if v, ok := v0.(ErrorValue); ok {
		return v, nil
	}

	keyTI := unwrapAlias(i.typeInfoFromValue(k0))
	valTI := unwrapAlias(i.typeInfoFromValue(v0))

	if expected != nil {
		if !isComparableValue(k0) {
			return ErrorValue{
				V: NewRuntimeError(expr, fmt.Sprintf("map key type %s is not comparable", keyTI.Name)),
			}, nil
		}

		if !typesAssignable(keyTI, expected.Key) {
			return ErrorValue{
				V: NewRuntimeError(expr, fmt.Sprintf("type mismatch: map key 0 expected %s but got %s", expected.Key.Name, keyTI.Name)),
			}, nil
		}
		keyTI = expected.Key

		if !typesAssignable(valTI, expected.Value) {
			return ErrorValue{
				V: NewRuntimeError(expr, fmt.Sprintf("type mismatch: map value 0 expected %s but got %s", expected.Value.Name, valTI.Name)),
			}, nil
		}
		valTI = expected.Value
	}

	elems := map[Value]Value{}

	for idx, e := range expr.Pairs {
		k, err := i.EvalExpression(e.Key)
		if err != nil {
			return NilValue{}, err
		}
		if v, ok := k.(ErrorValue); ok {
			return v, nil
		}

		v, err := i.EvalExpression(e.Value)
		if err != nil {
			return NilValue{}, err
		}

		kt := unwrapAlias(i.typeInfoFromValue(k))
		vt := unwrapAlias(i.typeInfoFromValue(v))

		if keyTI.Kind == TypeAny && !isComparableValue(k) {
			return ErrorValue{
				V: NewRuntimeError(expr, fmt.Sprintf("map key %d is not comparable", idx)),
			}, nil
		}

		if !typesAssignable(kt, keyTI) {
			return ErrorValue{
				V: NewRuntimeError(expr, fmt.Sprintf("map key %d expected %s but got %s", idx, keyTI.Name, kt.Name)),
			}, nil
		}

		if !typesAssignable(vt, valTI) {
			return ErrorValue{
				V: NewRuntimeError(expr, fmt.Sprintf("map value %d expected %s but got %s", idx, valTI.Name, vt.Name)),
			}, nil
		}

		elems[k] = v
	}

	return MapValue{
		Entries:   elems,
		KeyType:   keyTI,
		ValueType: valTI,
	}, nil
}

func (i *Interpreter) evalCall(e *parser.FuncCall) (Value, error) {
	if ident, ok := e.Callee.(*parser.Identifier); ok {
		if ti, ok := i.typeEnv[ident.Value]; ok {
			if len(e.Args) != 1 {
				return ErrorValue{
					V: NewRuntimeError(e, "type cast expects 1 arg"),
				}, nil
			}
			return i.evalTypeCast(ti, e.Args[0], e)
		}
	}

	return i.evalFuncCall(e)
}

func (i *Interpreter) evalTypeCast(target *TypeInfo, arg parser.Expression, node parser.Node) (Value, error) {
	v, err := i.EvalExpression(arg)
	if err != nil {
		return NilValue{}, err
	}

	v = unwrapNamed(v)

	switch target.Kind {
	case TypeInt:
		var val int

		if v, ok := v.(ErrorValue); ok {
			return v, nil
		}

		switch v := v.(type) {
		case IntValue:
			val = v.V
		case FloatValue:
			val = int(v.V)
		default:
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("int type cast does not support %s, try the function toInt to parse non-numeric types", string(v.Type()))),
			}, nil
		}

		return IntValue{V: val}, nil
	case TypeFloat:
		var val float64

		if v, ok := v.(ErrorValue); ok {
			return v, nil
		}

		switch v := v.(type) {
		case IntValue:
			val = float64(v.V)
		case FloatValue:
			val = v.V
		default:
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("float type cast does not support %s, try the function toFloat to parse non-numeric types", string(v.Type()))),
			}, nil
		}

		return FloatValue{V: val}, nil
	case TypeString:
		if s, ok := v.(StringValue); ok {
			return s, nil
		}
		if v, ok := v.(ErrorValue); ok {
			return v, nil
		}

		return ErrorValue{
			V: NewRuntimeError(node, fmt.Sprintf("string cast does not support %s, try the function toString to parse other types", string(v.Type()))),
		}, nil
	case TypeBool:
		var val bool

		if v, ok := v.(ErrorValue); ok {
			return v, nil
		}

		switch v := v.(type) {
		case BoolValue:
			val = v.V
		default:
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("bool type cast does not support %s, try the function toBool to parse other types", string(v.Type()))),
			}, nil
		}

		return BoolValue{V: val}, nil

	case TypeArray:
		if a, ok := v.(ArrayValue); ok {
			return a, nil
		}
		if v, ok := v.(ErrorValue); ok {
			return v, nil
		}

		return ErrorValue{
			NewRuntimeError(node, fmt.Sprintf("array cast does not support %s, try the function toArr to construct arrays", string(v.Type()))),
		}, nil
	case TypeNamed:
		base := target.Underlying

		if v, ok := v.(ErrorValue); ok {
			return v, nil
		}

		casted, err := i.evalTypeCast(base, arg, node)
		if err != nil {
			return NilValue{}, err
		}
		if v, ok := casted.(ErrorValue); ok {
			return v, nil
		}

		if sv, ok := casted.(*StructValue); ok {
			sv.TypeName = target
			return sv, nil
		}

		return NamedValue{
			TypeName: target,
			Value:    casted,
		}, nil
	case TypeError:
		s, ok := v.(StringValue)
		if ok {
			return ErrorValue{V: errors.New(s.V)}, nil
		}

		return ErrorValue{
			V: NewRuntimeError(node, fmt.Sprintf("error cast does not support %s", string(v.Type()))),
		}, nil
	default:
		return ErrorValue{
			V: NewRuntimeError(node, fmt.Sprintf("unknown type cast: %s", target.Name)),
		}, nil
	}
}

func (i *Interpreter) evalArgs(args []parser.Expression) ([]Value, error) {
	var values []Value

	for _, arg := range args {
		if spread, ok := arg.(*parser.SpreadExpression); ok {

			v, err := i.EvalExpression(spread.Expression)
			if err != nil {
				return nil, err
			}

			arr, ok := v.(ArrayValue)
			if !ok {
				return nil, NewRuntimeError(spread,
					"spread argument must be an array or slice")
			}

			values = append(values, arr.Elements...)
			continue
		}

		v, err := i.EvalExpression(arg)
		if err != nil {
			return nil, err
		}

		values = append(values, v)
	}

	return values, nil
}

func (i *Interpreter) evalFuncCall(expr *parser.FuncCall) (Value, error) {
	// builtin
	if ident, ok := expr.Callee.(*parser.Identifier); ok {
		if b, ok := i.Env.builtins[ident.Value]; ok {
			args, err := i.evalArgs(expr.Args)
			if err != nil {
				return NilValue{}, err
			}
			if b.Arity >= 0 && len(args) != b.Arity {
				return NilValue{}, NewRuntimeError(expr,
					fmt.Sprintf("expected %d args, got %d", b.Arity, len(args)))
			}
			return b.Fn(i, expr, args)
		}
	}

	// user-defined
	val, err := i.EvalExpression(expr.Callee)
	if err != nil {
		return NilValue{}, err
	}
	if v, ok := val.(ErrorValue); ok {
		return v, nil
	}

	args, err := i.evalArgs(expr.Args)
	if err != nil {
		return NilValue{}, err
	}

	switch fn := val.(type) {
	case *BuiltinFunc:
		if fn.Arity >= 0 && len(args) != fn.Arity {
			return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("expected %d args, got %d", fn.Arity, len(args)))
		}
		return fn.Fn(i, expr, args)
	case *Func:
		return i.callFunction(fn, args, expr)
	case BoundMethodValue:
		args = append([]Value{fn.Receiver}, args...)
		return i.callFunction(fn.Func, args, expr)
	default:
		return ErrorValue{
			V: NewRuntimeError(expr, fmt.Sprintf("expected 'function' but got '%s'", unwrapAlias(i.typeInfoFromValue(val)).Name)),
		}, nil
	}
}

func (i *Interpreter) callFunction(fn *Func, args []Value, callNode parser.Node) (Value, error) {
	paramCount := len(fn.Params)
	argCount := len(args)

	isVariadic := false
	if paramCount > 0 && fn.Params[paramCount-1].Variadic {
		isVariadic = true
	}

	if !isVariadic {
		if argCount != paramCount {
			return ErrorValue{
				V: NewRuntimeError(callNode, fmt.Sprintf("expected %d args, got %d", paramCount, argCount)),
			}, nil
		}
	} else {
		fixedCount := paramCount - 1
		if argCount < fixedCount {
			return ErrorValue{
				V: NewRuntimeError(callNode, fmt.Sprintf("expected atleast %d args, got %d", fixedCount, argCount)),
			}, nil
		}
	}

	newEnv := NewEnvironment(fn.Env)

	fixedCount := paramCount
	if isVariadic {
		fixedCount = paramCount - 1
	}

	for idx := 0; idx < fixedCount; idx++ {
		param := fn.Params[idx]
		val := args[idx]

		if param.Type != nil {
			expected, err := i.resolveTypeNode(param.Type)
			if err != nil {
				return NilValue{}, err
			}

			actual := unwrapAlias(i.typeInfoFromValue(val))

			val, err = i.assignWithType(callNode, val, expected)
			if err != nil {
				return ErrorValue{
					V: NewRuntimeError(callNode, fmt.Sprintf("param '%s' expected '%s' but got '%s'", param.Name.Value, expected.Name, actual.Name)),
				}, nil
			}
		}

		newEnv.Define(param.Name.Value, val)
	}

	if isVariadic {
		variadicParam := fn.Params[paramCount-1]

		expectedSliceType, err := i.resolveTypeNode(variadicParam.Type)
		if err != nil {
			return NilValue{}, err
		}

		// expectedSliceType should already be []T
		elemType := expectedSliceType.Elem

		var elements []Value

		for idx := fixedCount; idx < argCount; idx++ {
			v := args[idx]

			actual := unwrapAlias(i.typeInfoFromValue(v))
			v, err = i.assignWithType(callNode, v, elemType)
			if err != nil {
				return NilValue{}, NewRuntimeError(callNode,
					fmt.Sprintf("variadic param '%s' expected '%s' but got '%s'",
						variadicParam.Name.Value,
						elemType.Name,
						actual.Name))
			}

			elements = append(elements, v)
		}

		sliceValue := ArrayValue{
			Elements: elements,
			ElemType: elemType,
			Capacity: len(elements),
			Fixed:    false,
		}

		newEnv.Define(variadicParam.Name.Value, sliceValue)
	}

	// execute
	prevEnv := i.Env
	i.Env = newEnv

	sig, err := i.EvalBlock(fn.Body, false)

	deferErr := i.runDefers(newEnv)

	i.Env = prevEnv

	if err != nil {
		return NilValue{}, err
	}
	if deferErr != nil {
		return NilValue{}, deferErr
	}

	// handle return
	if ret, ok := sig.(SignalReturn); ok {
		if len(fn.TypeName.Returns) > 0 && len(fn.TypeName.Returns) != len(ret.Values) {
			return ErrorValue{
				V: NewRuntimeError(callNode,
					fmt.Sprintf("expected %d return values, got %d",
						len(fn.TypeName.Returns), len(ret.Values))),
			}, nil
		}

		for idx, expectedType := range fn.TypeName.Returns {
			actual := ret.Values[idx]

			if err != nil {
				return ErrorValue{
					V: NewRuntimeError(callNode, err.Error()),
				}, nil
			}
			expectedTI := unwrapAlias(expectedType)

			if expectedTI.Name == "error" {
				if _, isNil := actual.(NilValue); isNil {
					continue
				}
			}

			if unwrapAlias(i.typeInfoFromValue(actual)).Kind == TypeError {
				return actual, nil
			}

			actual, err = i.assignWithType(callNode, actual, expectedTI)
			if err != nil {
				return NilValue{}, err
			}

			ret.Values[idx] = actual
		}

		if len(fn.TypeName.Returns) > 1 {
			return TupleValue{Values: ret.Values}, nil
		}
		return ret.Values[0], nil
	}

	return NilValue{}, nil
}

func (i *Interpreter) evalIndexExpression(node parser.Expression, left, idx Value) (Value, error) {
	if nv, ok := left.(NamedValue); ok && nv.TypeName.Kind == TypeAny {
		return ErrorValue{
			V: NewRuntimeError(node, "cannot index value of type 'thing' without type assertion"),
		}, nil
	}

	left = unwrapNamed(left)

	typ := i.typeInfoFromValue(left)

	switch typ.Kind {
	case TypeArray, TypeFixedArray:
		arr, ok := left.(ArrayValue)

		idxVal, ok := idx.(IntValue)
		if !ok {
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("array index: %v, must be int", idxVal.V)),
			}, nil
		}

		idx := idxVal.V

		if idx < 0 || idx >= len(arr.Elements) {
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("array index: %d, out of bounds", idx)),
			}, nil
		}

		elem := arr.Elements[idx]

		if arr.ElemType.Kind == TypeAny {
			elem = NamedValue{
				TypeName: arr.ElemType,
				Value:    elem,
			}
		}

		return elem, nil

	case TypeString:
		idxVal, ok := idx.(IntValue)
		if !ok {
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("string index: %v, must be int", idxVal.V)),
			}, nil
		}

		idx := idxVal.V

		if idx < 0 || idx >= len(left.(StringValue).V) {
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("string index: %d, out of bounds", idx)),
			}, nil
		}

		r := []rune(left.(StringValue).V)
		return StringValue{V: string(r[idx])}, nil

	case TypeMap:
		mv := left.(MapValue)

		// 1. type check key
		keyType := unwrapAlias(i.typeInfoFromValue(idx))

		if mv.KeyType.Kind == TypeAny {
			if !isComparableValue(idx) {
				return ErrorValue{
					V: NewRuntimeError(
						node,
						"value of this type cannot be used as map key",
					),
				}, nil
			}
		} else {
			if !typesAssignable(keyType, mv.KeyType) {
				return ErrorValue{
					V: NewRuntimeError(
						node,
						fmt.Sprintf(
							"map index expected %s but got %s",
							mv.KeyType.Name,
							keyType.Name,
						),
					),
				}, nil
			}
		}

		val, ok := mv.Entries[idx]
		if !ok {
			return NilValue{}, nil
		}

		if mv.ValueType.Kind == TypeAny {
			return NamedValue{
				TypeName: mv.ValueType,
				Value:    val,
			}, nil
		}

		return val, nil

	default:
		var typeStr string
		var typeInt int

		switch typ.Kind {
		case 0:
			typeStr = "int"
		case 1:
			typeStr = "float"
		case 3:
			typeStr = "bool"
		case 5:
			typeStr = "nil"
		case 6:
			typeStr = "struct"
		case 8:
			typeStr = "thing"
		default:
			typeStr = ""
			typeInt = int(typ.Kind)
		}

		if typeStr != "" {
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("indexing is not allowed with type: '%s'", typeStr)),
			}, nil
		}

		return ErrorValue{
			V: NewRuntimeError(node, fmt.Sprintf("indexing is not allowed with type: %d", typeInt)),
		}, nil
	}
}

func (i *Interpreter) evalSliceExpression(node parser.Expression, left, startVal, endVal Value) (Value, error) {
	if nv, ok := left.(NamedValue); ok && nv.TypeName.Kind == TypeAny {
		return ErrorValue{
			V: NewRuntimeError(node,
				"cannot slice value of type 'thing' without type assertion"),
		}, nil
	}

	left = unwrapNamed(left)
	typ := i.typeInfoFromValue(left)

	var length int
	switch typ.Kind {
	case TypeArray, TypeFixedArray:
		length = len(left.(ArrayValue).Elements)
	case TypeString:
		length = len([]rune(left.(StringValue).V))
	default:
		return ErrorValue{
			V: NewRuntimeError(node,
				fmt.Sprintf("slicing is not allowed with type: '%s'", typ.Name)),
		}, nil
	}

	start := 0
	end := length

	if _, ok := startVal.(NilValue); !ok {
		intVal, ok := startVal.(IntValue)
		if !ok {
			return ErrorValue{
				V: NewRuntimeError(node, "slice start must be int"),
			}, nil
		}
		start = intVal.V
	}

	if _, ok := endVal.(NilValue); !ok {
		intVal, ok := endVal.(IntValue)
		if !ok {
			return ErrorValue{
				V: NewRuntimeError(node, "slice end must be int"),
			}, nil
		}
		end = intVal.V
	}

	if start < 0 || end < 0 || start > end || end > length {
		return ErrorValue{
			V: NewRuntimeError(node,
				fmt.Sprintf("slice bounds out of range [%d:%d]", start, end)),
		}, nil
	}

	switch typ.Kind {

	case TypeArray, TypeFixedArray:
		arr := left.(ArrayValue)
		newElems := arr.Elements[start:end]

		return ArrayValue{
			Elements: newElems,
			ElemType: arr.ElemType,
		}, nil

	case TypeString:
		runes := []rune(left.(StringValue).V)
		return StringValue{
			V: string(runes[start:end]),
		}, nil
	}

	return NilValue{}, nil
}

func (i *Interpreter) evalMemberExpression(node parser.Expression, left Value, field string) (Value, error) {
	recvType := unwrapAlias(i.typeInfoFromValue(left))
	if fn, ok := i.Env.GetMethod(recvType, field); ok {
		return BoundMethodValue{
			Receiver: left,
			Func:     fn,
		}, nil
	}

	switch obj := left.(type) {
	case ModuleValue:
		val, ok := obj.Env.Get(field)
		if !ok {
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("unknown '%s'", field)),
			}, nil
		}

		return val, nil
	case *StructValue:

		val, ok := obj.Fields[field]
		if !ok {
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("unknown field %s", field)),
			}, nil
		}

		structTI := obj.TypeName
		if structTI.Kind == TypeNamed {
			structTI = structTI.Underlying
		}

		expectedType, ok := structTI.Fields[field]
		if !ok {
			return ErrorValue{
				V: NewRuntimeError(node,
					fmt.Sprintf("unknown field %s", field)),
			}, nil
		}

		actualTI := unwrapAlias(i.typeInfoFromValue(val))
		expectedTI := unwrapAlias(expectedType)

		if !typesAssignable(actualTI, expectedTI) {
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("field '%s' expected '%s' but got '%s'", field, expectedType.Name, actualTI.Name)),
			}, nil
		}

		return val, nil
	case TypeValue:
		if obj.TypeName.Kind != TypeEnum {
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("type '%s' has no members", obj.TypeName.Name)),
			}, nil
		}

		idx, ok := obj.TypeName.Variants[field]
		if !ok {
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("unknown enum variant '%s.%s'", obj.TypeName.Name, field)),
			}, nil
		}

		return EnumValue{
			Enum:    obj.TypeName,
			Variant: field,
			Index:   idx,
		}, nil
	}

	return ErrorValue{
		V: NewRuntimeError(node, fmt.Sprintf("member expression expects enums or structs, but got '%s'", string(left.Type()))),
	}, nil
}

func (i *Interpreter) evalInfix(node *parser.InfixExpression, left Value, op string, right Value) (Value, error) {
	if left.Type() == ERROR {
		return evalErrorInfix(node, left.(ErrorValue), op, right)
	}
	if right.Type() == ERROR {
		return evalErrorInfix(node, right.(ErrorValue), op, left)
	}

	// nil handling
	if _, ok := left.(NilValue); ok {
		return evalNilInfix(node, op, right)
	}
	if _, ok := right.(NilValue); ok {
		return evalNilInfix(node, op, left)
	}

	// strict any handling
	leftTI := unwrapAlias(i.typeInfoFromValue(left))
	rightTI := unwrapAlias(i.typeInfoFromValue(right))

	if leftTI.Kind == TypeAny || rightTI.Kind == TypeAny {
		return ErrorValue{
			V: NewRuntimeError(
				node,
				"cannot use 'thing' in operations, assert a type first",
			),
		}, nil
	}

	// named values
	lnv, lok := left.(NamedValue)
	rnv, rok := right.(NamedValue)

	if lok || rok {
		if !lok || !rok || lnv.TypeName != rnv.TypeName {
			return ErrorValue{
				V: NewRuntimeError(
					node,
					"cannot operate on mismatched named types (try casting)",
				),
			}, nil
		}

		// Same named type → unwrap
		ul := unwrapNamed(left)
		ur := unwrapNamed(right)

		res, err := i.evalInfix(node, ul, op, ur)
		if err != nil {
			return NilValue{}, err
		}
		if v, ok := res.(ErrorValue); ok {
			return v, nil
		}

		// Re-wrap result
		return NamedValue{
			TypeName: lnv.TypeName,
			Value:    res,
		}, nil
	}

	if left.Type() == INT && right.Type() == FLOAT {
		return evalFloatInfix(node, FloatValue{V: float64(left.(IntValue).V)}, op, right.(FloatValue))
	}

	if left.Type() == FLOAT && right.Type() == INT {
		return evalFloatInfix(node, left.(FloatValue), op, FloatValue{V: float64(right.(IntValue).V)})
	}

	// type mismatch check
	if left.Type() != right.Type() {
		return ErrorValue{
			V: NewRuntimeError(node, fmt.Sprintf("type mismatch: '%s' %s '%s'", left.Type(), op, right.Type())),
		}, nil
	}

	switch left.Type() {
	case INT:
		return evalIntInfix(node, left.(IntValue), op, right.(IntValue))
	case FLOAT:
		return evalFloatInfix(node, left.(FloatValue), op, right.(FloatValue))
	case STRING:
		return evalStringInfix(node, left.(StringValue), op, right.(StringValue))
	case BOOL:
		return evalBoolInfix(node, left.(BoolValue), op, right.(BoolValue))
	}

	return ErrorValue{
		V: NewRuntimeError(node, "unsupported operand types"),
	}, nil
}

func evalIntInfix(node *parser.InfixExpression, left IntValue, op string, right IntValue) (Value, error) {
	switch op {
	case "+":
		return IntValue{V: left.V + right.V}, nil
	case "-":
		return IntValue{V: left.V - right.V}, nil
	case "*":
		return IntValue{V: left.V * right.V}, nil
	case "/":
		if right.V == 0 {
			return ErrorValue{
				V: NewRuntimeError(node, "undefined: division by zero"),
			}, nil
		}

		return FloatValue{V: float64(left.V) / float64(right.V)}, nil

	case "%":
		if right.V == 0 {
			return ErrorValue{
				V: NewRuntimeError(node, "undefined: mod by zero"),
			}, nil
		}

		return IntValue{V: left.V % right.V}, nil
	case "==":
		return BoolValue{V: left.V == right.V}, nil
	case "!=":
		return BoolValue{V: left.V != right.V}, nil
	case ">":
		return BoolValue{V: left.V > right.V}, nil
	case "<":
		return BoolValue{V: left.V < right.V}, nil
	case ">=":
		return BoolValue{V: left.V >= right.V}, nil
	case "<=":
		return BoolValue{V: left.V <= right.V}, nil
	}

	return ErrorValue{
		V: NewRuntimeError(node, fmt.Sprintf("invalid operator %d %s %d", left.V, op, right.V)),
	}, nil
}

func evalFloatInfix(node *parser.InfixExpression, left FloatValue, op string, right FloatValue) (Value, error) {
	switch op {
	case "+":
		return FloatValue{V: left.V + right.V}, nil
	case "-":
		return FloatValue{V: left.V - right.V}, nil
	case "*":
		return FloatValue{V: left.V * right.V}, nil
	case "/":
		if right.V == 0 {
			return ErrorValue{
				V: NewRuntimeError(node, "undefined: division by zero"),
			}, nil
		}

		return FloatValue{V: left.V / right.V}, nil
	case "==":
		return BoolValue{V: left.V == right.V}, nil
	case "!=":
		return BoolValue{V: left.V != right.V}, nil
	case ">":
		return BoolValue{V: left.V > right.V}, nil
	case "<":
		return BoolValue{V: left.V < right.V}, nil
	case ">=":
		return BoolValue{V: left.V >= right.V}, nil
	}

	return ErrorValue{
		V: NewRuntimeError(node, fmt.Sprintf("invalid operator %f %s %f", left.V, op, right.V)),
	}, nil
}

func evalStringInfix(node *parser.InfixExpression, left StringValue, op string, right StringValue) (Value, error) {
	switch op {
	case "+":
		return StringValue{V: left.V + right.V}, nil
	case "==":
		return BoolValue{V: left.V == right.V}, nil
	case "!=":
		return BoolValue{V: left.V != right.V}, nil
	}

	return ErrorValue{
		V: NewRuntimeError(node, fmt.Sprintf("invalid operator %s %s %s", left.V, op, right.V)),
	}, nil
}

func evalBoolInfix(node *parser.InfixExpression, left BoolValue, op string, right BoolValue) (Value, error) {
	switch op {
	case "==":
		return BoolValue{V: left.V == right.V}, nil
	case "!=":
		return BoolValue{V: left.V != right.V}, nil
	}

	return ErrorValue{
		V: NewRuntimeError(node, fmt.Sprintf("invalid operator %t %s %t", left.V, op, right.V)),
	}, nil
}

func evalNilInfix(node *parser.InfixExpression, op string, other Value) (Value, error) {
	switch op {
	case "==":
		_, isNil := other.(NilValue)
		return BoolValue{V: isNil}, nil
	case "!=":
		_, isNil := other.(NilValue)
		return BoolValue{V: !isNil}, nil
	default:
		return ErrorValue{
			V: NewRuntimeError(node, fmt.Sprintf("invalid operator nil %s %s", op, other.String())),
		}, nil
	}
}

func evalErrorInfix(node *parser.InfixExpression, left ErrorValue, op string, right Value) (Value, error) {
	re, ok := right.(ErrorValue)
	if !ok {
		switch op {
		case "==":
			return BoolValue{V: false}, nil
		case "!=":
			return BoolValue{V: true}, nil
		}
	}

	switch op {
	case "==":
		return BoolValue{V: left.V.Error() == re.V.Error()}, nil
	case "!=":
		return BoolValue{V: left.V.Error() != re.V.Error()}, nil
	default:
		return ErrorValue{
			V: NewRuntimeError(node, "invalid operator for error"),
		}, nil
	}
}

func evalPrefix(node *parser.PrefixExpression, operator string, right Value) (Value, error) {
	switch operator {
	case "!":
		rTruthy, err := isTruthy(right)
		if err != nil {
			return ErrorValue{
				V: NewRuntimeError(node, err.Error()),
			}, nil
		}

		return BoolValue{V: !rTruthy}, nil
	case "-":
		switch v := right.(type) {
		case IntValue:
			return IntValue{V: -v.V}, nil
		case FloatValue:
			return FloatValue{V: -v.V}, nil
		default:
			return ErrorValue{
				V: NewRuntimeError(node, fmt.Sprintf("invalid operand, %s, for unary '-'", right.String())),
			}, nil
		}
	default:
		return ErrorValue{
			V: NewRuntimeError(node, fmt.Sprintf("unknown prefix operator: %s", operator)),
		}, nil
	}
}

func isTruthy(val Value) (bool, error) {
	b, ok := val.(BoolValue)
	if !ok {
		return false, fmt.Errorf("condition must be boolean")
	}
	return b.V, nil
}
