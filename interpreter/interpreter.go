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
	store    map[string]*Variable
	methods  map[string]map[string]*Func
	builtins map[string]*BuiltinFunc
	defers   []*parser.FuncCall

	mu     sync.RWMutex
	parent *Environment
}

type Interpreter struct {
	Env           *Environment
	typeEnv       map[string]TypeValue
	modules       map[string]ModuleValue
	nativeModules map[string]NativeLoader
	pointerCache  map[*TypeInfo]*TypeInfo
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
	isConst  bool
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
		i.Env.Define(name, mod, false)
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

	i.Env.Define(name, module, false)

	return module, nil
}

func (i *Interpreter) RegisterForward(stmts []parser.Statement) error {
	for _, stmt := range stmts {
		switch stmt := stmt.(type) {
		case *parser.ImportStatement:
			if _, err := i.loadModule(stmt.Name); err != nil {
				return err
			}

		case *parser.TypeStatement:
			ti := &TypeInfo{
				Name:  stmt.Name.Value,
				Kind:  TypeNamed,
				Alias: stmt.Alias,
			}

			i.typeEnv[stmt.Name.Value] = TypeValue{
				TypeInfo: ti,
			}

		case *parser.EnumStatement:
			if _, ok, _ := i.Env.Get(stmt.Name.Value); ok {
				return NewRuntimeError(stmt, fmt.Sprintf("cannot redeclare enum: %s", stmt.Name.Value))
			}

			enumType := &TypeInfo{
				Name:     stmt.Name.Value,
				Kind:     TypeEnum,
				Variants: make(map[string]int),
			}

			for idx, ident := range stmt.Variants {
				name := ident.Value

				if _, exists := enumType.Variants[name]; exists {
					return NewRuntimeError(stmt, fmt.Sprintf("duplicate enum variant: %s", name))
				}

				enumType.Variants[name] = idx
			}

			i.typeEnv[stmt.Name.Value] = TypeValue{TypeInfo: enumType}

		case *parser.FuncStatement:
			paramTypes := make([]*TypeInfo, 0)
			paramNames := make([]string, 0)

			returnTypes := make([]*TypeInfo, 0)
			returnNames := make([]string, 0)

			err := i.checkFuncStatement(stmt)
			if err != nil {
				return err
			}

			for _, typ := range stmt.ReturnTypes {
				ti, err := i.resolveTypeNode(typ)
				if err != nil {
					return err
				}

				ti = unwrapAlias(ti)

				returnTypes = append(returnTypes, ti)
				paramNames = append(paramNames, ti.Name)
			}

			for _, param := range stmt.Params {
				ti, err := i.resolveTypeNode(param.Type)
				if err != nil {
					return err
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

			i.Env.Define(stmt.Name.Value, &Func{Params: stmt.Params, Body: stmt.Body, TypeName: typeInfo, Env: i.Env}, false)
			return nil
		}
	}

	return nil
}

func (i *Interpreter) ResolveTypes(stmts []parser.Statement) error {
	for _, stmt := range stmts {
		switch stmt := stmt.(type) {

		case *parser.TypeStatement:
			tv := i.typeEnv[stmt.Name.Value]
			ti := tv.TypeInfo

			underlying, err := i.resolveTypeNode(stmt.Type)
			if err != nil {
				return err
			}

			ti.Underlying = underlying
		}
	}
	return nil
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

		val, err = i.assignWithType(stmt, val, expectedTI)
		if err != nil {
			return SignalNone{}, err
		}

		// variable must not exist
		if _, ok, _ := i.Env.GetLocal(stmt.Name.Value); ok {
			return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("cant redeclare var: %s", stmt.Name.Value))
		}

		if stmt.Lifetime != nil {
			lifetime, err := i.EvalExpression(stmt.Lifetime)
			if err != nil {
				return SignalNone{}, err
			}

			if lifetime.(IntValue).V > 0 {
				i.Env.DefineWithLifetime(stmt.Name.Value, copyValue(val), lifetime.(IntValue).V+1, false) // +1 because the var statement itself also decrements it
				return SignalNone{}, nil
			}
		}

		if stmt.Name.Value == "_" {
			return SignalNone{}, nil
		}

		i.Env.Define(stmt.Name.Value, copyValue(val), false)
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
		if _, ok, _ := i.Env.GetLocal(stmt.Name.Value); ok {
			return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("cant redeclare var: %s", stmt.Name.Value))
		}

		if stmt.Lifetime != nil {
			lifetime, err := i.EvalExpression(stmt.Lifetime)
			if err != nil {
				return SignalNone{}, err
			}

			if lifetime.(IntValue).V > 0 {
				i.Env.DefineWithLifetime(stmt.Name.Value, copyValue(val), lifetime.(IntValue).V+1, false) // +1 because the var statement itself also decrements it
				return SignalNone{}, nil
			}
		}

		if stmt.Name.Value == "_" {
			return SignalNone{}, nil
		}

		i.Env.Define(stmt.Name.Value, copyValue(val), false)
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
				expectedTI = unwrapAlias(expectedTI)
			}

			for _, name := range stmt.Names {

				if _, ok, _ := i.Env.GetLocal(name.Value); ok {
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
						i.Env.DefineWithLifetime(name.Value, copyValue(v), lifetime.(IntValue).V+1, false) // +1 because the var statement itself also decrements it
					}
				} else {
					i.Env.Define(name.Value, copyValue(v), false)
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

			if tup, ok := val.(TupleValue); ok {
				values = tup.Values
			} else {
				return SignalNone{}, NewRuntimeError(stmt, "multi assign expected multiple values")
			}
		} else {
			values = make([]Value, 0, len(stmt.Values))

			for _, expr := range stmt.Values {
				v, err := i.EvalExpression(expr)
				if err != nil {
					return SignalNone{}, err
				}
				if v, ok := v.(ErrorValue); ok {
					return v, nil
				}

				values = append(values, v)
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

			if _, ok, _ := i.Env.GetLocal(name.Value); ok {
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
					i.Env.DefineWithLifetime(name.Value, copyValue(v), lifetime+1, false)
				}
			} else {
				i.Env.Define(name.Value, copyValue(v), false)
			}
		}

	case *parser.MultiVarStatementNoKeyword:
		var values []Value

		if len(stmt.Values) == 1 {

			val, err := i.EvalExpression(stmt.Values[0])
			if err != nil {
				return SignalNone{}, err
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

			if _, ok, _ := i.Env.GetLocal(name.Value); ok {
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
					i.Env.DefineWithLifetime(name.Value, copyValue(values[idx]), lifetime+1, false)
				}
			} else {
				i.Env.Define(name.Value, copyValue(values[idx]), false)
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
			expectedTI = unwrapAlias(expectedTI)
		}

		if stmt.Value != nil {
			val, err = i.EvalExpression(stmt.Value)
			if err != nil {
				return SignalNone{}, err
			}

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
		if _, ok, _ := i.Env.GetLocal(stmt.Name.Value); ok {
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
				i.Env.DefineWithLifetime(stmt.Name.Value, copyValue(val), lifetime.(IntValue).V+1, true) // +1 because the var statement itself also decrements it
				return SignalNone{}, nil
			}
		}

		// store const val
		i.Env.Define(stmt.Name.Value, copyValue(val), true)
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

		var values []Value

		if len(stmt.Values) == 1 {
			val, err := i.EvalExpression(stmt.Values[0])
			if err != nil {
				return SignalNone{}, err
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

			if _, ok, _ := i.Env.GetLocal(name.Value); ok {
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
					i.Env.DefineWithLifetime(name.Value, copyValue(v), lifetime+1, true)
				}
			} else {
				i.Env.Define(name.Value, copyValue(v), true)
			}
		}

		return SignalNone{}, nil

	case *parser.AssignmentStatement:
		values := make([]Value, 0, len(stmt.Values))

		if len(stmt.Values) == 1 && len(stmt.Targets) > 1 {
			v, err := i.EvalExpression(stmt.Values[0])
			if err != nil {
				return SignalNone{}, err
			}

			if tup, ok := v.(TupleValue); ok {
				values = tup.Values
			} else {
				return SignalNone{}, NewRuntimeError(stmt, "expected multiple values")
			}

		} else {
			for _, expr := range stmt.Values {
				v, err := i.EvalExpression(expr)
				if err != nil {
					return SignalNone{}, err
				}
				values = append(values, v)
			}
		}

		if len(values) != len(stmt.Targets) {
			return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("expected '%d' values but got '%d'", len(stmt.Targets), len(values)))
		}

		targets := make([]Assignable, 0, len(stmt.Targets))

		for _, expr := range stmt.Targets {
			t, err := i.resolveAssignableTarget(expr)
			if err != nil {
				return SignalNone{}, err
			}
			targets = append(targets, t)
		}

		for idx := range targets {
			err := targets[idx].Set(i, copyValue(values[idx]))
			if err != nil {
				return SignalNone{}, err
			}
		}

		return SignalNone{}, nil

	case *parser.MethodStatement:
		recvType, err := i.resolveTypeNode(stmt.Receiver.Type)
		if err != nil {
			return SignalNone{}, NewRuntimeError(stmt, err.Error())
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
		var switchVal Value
		var err error

		if stmt.Value == nil {
			switchVal = BoolValue{V: true}
		} else {
			switchVal, err = i.EvalExpression(stmt.Value)
			if err != nil {
				return SignalNone{}, err
			}
		}

		for _, c := range stmt.Cases {
			matched := false
			for _, expr := range c.Exprs {
				caseVal, err := i.EvalExpression(expr)
				if err != nil {
					return SignalNone{}, err
				}
				if valuesEqual(switchVal, caseVal) {
					matched = true
					break
				}
			}

			if !matched {
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

		i.Env.Define("it", val, true)

		sig, err := i.EvalStatements(stmt.Body)

		i.Env = oldEnv

		return sig, err

	case *parser.ForStatement:
		loopEnv := NewEnvironment(i.Env)
		oldEnv := i.Env

		i.Env = loopEnv
		fmt.Printf("Init stmt type: %T\n", stmt.Init)
		_, err := i.EvalStatement(stmt.Init)
		if err != nil {
			return SignalNone{}, err
		}

		for {
			i.Env = loopEnv

			cond, err := i.EvalExpression(stmt.Condition)
			if err != nil {
				return SignalNone{}, err
			}

			truthy, _ := isTruthy(cond)
			if !truthy {
				break
			}

			i.Env = loopEnv
			sig, err := i.EvalStatements(stmt.Body)
			if err != nil {
				return SignalNone{}, err
			}

			switch sig.(type) {
			case SignalBreak:
				i.Env = oldEnv
				return SignalNone{}, nil
			case SignalContinue:
				i.Env = loopEnv
				_, err := i.EvalStatement(stmt.Post)
				if err != nil {
					return SignalNone{}, err
				}
				continue
			case SignalReturn:
				i.Env = oldEnv
				return sig, nil
			}

			i.Env = loopEnv
			_, err = i.EvalStatement(stmt.Post)
			if err != nil {
				return SignalNone{}, err
			}
		}

		i.Env = oldEnv

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
					i.Env.Define(stmt.Key.Value, IntValue{V: idx}, false)
				}

				if stmt.Value != nil && stmt.Value.Value != "_" {
					i.Env.Define(stmt.Value.Value, copyValue(elem), false)
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
					i.Env.Define(stmt.Key.Value, k, false)
				}

				if stmt.Value != nil && stmt.Value.Value != "_" {
					i.Env.Define(stmt.Value.Value, copyValue(val), false)
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
					i.Env.Define(stmt.Key.Value, IntValue{V: idx}, false)
				}

				if stmt.Value != nil && stmt.Value.Value != "_" {
					i.Env.Define(stmt.Value.Value, StringValue{V: string(s)}, false)
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
					i.Env.Define(stmt.Key.Value, IntValue{V: idx}, false)
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
			TypeInfo: ti,
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
			return NilValue{}, NewRuntimeError(expr, "cannot use '_' as a value")
		}

		if v, ok := i.typeEnv[expr.Value]; ok {
			return v, nil
		}

		v, ok, _ := i.Env.Get(expr.Value)
		if !ok {
			return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("undefined variable: %s", expr.Value))
		}

		return v, nil

	case *parser.CompositeLiteral:
		ti, err := i.resolveTypeNode(expr.Type)
		if err != nil {
			return NilValue{}, err
		}
		return i.evalCompositeLiteral(expr, ti)

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
			return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("type mismatch: '%s' asserted as '%s'", actualTI.Name, targetTI.Name))
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

		return i.evalPrefix(expr, expr.Operator, right)

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

		return NilValue{}, NewRuntimeError(expr, "in expects map or array or string")

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
		return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("unhandled expression type: %T", e))
	}
}

func (i *Interpreter) evalCompositeLiteral(expr *parser.CompositeLiteral, ti *TypeInfo) (Value, error) {
	ti = unwrapAlias(ti)

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
			return NilValue{}, NewRuntimeError(expr,
				fmt.Sprintf("%s is not a struct type", typeInfo.Name))
		}
		structType = typeInfo.Underlying

	default:
		return NilValue{}, NewRuntimeError(expr,
			fmt.Sprintf("%s is not a struct type", typeInfo.Name))
	}

	fields := make(map[string]Value)

	for name, e := range expr.Fields {
		expectedType, ok := structType.Fields[name]
		if !ok {
			return NilValue{}, NewRuntimeError(
				expr,
				fmt.Sprintf("unknown field '%s' in struct %s",
					name, typeInfo.Name),
			)
		}

		v, err := i.EvalExpression(e)
		if err != nil {
			return NilValue{}, err
		}

		actualTI := unwrapAlias(i.typeInfoFromValue(v))
		expectedTI := unwrapAlias(expectedType)

		if !typesAssignable(actualTI, expectedTI) {
			return NilValue{}, NewRuntimeError(
				expr,
				fmt.Sprintf(
					"field '%s' expects '%s' but got '%s'",
					name,
					expectedType.Name,
					actualTI.Name,
				),
			)
		}

		v = i.promoteValueToType(v, expectedTI)

		if err := validateRange(expr, v, expectedTI); err != nil {
			return NilValue{}, err
		}

		fields[name] = v
	}

	for name, ti := range structType.Fields {
		if _, ok := fields[name]; !ok {
			def, err := i.defaultValueFromTypeInfo(expr, ti)
			if err != nil {
				return NilValue{}, err
			}
			fields[name] = def
		}
	}

	valueType := typeInfo
	if typeInfo.Kind == TypeStruct {
		valueType = structType
	}

	v := &StructValue{
		TypeName: valueType,
		Fields:   fields,
	}
	return copyValue(v), nil
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

		err = validateRange(expr, val, elemType)
		if err != nil {
			return NilValue{}, err
		}

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
			return NilValue{}, NewRuntimeError(expr, "cannot infer type of empty map")
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
			return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("map key type %s is not comparable", keyTI.Name))
		}

		if !typesAssignable(keyTI, expected.Key) {
			return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("type mismatch: map key 0 expected %s but got %s", expected.Key.Name, keyTI.Name))
		}
		keyTI = expected.Key

		if !typesAssignable(valTI, expected.Value) {
			return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("type mismatch: map value 0 expected %s but got %s", expected.Value.Name, valTI.Name))
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
			return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("map key %d is not comparable", idx))
		}

		if !typesAssignable(kt, keyTI) {
			return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("map key %d expected %s but got %s", idx, keyTI.Name, kt.Name))
		}

		if !typesAssignable(vt, valTI) {
			return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("map value %d expected %s but got %s", idx, valTI.Name, vt.Name))
		}

		if err := validateRange(expr, k, keyTI); err != nil {
			return NilValue{}, err
		}

		if err := validateRange(expr, v, valTI); err != nil {
			return NilValue{}, err
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
				return NilValue{}, NewRuntimeError(e, "type cast expects 1 arg")
			}
			return i.evalTypeCast(ti.TypeInfo, e.Args[0], e)
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
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("int type cast does not support %s, try the function toInt to parse non-numeric types", string(v.Type())))
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
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("float type cast does not support %s, try the function toFloat to parse non-numeric types", string(v.Type())))
		}

		return FloatValue{V: val}, nil
	case TypeString:
		if s, ok := v.(StringValue); ok {
			return s, nil
		}
		if v, ok := v.(ErrorValue); ok {
			return v, nil
		}

		return NilValue{}, NewRuntimeError(node, fmt.Sprintf("string cast does not support %s, try the function toString to parse other types", string(v.Type())))
	case TypeBool:
		var val bool

		if v, ok := v.(ErrorValue); ok {
			return v, nil
		}

		switch v := v.(type) {
		case BoolValue:
			val = v.V
		default:
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("bool type cast does not support %s, try the function toBool to parse other types", string(v.Type())))
		}

		return BoolValue{V: val}, nil

	case TypeArray:
		if a, ok := v.(ArrayValue); ok {
			return a, nil
		}
		if v, ok := v.(ErrorValue); ok {
			return v, nil
		}

		return NilValue{}, NewRuntimeError(node, fmt.Sprintf("array cast does not support %s, try the function toArr to construct arrays", string(v.Type())))
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

		return NilValue{}, NewRuntimeError(node, fmt.Sprintf("error cast does not support %s", string(v.Type())))
	default:
		return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown type cast: %s", target.Name))
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
		return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("expected 'function' but got '%s'", unwrapAlias(i.typeInfoFromValue(val)).Name))
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
			return NilValue{}, NewRuntimeError(callNode, fmt.Sprintf("expected %d args, got %d", paramCount, argCount))
		}
	} else {
		fixedCount := paramCount - 1
		if argCount < fixedCount {
			return NilValue{}, NewRuntimeError(callNode, fmt.Sprintf("expected atleast %d args, got %d", fixedCount, argCount))
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
				return NilValue{}, NewRuntimeError(callNode, fmt.Sprintf("param '%s' expects '%s' but got '%s'", param.Name.Value, expected.Name, actual.Name))
			}
		}

		newEnv.Define(param.Name.Value, val, false)
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

		newEnv.Define(variadicParam.Name.Value, sliceValue, false)
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
			return NilValue{}, NewRuntimeError(callNode,
				fmt.Sprintf("expected %d return values, got %d",
					len(fn.TypeName.Returns), len(ret.Values)))
		}

		for idx, expectedType := range fn.TypeName.Returns {
			actual := ret.Values[idx]

			if err != nil {
				return NilValue{}, NewRuntimeError(callNode, err.Error())
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
		return NilValue{}, NewRuntimeError(node, "cannot index value of type 'thing' without type assertion")
	}

	left = unwrapNamed(left)

	typ := i.typeInfoFromValue(left)

	switch typ.Kind {
	case TypeArray, TypeFixedArray:
		arr, ok := left.(ArrayValue)

		idxVal, ok := idx.(IntValue)
		if !ok {
			return NilValue{}, NewRuntimeError(node, "index: must be int")
		}

		idx := idxVal.V

		if idx < 0 || idx >= len(arr.Elements) {
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("index: %d, out of bounds", idx))
		}

		elem := arr.Elements[idx]

		elemType := unwrapAlias(i.typeInfoFromValue(elem))

		if !typesAssignable(elemType, arr.ElemType) {
			return NilValue{}, NewRuntimeError(node,
				fmt.Sprintf("array element expected %s but got %s",
					arr.ElemType.Name, elemType.Name))
		}

		if err := validateRange(node, elem, arr.ElemType); err != nil {
			return NilValue{}, err
		}

		if arr.ElemType.Kind == TypeAny {
			elem = NamedValue{
				TypeName: arr.ElemType,
				Value:    elem,
			}
		}

		return copyValue(elem), nil

	case TypeString:
		idxVal, ok := idx.(IntValue)
		if !ok {
			return NilValue{}, NewRuntimeError(node, "index must be int")
		}

		idx := idxVal.V

		if idx < 0 || idx >= len(left.(StringValue).V) {
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("index: %d, out of bounds", idx))
		}

		r := []rune(left.(StringValue).V)
		return StringValue{V: string(r[idx])}, nil

	case TypeMap:
		mv := left.(MapValue)

		// 1. type check key
		keyType := unwrapAlias(i.typeInfoFromValue(idx))

		if mv.KeyType.Kind == TypeAny {
			if !isComparableValue(idx) {
				return NilValue{}, NewRuntimeError(
					node,
					"value of this type cannot be used as map key",
				)
			}
		} else {
			if !typesAssignable(keyType, mv.KeyType) {
				return NilValue{}, NewRuntimeError(
					node,
					fmt.Sprintf(
						"map index expected %s but got %s",
						mv.KeyType.Name,
						keyType.Name,
					),
				)
			}

			if err := validateRange(node, idx, mv.KeyType); err != nil {
				return NilValue{}, err
			}
		}

		val, ok := mv.Entries[idx]
		if !ok {
			return NilValue{}, nil
		}

		valType := unwrapAlias(i.typeInfoFromValue(val))

		if !typesAssignable(valType, mv.ValueType) {
			return NilValue{}, NewRuntimeError(node,
				fmt.Sprintf("map value expected %s but got %s",
					mv.ValueType.Name, valType.Name))
		}

		if err := validateRange(node, val, mv.ValueType); err != nil {
			return NilValue{}, err
		}

		if mv.ValueType.Kind == TypeAny {
			return NamedValue{
				TypeName: mv.ValueType,
				Value:    val,
			}, nil
		}

		return val, nil

	default:
		types := map[TypeKind]string{
			TypeInt:        "int",
			TypeFloat:      "float",
			TypeString:     "string",
			TypeBool:       "bool",
			TypeArray:      "slice",
			TypeFixedArray: "array",
			TypeFunc:       "function",
			TypeNil:        "nil",
			TypeStruct:     "struct",
			TypeMap:        "map",
			TypeEnum:       "enum",
			TypeError:      "error",
		}

		typeStr, ok := types[typ.Kind]

		if ok {
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("indexing is not allowed with type: '%s'", typeStr))
		}

		return NilValue{}, NewRuntimeError(node, fmt.Sprintf("indexing is not allowed with type: %d", typ.Kind))
	}
}

func (i *Interpreter) evalSliceExpression(node parser.Expression, left, startVal, endVal Value) (Value, error) {
	if nv, ok := left.(NamedValue); ok && nv.TypeName.Kind == TypeAny {
		return NilValue{}, NewRuntimeError(node,
			"cannot slice value of type 'thing' without type assertion")
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
		return NilValue{}, NewRuntimeError(node,
			fmt.Sprintf("slicing is not allowed with type: '%s'", typ.Name))
	}

	start := 0
	end := length

	if _, ok := startVal.(NilValue); !ok {
		intVal, ok := startVal.(IntValue)
		if !ok {
			return NilValue{}, NewRuntimeError(node, "slice start must be int")
		}
		start = intVal.V
	}

	if _, ok := endVal.(NilValue); !ok {
		intVal, ok := endVal.(IntValue)
		if !ok {
			return NilValue{}, NewRuntimeError(node, "slice end must be int")
		}
		end = intVal.V
	}

	if start < 0 || end < 0 || start > end || end > length {
		return NilValue{}, NewRuntimeError(node,
			fmt.Sprintf("slice bounds out of range [%d:%d]", start, end))
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
	if ptr, ok := left.(*PointerValue); ok {
		if ptr.Target == nil || ptr.Target.Value == nil {
			return NilValue{}, NewRuntimeError(node, "nil pointer dereference")
		}
		left = ptr.Target.Value
	}

	if left == nil {
		return NilValue{}, NewRuntimeError(node, "nil value in member expression")
	}
	recvType := unwrapAlias(i.typeInfoFromValue(left))
	ptrType := i.pointerTo(recvType)

	if fn, ok := i.Env.GetMethod(ptrType, field); ok {
		tmp := &Variable{Value: left}
		return BoundMethodValue{
			Receiver: &PointerValue{Target: tmp, ElemType: recvType},
			Func:     fn,
		}, nil
	}

	if fn, ok := i.Env.GetMethod(recvType, field); ok {
		recv := left

		if sv, ok := left.(*StructValue); ok {
			newFields := make(map[string]Value)
			for k, v := range sv.Fields {
				newFields[k] = v
			}

			recv = &StructValue{
				TypeName: sv.TypeName,
				Fields:   newFields,
			}
		}

		return BoundMethodValue{
			Receiver: recv,
			Func:     fn,
		}, nil
	}

	switch obj := left.(type) {
	case ModuleValue:
		if typ, ok := obj.typeEnv[field]; ok {
			return typ, nil
		}

		val, ok, _ := obj.Env.Get(field)
		if !ok {
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown '%s'", field))
		}

		return val, nil
	case *StructValue:
		val, ok := obj.Fields[field]
		if !ok {
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown field %s", field))
		}

		structTI := obj.TypeName
		if structTI.Kind == TypeNamed {
			structTI = structTI.Underlying
		}

		expectedType, ok := structTI.Fields[field]
		if !ok {
			return NilValue{}, NewRuntimeError(node,
				fmt.Sprintf("unknown field %s", field))
		}

		actualTI := unwrapAlias(i.typeInfoFromValue(val))
		expectedTI := unwrapAlias(expectedType)

		if !typesAssignable(actualTI, expectedTI) {
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("field '%s' expected '%s' but got '%s'", field, expectedType.Name, actualTI.Name))
		}

		return val, nil
	case TypeValue:
		if obj.TypeInfo.Kind != TypeEnum {
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("type '%s' has no members", obj.TypeInfo.Name))
		}

		idx, ok := obj.TypeInfo.Variants[field]
		if !ok {
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown enum variant '%s.%s'", obj.TypeInfo.Name, field))
		}

		return EnumValue{
			Enum:    obj.TypeInfo,
			Variant: field,
			Index:   idx,
		}, nil
	}

	return NilValue{}, NewRuntimeError(node, fmt.Sprintf("member expression expects enums or structs, but got '%s'", string(left.Type())))
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
		return NilValue{}, NewRuntimeError(
			node,
			"cannot use 'thing' in operations, assert a type first",
		)
	}

	// named values
	lnv, lok := left.(NamedValue)
	rnv, rok := right.(NamedValue)

	if lok || rok {
		if !lok || !rok || lnv.TypeName != rnv.TypeName {
			return NilValue{}, NewRuntimeError(
				node,
				"cannot operate on mismatched named types (try casting)",
			)
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
		return NilValue{}, NewRuntimeError(node, fmt.Sprintf("type mismatch: '%s' %s '%s'", left.Type(), op, right.Type()))
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
	case ENUM:
		return evalIntInfix(node, IntValue{V: left.(EnumValue).Index}, op, IntValue{V: right.(EnumValue).Index})
	}

	return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unsupported operand types: %s %s %s", i.typeInfoFromValue(left).Name, op, i.typeInfoFromValue(right).Name))
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
			return NilValue{}, NewRuntimeError(node, "undefined: division by zero")
		}

		return FloatValue{V: float64(left.V) / float64(right.V)}, nil

	case "%":
		if right.V == 0 {
			return NilValue{}, NewRuntimeError(node, "undefined: mod by zero")
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

	return NilValue{}, NewRuntimeError(node, fmt.Sprintf("invalid operator %d %s %d", left.V, op, right.V))
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
			return NilValue{}, NewRuntimeError(node, "undefined: division by zero")
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

	return NilValue{}, NewRuntimeError(node, fmt.Sprintf("invalid operator %f %s %f", left.V, op, right.V))
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

	return NilValue{}, NewRuntimeError(node, fmt.Sprintf("invalid operator %s %s %s", left.V, op, right.V))
}

func evalBoolInfix(node *parser.InfixExpression, left BoolValue, op string, right BoolValue) (Value, error) {
	switch op {
	case "==":
		return BoolValue{V: left.V == right.V}, nil
	case "!=":
		return BoolValue{V: left.V != right.V}, nil
	}

	return NilValue{}, NewRuntimeError(node, fmt.Sprintf("invalid operator %t %s %t", left.V, op, right.V))
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
		return NilValue{}, NewRuntimeError(node, fmt.Sprintf("invalid operator nil %s %s", op, other.String()))
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
		return NilValue{}, NewRuntimeError(node, "invalid operator for error")
	}
}

func (i *Interpreter) evalPrefix(node *parser.PrefixExpression, operator string, right Value) (Value, error) {
	switch operator {

	case "!":
		rTruthy, err := isTruthy(right)
		if err != nil {
			return NilValue{}, NewRuntimeError(node, err.Error())
		}
		return BoolValue{V: !rTruthy}, nil

	case "-":
		switch v := right.(type) {
		case IntValue:
			return IntValue{V: -v.V}, nil
		case FloatValue:
			return FloatValue{V: -v.V}, nil
		default:
			return NilValue{}, NewRuntimeError(node, "invalid operand for unary '-'")
		}

	case "&":
		ident, ok := node.Right.(*parser.Identifier)
		if !ok {
			return NilValue{}, NewRuntimeError(node, "cannot take address of expression")
		}
		v, ok := i.Env.GetVar(ident.Value)
		if !ok {
			return NilValue{}, NewRuntimeError(node, "undefined variable")
		}

		ptr := &PointerValue{Target: v, ElemType: i.typeInfoFromValue(v.Value)}
		return ptr, nil

	case "*":
		ptr, ok := right.(*PointerValue)
		if !ok {
			return NilValue{}, NewRuntimeError(node, "cannot dereference a non-pointer")
		}
		if ptr.Target == nil {
			return NilValue{}, NewRuntimeError(node, "nil pointer dereference")
		}
		return ptr.Target.Value, nil
	}

	return NilValue{}, NewRuntimeError(node, "unknown prefix operator")
}

func isTruthy(val Value) (bool, error) {
	b, ok := val.(BoolValue)
	if !ok {
		return false, fmt.Errorf("condition must be boolean")
	}
	return b.V, nil
}
