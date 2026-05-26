package interpreter

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"runtime"
	"strings"

	"os"
	"sync"

	"github.com/z-sk1/ayla-lang/lexer"
	"github.com/z-sk1/ayla-lang/parser"
	"github.com/z-sk1/ayla-lang/token"
)

type Environment struct {
	store    map[string]*Variable
	builtins map[string]*BuiltinFunc
	defers   []*parser.DeferStatement

	mu     sync.RWMutex
	parent *Environment
}

type Interpreter struct {
	Env          *Environment
	TypeEnv      map[string]TypeValue
	pointerCache map[*TypeInfo]*TypeInfo
	modulePaths  []string
	currentDir   string
	projectRoot  string

	Wg sync.WaitGroup
}

var GlobalModules map[string]ModuleValue = map[string]ModuleValue{}
var NativeModules map[string]NativeLoader = map[string]NativeLoader{}

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

var compoundOps = map[token.TokenType]string{
	token.PLUS_ASSIGN:  "+",
	token.SUB_ASSIGN:   "-",
	token.MUL_ASSIGN:   "*",
	token.SLASH_ASSIGN: "/",
	token.MOD_ASSIGN:   "%",

	token.AND_ASSIGN: "&",
	token.OR_ASSIGN:  "|",
	token.XOR_ASSIGN: "^",
	token.SHL_ASSIGN: "<<",
	token.SHR_ASSIGN: ">>",
}

type EvalResult struct {
	Values []Value
	Err    error
}

func (r EvalResult) First() Value {
	if len(r.Values) == 0 {
		return NilValue{}
	}
	return r.Values[0]
}

func (r EvalResult) MustSingle(node parser.Node) (Value, error) {
	if len(r.Values) != 1 {
		return NilValue{}, NewRuntimeError(node,
			fmt.Sprintf("expected 1 value, got %d", len(r.Values)))
	}
	return r.Values[0], nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (i *Interpreter) resolveModule(name string) (string, error) {
	exts := []string{".ayla", ".ayl"}

	wd, _ := os.Getwd()

	ud, _ := os.UserHomeDir()

	searchPaths := []string{
		filepath.Join(i.currentDir, name),
		filepath.Join(i.currentDir, "lib"),
		i.currentDir,
		filepath.Join(i.currentDir, "lib", name),
		filepath.Join(wd, "lib", name),
		filepath.Join(ud, ".ayla", "lib", name),
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
	if mod, ok := GlobalModules[name]; ok {
		i.Env.Define(name, mod, false)
		return mod, nil
	}

	if loader, ok := NativeModules[name]; ok {
		mod, err := loader(i)
		if err != nil {
			return NilValue{}, err
		}

		GlobalModules[name] = mod
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
	modInterp.TypeEnv = i.TypeEnv
	modInterp.currentDir = filepath.Dir(path)

	if err := modInterp.RegisterForward(program); err != nil {
		return NilValue{}, err
	}

	if err := modInterp.ResolveTypes(program); err != nil {
		return NilValue{}, err
	}

	if err := modInterp.TypeCheck(program); err != nil {
		return NilValue{}, err
	}

	_, err = modInterp.EvalStatements(program)
	if err != nil {
		return NilValue{}, err
	}

	module := ModuleValue{
		Name:    name,
		Env:     Env,
		TypeEnv: modInterp.TypeEnv,
	}

	for name, typ := range modInterp.TypeEnv {
		i.TypeEnv[name] = typ
	}

	i.Env.Define(name, module, false)

	return module, nil
}

func (i *Interpreter) RegisterForward(stmts []parser.Statement) error {
	for _, stmt := range stmts {
		switch stmt := stmt.(type) {
		case *parser.ImportStatement:
			mod, err := i.loadModule(stmt.Name)
			if err != nil {
				return err
			}

			i.Env.Define(stmt.Name, mod, false)

		case *parser.TypeStatement:
			ti := &TypeInfo{
				Name:        stmt.Name.Value,
				Kind:        TypeNamed,
				Alias:       stmt.Alias,
				Methods:     make(map[string]*Func),
				MethodTypes: make(map[string]*TypeInfo),
			}

			i.TypeEnv[stmt.Name.Value] = TypeValue{
				TypeInfo: ti,
			}

		case *parser.EnumStatement:
			if _, ok, _ := i.Env.Get(stmt.Name.Value); ok {
				return NewRuntimeError(stmt, fmt.Sprintf("cannot redeclare enum: %s", stmt.Name.Value))
			}

			elemTI, err := i.resolveTypeNode(stmt.Type)
			if err != nil {
				return err
			}

			enumType := &TypeInfo{
				Name:         stmt.Name.Value,
				Kind:         TypeEnum,
				Elem:         elemTI,
				Variants:     make(map[string]*EnumVariant),
				VariantOrder: make([]string, 0),
				Nested:       make(map[string]*TypeInfo),
			}

			for idx, member := range stmt.Members {

				if nested, ok := member.(*parser.EnumStatement); ok {
					err := i.RegisterForward([]parser.Statement{nested})
					if err != nil {
						return err
					}
					continue
				}

				variant := member.(*parser.Variant)
				name := variant.Name.Value

				if _, exists := enumType.Variants[name]; exists {
					return NewRuntimeError(stmt, fmt.Sprintf("duplicate enum variant: %s", name))
				}

				var val Value

				if variant.Value != nil {
					v, err := i.evalOne(variant.Value)
					if err != nil {
						return err
					}
					val = v

					if !TypesAssignable(i.TypeInfoFromValue(val), enumType.Elem) {
						return NewRuntimeError(stmt, fmt.Sprintf("type mismatch: variant '%s' expected '%s' but got '%s'", name, enumType.Elem.Name, i.TypeInfoFromValue(val).Name))
					}
				} else {
					val = IntValue{V: idx}

					if !TypesAssignable(i.TypeInfoFromValue(val), enumType.Elem) {
						return NewRuntimeError(stmt, fmt.Sprintf("type mismatch: variant '%s' expected '%s' but got '%s'", name, enumType.Elem.Name, i.TypeInfoFromValue(val).Name))
					}
				}

				enumType.Variants[name] = &EnumVariant{
					Name:  name,
					Index: idx,
					Value: val,
				}

				enumType.VariantOrder = append(enumType.VariantOrder, name)
			}

			i.TypeEnv[stmt.Name.Value] = TypeValue{TypeInfo: enumType}

		case *parser.MethodStatement:
			recvType, err := i.resolveTypeNode(stmt.Receiver.Type)
			if err != nil {
				return err
			}

			recvType = UnwrapAlias(recvType)

			i.Env.SetMethod(recvType, stmt.Name.Value, &Func{
				Body:    stmt.Body,
				Env:     i.Env,
				TypeEnv: i.TypeEnv,
			})

		case *parser.FuncStatement:
			i.Env.Define(stmt.Name.Value, &Func{
				Params:  stmt.Params,
				Body:    stmt.Body,
				Env:     i.Env,
				TypeEnv: i.TypeEnv,
			}, false)

		}
	}

	return nil
}

func (i *Interpreter) ResolveTypes(stmts []parser.Statement) error {
	for _, stmt := range stmts {
		switch stmt := stmt.(type) {

		case *parser.TypeStatement:
			tv := i.TypeEnv[stmt.Name.Value]
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

func (i *Interpreter) TypeCheck(stmts []parser.Statement) error {
	for _, stmt := range stmts {
		switch stmt := stmt.(type) {

		case *parser.FuncStatement:
			paramTypes := []*TypeInfo{}
			paramNames := []string{}
			returnTypes := []*TypeInfo{}
			returnNames := []string{}

			for _, typ := range stmt.ReturnTypes {
				ti, err := i.resolveTypeNode(typ)
				if err != nil {
					return err
				}
				ti = UnwrapAlias(ti)
				returnTypes = append(returnTypes, ti)
				returnNames = append(returnNames, ti.Name)
			}

			for _, param := range stmt.Params {
				ti, err := i.resolveTypeNode(param.Type)
				if err != nil {
					return err
				}
				ti = UnwrapAlias(ti)
				paramTypes = append(paramTypes, ti)
				paramNames = append(paramNames, ti.Name)
			}

			typeInfo := &TypeInfo{
				Name:    fmt.Sprintf("fun(%s) (%s)", strings.Join(paramNames, ", "), strings.Join(returnNames, ", ")),
				Kind:    TypeFunc,
				Returns: returnTypes,
				Params:  paramTypes,
			}

			variable, ok := i.Env.GetVar(stmt.Name.Value)
			if !ok {
				return fmt.Errorf("function not found: %s", stmt.Name.Value)
			}

			variable.Value.(*Func).TypeName = typeInfo

			if err := i.checkFuncStatement(stmt); err != nil {
				return err
			}

		case *parser.MethodStatement:
			recvType, err := i.resolveTypeNode(stmt.Receiver.Type)
			if err != nil {
				return NewRuntimeError(stmt, err.Error())
			}

			recvType = UnwrapAlias(recvType)

			params := append(
				[]*parser.Param{
					{
						Name: stmt.Receiver.Name,
						Type: stmt.Receiver.Type,
					},
				},
				stmt.Params...,
			)

			paramTypes := []*TypeInfo{}
			paramNames := []string{}
			returnTypes := []*TypeInfo{}
			returnNames := []string{}

			err = i.checkMethodStatement(stmt)
			if err != nil {
				return err
			}

			for _, typ := range stmt.ReturnTypes {
				ti, err := i.resolveTypeNode(typ)
				if err != nil {
					return err
				}
				ti = UnwrapAlias(ti)

				returnTypes = append(returnTypes, ti)
				returnNames = append(returnNames, ti.Name)
			}

			for _, param := range stmt.Params {
				ti, err := i.resolveTypeNode(param.Type)
				if err != nil {
					return err
				}
				ti = UnwrapAlias(ti)

				paramTypes = append(paramTypes, ti)
				paramNames = append(paramNames, ti.Name)
			}

			typeInfo := &TypeInfo{
				Name:    fmt.Sprintf("fun(%s, %s) (%s)", recvType.Name, strings.Join(paramNames, ", "), strings.Join(returnNames, ", ")),
				Kind:    TypeFunc,
				Returns: returnTypes,
				Params:  paramTypes,
			}

			method, ok := recvType.Methods[stmt.Name.Value]
			if !ok {
				return fmt.Errorf("method not found: %s", stmt.Name.Value)
			}

			method.Params = params
			method.TypeName = typeInfo

			if err := i.checkMethodStatement(stmt); err != nil {
				return err
			}

		}
	}
	return nil
}

func (i *Interpreter) EvalProgram(stmts []parser.Statement) (Value, error) {
	var last Value
	for _, s := range stmts {
		sig, err := i.EvalStatement(s)
		if err != nil {
			return nil, err
		}
		switch v := sig.(type) {
		case SignalValue:
			last = v.Value
		case SignalReturn:
			return TupleValue{Values: v.Values}, nil
		}
		i.tickLifetimes()
	}
	return UnwrapFully(last), nil
}

func (i *Interpreter) EvalStatements(stmts []parser.Statement) (ControlSignal, error) {
	for _, s := range stmts {
		sig, err := i.EvalStatement(s)
		if err != nil {
			return SignalNone{}, err
		}

		switch sig.(type) {
		case SignalReturn, SignalBreak, SignalContinue:
			return sig, nil
		}

		i.tickLifetimes()
	}

	return SignalNone{}, nil
}

func (i *Interpreter) EvalBlock(stmts []parser.Statement, newScope bool, vars map[string]Value) (ControlSignal, error) {
	blockEnv := NewEnvironment(i.Env)
	oldEnv := i.Env

	if newScope {
		i.Env = blockEnv
	}

	for k, v := range vars {
		i.Env.Define(k, v, false)
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
		}

		if stmt.Value != nil {
			res, err := i.EvalExpression(stmt.Value)
			if err != nil {
				return SignalNone{}, err
			}

			vals, err := i.unpackForAssign(stmt, res, 1)
			if err != nil {
				return SignalNone{}, err
			}

			val = vals[0]
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
			lifetime, err := i.evalOne(stmt.Lifetime)
			if err != nil {
				return SignalNone{}, err
			}

			lifetime = UnwrapFully(lifetime)

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
			_, err := i.EvalStatement(decl)
			if err != nil {
				return SignalNone{}, err
			}
		}
		return SignalNone{}, nil

	case *parser.VarStatementNoKeyword:
		res, err := i.EvalExpression(stmt.Value)
		if err != nil {
			return SignalNone{}, err
		}

		vals, err := i.unpackForAssign(stmt, res, 1)
		if err != nil {
			return SignalNone{}, err
		}

		val := vals[0]

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
			lifetime, err := i.evalOne(stmt.Lifetime)
			if err != nil {
				return SignalNone{}, err
			}

			lifetime = UnwrapFully(lifetime)

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
					lifetime, err := i.evalOne(stmt.Lifetime)
					if err != nil {
						return SignalNone{}, err
					}

					lifetime = UnwrapFully(lifetime)

					if lifetime.(IntValue).V > 0 {
						i.Env.DefineWithLifetime(name.Value, copyValue(v), lifetime.(IntValue).V+1, false) // +1 because the var statement itself also decrements it
						return SignalNone{}, nil
					}
				} else {
					i.Env.Define(name.Value, copyValue(v), false)
				}
			}

			return SignalNone{}, nil
		}

		var values []Value
		var err error

		if len(stmt.Values) == 1 {
			res, err := i.EvalExpression(stmt.Values[0])
			if err != nil {
				return SignalNone{}, err
			}

			values, err = i.unpackForAssign(stmt, res, len(stmt.Names))
			if err != nil {
				return SignalNone{}, err
			}
		} else {
			values = make([]Value, len(stmt.Values))
			for idx, expr := range stmt.Values {
				res, err := i.EvalExpression(expr)
				if err != nil {
					return SignalNone{}, err
				}

				vals, err := i.unpackForAssign(stmt, res, 1)
				if err != nil {
					return SignalNone{}, err
				}
				values[idx] = vals[0]
			}
		}

		if len(values) != len(stmt.Names) {
			return SignalNone{}, NewRuntimeError(stmt,
				fmt.Sprintf("expected %d values, got %d",
					len(stmt.Names), len(stmt.Values)))
		}

		var expectedTI *TypeInfo
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
				lifetime, err := i.evalOne(stmt.Lifetime)
				if err != nil {
					return SignalNone{}, err
				}

				lifetime = UnwrapFully(lifetime)

				if lifetime.(IntValue).V > 0 {
					i.Env.DefineWithLifetime(name.Value, copyValue(v), lifetime.(IntValue).V+1, false) // +1 because the var statement itself also decrements it
					return SignalNone{}, nil
				}
			} else {
				i.Env.Define(name.Value, copyValue(v), false)
			}
		}

	case *parser.MultiVarStatementNoKeyword:
		var values []Value

		if len(stmt.Values) == 1 {
			res, err := i.EvalExpression(stmt.Values[0])
			if err != nil {
				return SignalNone{}, err
			}

			values, err = i.unpackForAssign(stmt, res, len(stmt.Names))
			if err != nil {
				return SignalNone{}, err
			}
		} else {
			values = make([]Value, len(stmt.Values))
			for idx, expr := range stmt.Values {
				res, err := i.EvalExpression(expr)
				if err != nil {
					return SignalNone{}, err
				}

				vals, err := i.unpackForAssign(stmt, res, 1)
				if err != nil {
					return SignalNone{}, err
				}
				values[idx] = vals[0]
			}
		}

		if len(values) != len(stmt.Names) {
			return SignalNone{}, NewRuntimeError(
				stmt,
				fmt.Sprintf("expected %d values, got %d",
					len(stmt.Names), len(values)),
			)
		}

		hasNew := false

		for _, name := range stmt.Names {
			if name.Value == "_" {
				continue
			}
			if _, exists, _ := i.Env.GetLocal(name.Value); !exists {
				hasNew = true
			}
		}

		if !hasNew {
			return SignalNone{}, NewRuntimeError(stmt,
				"no new variables on left side of :=")
		}

		for idx, name := range stmt.Names {
			if name.Value == "_" {
				continue
			}

			if _, exists, _ := i.Env.GetLocal(name.Value); exists {
				i.Env.Set(name.Value, copyValue(values[idx]))
			} else {
				if stmt.Lifetime != nil {
					lifetime, err := i.evalOne(stmt.Lifetime)
					if err != nil {
						return SignalNone{}, err
					}

					lifetime = UnwrapFully(lifetime)

					if lifetime.(IntValue).V > 0 {
						i.Env.DefineWithLifetime(name.Value, copyValue(values[idx]), lifetime.(IntValue).V+1, false) // +1 because the var statement itself also decrements it
						return SignalNone{}, nil
					}
				} else {
					i.Env.Define(name.Value, copyValue(values[idx]), false)
				}
			}
		}

		return SignalNone{}, nil

	case *parser.ConstStatementBlock:
		for _, decl := range stmt.Decls {
			_, err := i.EvalStatement(decl)
			if err != nil {
				return SignalNone{}, err
			}
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
			val, err = i.evalOne(stmt.Value)
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
			lifetime, err := i.evalOne(stmt.Lifetime)
			if err != nil {
				return SignalNone{}, err
			}

			lifetime = UnwrapFully(lifetime)

			if lifetime.(IntValue).V > 0 {
				i.Env.DefineWithLifetime(stmt.Name.Value, copyValue(val), lifetime.(IntValue).V+1, false) // +1 because the var statement itself also decrements it
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
		var err error

		if len(stmt.Values) == 1 {
			res, err := i.EvalExpression(stmt.Values[0])
			if err != nil {
				return SignalNone{}, err
			}

			values, err = i.unpackForAssign(stmt, res, len(stmt.Names))
			if err != nil {
				return SignalNone{}, err
			}
		} else {
			values = make([]Value, len(stmt.Values))
			for idx, expr := range stmt.Values {
				res, err := i.EvalExpression(expr)
				if err != nil {
					return SignalNone{}, err
				}

				vals, err := i.unpackForAssign(stmt, res, 1)
				if err != nil {
					return SignalNone{}, err
				}
				values[idx] = vals[0]
			}
		}

		if len(values) != len(stmt.Names) {
			return SignalNone{}, NewRuntimeError(stmt,
				fmt.Sprintf("expected %d values, got %d",
					len(stmt.Names), len(stmt.Values)))
		}

		var expectedTI *TypeInfo
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
				lifetime, err := i.evalOne(stmt.Lifetime)
				if err != nil {
					return SignalNone{}, err
				}

				lifetime = UnwrapFully(lifetime)

				if lifetime.(IntValue).V > 0 {
					i.Env.DefineWithLifetime(name.Value, copyValue(v), lifetime.(IntValue).V+1, false) // +1 because the var statement itself also decrements it
					return SignalNone{}, nil
				}
			} else {
				i.Env.Define(name.Value, copyValue(v), true)
			}
		}

		return SignalNone{}, nil

	case *parser.AssignmentStatement:
		values := make([]Value, 0, len(stmt.Values))

		if len(stmt.Values) == 1 && len(stmt.Targets) > 1 {
			res, err := i.EvalExpression(stmt.Values[0])
			if err != nil {
				return SignalNone{}, err
			}

			values, err = i.unpackForAssign(stmt, res, len(stmt.Targets))
			if err != nil {
				return SignalNone{}, err
			}
		} else {
			values = make([]Value, len(stmt.Values))
			for idx, expr := range stmt.Values {
				res, err := i.EvalExpression(expr)
				if err != nil {
					return SignalNone{}, err
				}

				vals, err := i.unpackForAssign(stmt, res, 1)
				if err != nil {
					return SignalNone{}, err
				}
				values[idx] = vals[0]
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
			if op, ok := compoundOps[stmt.Op]; ok {
				cur, err := targets[idx].Get(i)
				if err != nil {
					return SignalNone{}, err
				}

				res, err := i.evalInfix(
					&parser.InfixExpression{
						NodeBase: stmt.NodeBase,
						Left:     stmt.Targets[idx],
						Right:    stmt.Values[idx],
						Operator: op,
					},
					cur,
					op,
					values[idx],
				)
				if err != nil {
					return SignalNone{}, err
				}

				err = targets[idx].Set(i, res)
			} else {
				err := targets[idx].Set(i, copyValue(values[idx]))
				if err != nil {
					return SignalNone{}, NewRuntimeError(stmt.Targets[idx], err.Error())
				}
			}
		}

		return SignalNone{}, nil

	case *parser.ReturnStatement:
		values := []Value{}

		for _, expr := range stmt.Values {
			v, err := i.evalOne(expr)
			if err != nil {
				return SignalNone{}, err
			}
			values = append(values, v)
		}

		return SignalReturn{Values: values}, nil

	case *parser.ExpressionStatement:
		val, err := i.evalOne(stmt.Expression)
		if err != nil {
			return SignalNone{}, err
		}

		return SignalValue{Value: val}, nil

	case *parser.IfStatement:
		if stmt.Condition == nil {
			return SignalNone{}, NewRuntimeError(s, "if statement missing condition")
		}
		if stmt.Consequence == nil {
			return SignalNone{}, NewRuntimeError(s, "if statement missing consequence")
		}
		cond, err := i.evalOne(stmt.Condition)
		if err != nil {
			return SignalNone{}, err
		}

		truthy, err := isTruthy(cond)
		if err != nil {
			return SignalNone{}, NewRuntimeError(stmt, err.Error())
		}
		if truthy {
			if stmt.Consequence != nil {
				return i.EvalBlock(stmt.Consequence, true, nil)
			}
		} else {
			if stmt.Alternative != nil {
				return i.EvalBlock(stmt.Alternative, true, nil)
			}
		}
		return SignalNone{}, nil

	case *parser.StartStatement:
		i.Wg.Add(1)

		go func(parent *Interpreter) {
			defer i.Wg.Done()

			sub := parent.Clone()

			defer func() {
				if r := recover(); r != nil {
					fmt.Println("panic in start:", r)
				}
			}()

			if stmt.Body != nil {
				sub.EvalBlock(stmt.Body, true, nil)
			} else if stmt.Expr != nil {
				sub.EvalExpression(stmt.Expr)
			}
		}(i)

		return SignalNone{}, nil

	case *parser.SelectStatement:
		var cases []cachedCase
		for _, c := range stmt.Cases {
			cases = append(cases, cachedCase{clause: c})
		}

		for {
			perm := rand.Perm(len(cases))

			for _, idx := range perm {
				cc := &cases[idx]

				if !cc.hasChan {
					switch op := cc.clause.Op.(type) {
					case *parser.SendExpression:
						chVal, err := i.evalOne(op.Channel)
						if err != nil {
							continue
						}
						cc.ch = chVal.(*Channel)
						cc.hasChan = true

					case *parser.PrefixExpression:
						chVal, err := i.evalOne(op.Right)
						if err != nil {
							continue
						}
						cc.ch = chVal.(*Channel)
						cc.hasChan = true
					}
				}

				if op, ok := cc.clause.Op.(*parser.SendExpression); ok && !cc.hasVal {
					val, err := i.evalOne(op.Value)
					if err != nil {
						continue
					}
					cc.sendVal = val
					cc.hasVal = true
				}

				switch cc.clause.Op.(type) {

				case *parser.SendExpression:
					select {
					case cc.ch.ch <- cc.sendVal:
						return i.runCase(cc.clause, NilValue{})
					default:
					}

				case *parser.ReceiveExpression:
					select {
					case v := <-cc.ch.ch:
						return i.runCase(cc.clause, v)
					default:
					}
				}
			}

			if stmt.Default != nil {
				return i.EvalBlock(stmt.Default.Body, true, nil)
			}

			runtime.Gosched()
		}

	case *parser.SwitchStatement:
		var switchVal Value
		var err error

		if stmt.Value == nil {
			switchVal = BoolValue{V: true}
		} else {
			switchVal, err = i.evalOne(stmt.Value)
			if err != nil {
				return SignalNone{}, err
			}
		}

		switchVal = UnwrapFully(switchVal)

		for _, c := range stmt.Cases {
			matched := false
			for _, expr := range c.Exprs {
				caseVal, err := i.evalOne(expr)
				if err != nil {
					return SignalNone{}, err
				}

				caseVal = UnwrapFully(caseVal)

				if valuesEqual(switchVal, caseVal) {
					matched = true
					break
				}
			}

			if !matched {
				continue
			}

			sig, err := i.EvalBlock(c.Body, true, nil)
			if err != nil {
				return SignalNone{}, err
			}

			if _, ok := sig.(SignalNone); !ok {
				return sig, nil
			}

			return SignalNone{}, nil
		}

		if stmt.Default != nil {
			sig, err := i.EvalBlock(stmt.Default.Body, true, nil)
			if err != nil {
				return SignalNone{}, err
			}
			if _, ok := sig.(SignalNone); !ok {
				return sig, nil
			}
		}

		return SignalNone{}, nil

	case *parser.WithStatement:
		val, err := i.evalOne(stmt.Expr)
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
		_, err := i.EvalStatement(stmt.Init)
		if err != nil {
			return SignalNone{}, err
		}

		for {
			i.Env = loopEnv
			cond, err := i.evalOne(stmt.Condition)
			if err != nil {
				return SignalNone{}, err
			}

			truthy, _ := isTruthy(cond)
			if !truthy {
				break
			}

			bodyEnv := NewEnvironment(loopEnv)
			i.Env = bodyEnv

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
		iterable, err := i.evalOne(stmt.Expr)
		if err != nil {
			return SignalNone{}, err
		}

		iterable = UnwrapFully(iterable)

		runIteration := func(setVars func()) (ControlSignal, error) {
			oldEnv := i.Env
			env := NewEnvironment(oldEnv)
			i.Env = env

			setVars()
			sig, err := i.EvalBlock(stmt.Body, false, nil)

			i.Env = oldEnv
			return sig, err
		}

		switch v := iterable.(type) {
		case ArrayValue:
			for idx, elem := range v.Elements {
				sig, err := runIteration(func() {
					if stmt.Key != nil && stmt.Key.Value != "_" {
						i.Env.Define(stmt.Key.Value, IntValue{V: idx}, false)
					}

					if stmt.Value != nil && stmt.Value.Value != "_" {
						i.Env.Define(stmt.Value.Value, copyValue(elem), false)
					}
				})

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
				sig, err := runIteration(func() {
					if stmt.Key != nil && stmt.Key.Value != "_" {
						i.Env.Define(stmt.Key.Value, copyValue(v.Keys[k]), false)
					}

					if stmt.Value != nil && stmt.Value.Value != "_" {
						i.Env.Define(stmt.Value.Value, copyValue(val), false)
					}
				})

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
				sig, err := runIteration(func() {
					if stmt.Key != nil && stmt.Key.Value != "_" {
						i.Env.Define(stmt.Key.Value, IntValue{V: idx}, false)
					}

					if stmt.Value != nil && stmt.Value.Value != "_" {
						i.Env.Define(stmt.Value.Value, StringValue{V: string(s)}, false)
					}
				})

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

				if stmt.Value != nil {
					return SignalNone{}, NewRuntimeError(stmt, "integer range expects 1 variable")
				}

				sig, err := i.EvalBlock(stmt.Body, false, nil)

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
			return SignalNone{}, NewRuntimeError(stmt, fmt.Sprintf("range expects (slice|array|map|int|string), but got %s", UnwrapAlias(i.TypeInfoFromValue(iterable)).Name))
		}

		return SignalNone{}, nil

	case *parser.WhileStatement:
		for {
			cond, err := i.evalOne(stmt.Condition)
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

			oldEnv := i.Env
			i.Env = NewEnvironment(oldEnv)

			sig, err := i.EvalBlock(stmt.Body, false, nil)

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

		return SignalNone{}, nil

	case *parser.DeferStatement:
		i.Env.AddDefer(stmt)
		return SignalNone{}, nil

	case *parser.BreakStatement:
		return SignalBreak{}, nil

	case *parser.ContinueStatement:
		return SignalContinue{}, nil

	}

	return SignalNone{}, nil
}

func (i *Interpreter) EvalExpression(e parser.Expression) (EvalResult, error) {
	if e == nil {
		return EvalResult{}, nil
	}

	switch expr := e.(type) {
	case *parser.IntLiteral:
		return EvalResult{[]Value{UntypedValue{IntValue{V: expr.Value}}}, nil}, nil

	case *parser.FloatLiteral:
		return EvalResult{[]Value{UntypedValue{FloatValue{V: expr.Value}}}, nil}, nil

	case *parser.StringLiteral:
		return EvalResult{[]Value{UntypedValue{StringValue{V: expr.Value}}}, nil}, nil

	case *parser.BoolLiteral:
		return EvalResult{[]Value{UntypedValue{BoolValue{V: expr.Value}}}, nil}, nil

	case *parser.NilLiteral:
		return EvalResult{[]Value{NilValue{}}, nil}, nil

	case parser.TypeNode:
		ti, err := i.resolveTypeNode(expr)
		if err != nil {
			return EvalResult{[]Value{NilValue{}}, nil}, err
		}

		return EvalResult{[]Value{TypeValue{
			TypeInfo: ti,
		}}, nil}, nil

	case *parser.MemberExpression:
		left, err := i.evalOne(expr.Left)
		if err != nil {
			return EvalResult{[]Value{NilValue{}}, nil}, err
		}

		val, err := i.evalMemberExpression(expr, left, expr.Field.Value)

		return EvalResult{[]Value{val}, nil}, nil

	case *parser.Identifier:
		if expr.Value == "_" {
			return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(expr, "cannot use '_' as a value")
		}

		if v, ok := i.TypeEnv[expr.Value]; ok {
			return EvalResult{[]Value{v}, nil}, nil
		}

		v, ok, _ := i.Env.Get(expr.Value)
		if !ok {
			return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(expr, fmt.Sprintf("undefined variable: %s", expr.Value))
		}

		return EvalResult{[]Value{v}, nil}, nil

	case *parser.CompositeLiteral:
		ti, err := i.resolveTypeNode(expr.Type)
		if err != nil {
			return EvalResult{[]Value{NilValue{}}, nil}, err
		}

		val, err := i.evalCompositeLiteral(expr, ti)

		return EvalResult{[]Value{val}, nil}, nil

	case *parser.FuncLiteral:
		paramTypes := make([]*TypeInfo, 0)
		paramNames := make([]string, 0)

		returnTypes := make([]*TypeInfo, 0)
		returnNames := make([]string, 0)

		err := i.checkFuncLiteral(expr)
		if err != nil {
			return EvalResult{[]Value{NilValue{}}, nil}, err
		}

		for _, typ := range expr.ReturnTypes {
			ti, err := i.resolveTypeNode(typ)
			if err != nil {
				return EvalResult{[]Value{NilValue{}}, nil}, err
			}

			ti = UnwrapAlias(ti)

			returnTypes = append(returnTypes, ti)
			paramNames = append(paramNames, ti.Name)
		}

		for _, param := range expr.Params {
			ti, err := i.resolveTypeNode(param.Type)
			if err != nil {
				return EvalResult{[]Value{NilValue{}}, nil}, err
			}

			ti = UnwrapAlias(ti)

			paramTypes = append(paramTypes, ti)
			returnNames = append(returnNames, ti.Name)
		}

		typeInfo := &TypeInfo{
			Name:    fmt.Sprintf("fun(%s) (%s)", strings.Join(paramNames, ", "), strings.Join(returnNames, ", ")),
			Kind:    TypeFunc,
			Returns: returnTypes,
			Params:  paramTypes,
		}

		return EvalResult{[]Value{&Func{
			Params:   expr.Params,
			Body:     expr.Body,
			TypeName: typeInfo,
			Env:      i.Env,
		}}, nil}, nil

	case *parser.FuncCall:
		val, err := i.evalCall(expr)

		return EvalResult{[]Value{val}, nil}, err

	case *parser.IndexExpression:
		left, err := i.evalOne(expr.Left)
		if err != nil {
			return EvalResult{[]Value{NilValue{}}, nil}, err
		}

		index, err := i.evalOne(expr.Index)
		if err != nil {
			return EvalResult{[]Value{NilValue{}}, nil}, err
		}

		val, err := i.evalIndexExpression(expr, left, index)
		if err != nil {
			return EvalResult{[]Value{NilValue{}}, nil}, err
		}

		return EvalResult{val.Values, nil}, nil

	case *parser.SliceExpression:
		left, err := i.evalOne(expr.Left)
		if err != nil {
			return EvalResult{[]Value{NilValue{}}, nil}, err
		}

		start, err := i.evalOne(expr.Start)
		if err != nil {
			return EvalResult{[]Value{NilValue{}}, nil}, err
		}

		end, err := i.evalOne(expr.End)
		if err != nil {
			return EvalResult{[]Value{NilValue{}}, nil}, err
		}

		val, err := i.evalSliceExpression(expr, left, start, end)
		if err != nil {
			return EvalResult{[]Value{NilValue{}}, nil}, err
		}

		return EvalResult{[]Value{val}, nil}, nil

	case *parser.TypeAssertExpression:
		val, err := i.evalOne(expr.Expr)
		if err != nil {
			return EvalResult{[]Value{NilValue{}}, nil}, err
		}

		staticTI := UnwrapAlias(i.TypeInfoFromValue(val))
		if staticTI.Kind != TypeInterface {
			return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(expr,
				"type assertion only allowed on interface values")
		}

		targetTI, err := i.resolveTypeNode(expr.Type)
		if err != nil {
			return EvalResult{[]Value{NilValue{}}, nil}, err
		}

		inner := UnwrapFully(val)
		actualTI := UnwrapAlias(i.TypeInfoFromValue(inner))

		if !TypesAssignable(actualTI, targetTI) {
			if expr.ExpectOk {
				return EvalResult{[]Value{NilValue{}, BoolValue{false}}, nil}, nil
			}

			return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(expr,
				fmt.Sprintf("interface conversion: '%s' is not '%s'",
					actualTI.Name, targetTI.Name))
		}

		if expr.ExpectOk {
			return EvalResult{[]Value{i.promoteValueToType(inner, targetTI), BoolValue{true}}, nil}, nil
		}

		return EvalResult{[]Value{i.promoteValueToType(inner, targetTI)}, nil}, nil

	case *parser.SendExpression:
		chVal, err := i.evalOne(expr.Channel)
		if err != nil {
			return EvalResult{[]Value{NilValue{}}, nil}, err
		}

		val, err := i.evalOne(expr.Value)
		if err != nil {
			return EvalResult{[]Value{NilValue{}}, nil}, err
		}

		channel, ok := chVal.(*Channel)
		if !ok {
			return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(expr, "not a channel")
		}

		if channel.ch == nil {
			return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(expr, "cannot send on nil channel")
		}

		if !channel.canSend {
			return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(expr, "cannot send on receive-only channel")
		}

		if channel.closed {
			return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(expr, "cannot send on closed channel")
		}

		if !TypesAssignable(i.TypeInfoFromValue(val), channel.ElemType) {
			return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(expr, fmt.Sprintf("channel expected '%s', was sent '%s'", i.TypeInfoFromValue(val).Name, channel.ElemType.Name))
		}

		channel.ch <- val

		return EvalResult{[]Value{NilValue{}}, nil}, nil

	case *parser.ReceiveExpression:
		chVal, err := i.evalOne(expr.Channel)
		if err != nil {
			return EvalResult{[]Value{NilValue{}}, nil}, err
		}

		channel, ok := chVal.(*Channel)
		if !ok {
			return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(expr, "not a channel")
		}
		if channel.ch == nil {
			return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(expr, "cannot receive from nil channel")
		}
		if !channel.canRecv {
			return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(expr, "cannot receive from a send-only channel")
		}

		if channel.closed {
			zero, err := i.defaultValueFromTypeInfo(expr, channel.ElemType)
			if err != nil {
				return EvalResult{[]Value{NilValue{}}, nil}, err
			}

			if expr.ExpectOk {
				return EvalResult{[]Value{zero, BoolValue{false}}, nil}, nil
			}

			return EvalResult{[]Value{zero}, nil}, nil
		}

		val := <-channel.ch

		if expr.ExpectOk {
			return EvalResult{[]Value{val, BoolValue{true}}, nil}, nil
		}

		return EvalResult{[]Value{val}, nil}, nil

	case *parser.InfixExpression:
		if expr.Operator == "&&" {
			left, err := i.evalOne(expr.Left)
			if err != nil {
				return EvalResult{[]Value{NilValue{}}, nil}, err
			}

			lTruthy, err := isTruthy(left)
			if err != nil {
				return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(expr, err.Error())
			}

			if !lTruthy {
				return EvalResult{[]Value{BoolValue{false}}, nil}, nil
			}

			right, err := i.evalOne(expr.Right)
			if err != nil {
				return EvalResult{[]Value{NilValue{}}, nil}, err
			}

			rTruthy, err := isTruthy(right)
			if err != nil {
				return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(expr, err.Error())
			}

			return EvalResult{[]Value{BoolValue{rTruthy}}, nil}, nil
		}

		if expr.Operator == "||" {
			left, err := i.evalOne(expr.Left)
			if err != nil {
				return EvalResult{[]Value{NilValue{}}, nil}, err
			}

			lTruthy, err := isTruthy(left)
			if err != nil {
				return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(expr, err.Error())
			}

			if lTruthy {
				return EvalResult{[]Value{BoolValue{true}}, nil}, nil
			}

			right, err := i.evalOne(expr.Right)
			if err != nil {
				return EvalResult{[]Value{NilValue{}}, nil}, err
			}

			rTruthy, err := isTruthy(right)
			if err != nil {
				return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(expr, err.Error())
			}

			return EvalResult{[]Value{BoolValue{rTruthy}}, nil}, nil
		}

		left, err := i.evalOne(expr.Left)
		if err != nil {
			return EvalResult{[]Value{NilValue{}}, nil}, err
		}

		right, err := i.evalOne(expr.Right)
		if err != nil {
			return EvalResult{[]Value{NilValue{}}, nil}, err
		}

		val, err := i.evalInfix(expr, left, expr.Operator, right)

		return EvalResult{[]Value{val}, nil}, err

	case *parser.PrefixExpression:
		right, err := i.evalOne(expr.Right)
		if err != nil {
			return EvalResult{[]Value{NilValue{}}, nil}, err
		}

		val, err := i.evalPrefix(expr, expr.Operator, right)

		return EvalResult{[]Value{val}, nil}, nil

	case *parser.PostfixExpression:
		left, err := i.evalOne(expr.Left)
		if err != nil {
			return EvalResult{[]Value{NilValue{}}, nil}, err
		}

		val, err := i.evalPostfix(expr, left, expr.Operator)

		return EvalResult{[]Value{val}, nil}, err

	case *parser.GroupedExpression:
		return i.EvalExpression(expr.Expression)

	case *parser.InterpolatedString:
		is := expr
		var out strings.Builder

		for _, part := range is.Parts {
			val, err := i.evalOne(part)
			if err != nil {
				return EvalResult{[]Value{NilValue{}}, nil}, err
			}
			out.WriteString(val.String())
		}

		return EvalResult{[]Value{StringValue{out.String()}}, nil}, nil

	default:
		return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(expr, fmt.Sprintf("unhandled expression type: %T", e))
	}
}

func (i *Interpreter) evalOne(expr parser.Expression) (Value, error) {
	res, err := i.EvalExpression(expr)
	if err != nil {
		return NilValue{}, err
	}
	return res.MustSingle(expr)
}

func (i *Interpreter) evalCompositeLiteral(expr *parser.CompositeLiteral, ti *TypeInfo) (Value, error) {
	ti = UnwrapAlias(ti)

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

	if typeInfo.Opaque && len(expr.Fields) > 0 {
		return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("type '%s' is opaque and cannot be constructed with fields", typeInfo.Name))
	}

	fields := make(map[string]Value)

	for name, e := range expr.Fields {
		expectedType, ok := structType.Fields[name]
		if !ok {
			return NilValue{}, NewRuntimeError(
				expr,
				fmt.Sprintf("unknown field '%s' in struct '%s'",
					name, typeInfo.Name),
			)
		}

		v, err := i.evalOne(e)
		if err != nil {
			return NilValue{}, err
		}

		actualTI := UnwrapAlias(i.TypeInfoFromValue(v))
		expectedTI := UnwrapAlias(expectedType)

		v = i.promoteValueToType(v, expectedTI)

		actualTI = UnwrapAlias(i.TypeInfoFromValue(v))

		if !TypesAssignable(actualTI, expectedTI) {
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
		val, err := i.evalOne(el)
		if err != nil {
			return NilValue{}, err
		}

		valType := UnwrapAlias(i.TypeInfoFromValue(val))

		if !TypesAssignable(valType, elemType) {
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
			Entries:   map[string]Value{},
			Keys:      map[string]Value{},
			KeyType:   expected.Key,
			ValueType: expected.Value,
		}, nil
	}

	k0, err := i.evalOne(expr.Pairs[0].Key)
	if err != nil {
		return NilValue{}, err
	}

	v0, err := i.evalOne(expr.Pairs[0].Value)
	if err != nil {
		return NilValue{}, err
	}

	keyTI := UnwrapAlias(i.TypeInfoFromValue(k0))
	valTI := UnwrapAlias(i.TypeInfoFromValue(v0))

	if expected != nil {
		if !isComparableValue(k0) {
			return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("map key type %s is not comparable", keyTI.Name))
		}

		if !TypesAssignable(keyTI, expected.Key) {
			return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("type mismatch: map key 0 expected %s but got %s", expected.Key.Name, keyTI.Name))
		}
		keyTI = expected.Key

		if !TypesAssignable(valTI, expected.Value) {
			return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("type mismatch: map value 0 expected %s but got %s", expected.Value.Name, valTI.Name))
		}
		valTI = expected.Value
	}

	elems := map[string]Value{}
	keys := map[string]Value{}

	for idx, e := range expr.Pairs {
		k, err := i.evalOne(e.Key)
		if err != nil {
			return NilValue{}, err
		}

		v, err := i.evalOne(e.Value)
		if err != nil {
			return NilValue{}, err
		}

		kt := UnwrapAlias(i.TypeInfoFromValue(k))
		vt := UnwrapAlias(i.TypeInfoFromValue(v))

		if keyTI.Kind == TypeInterface && !isComparableValue(k) {
			return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("map key %d is not comparable", idx))
		}

		if !TypesAssignable(kt, keyTI) {
			return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("map key %d expected %s but got %s", idx, keyTI.Name, kt.Name))
		}

		if !TypesAssignable(vt, valTI) {
			return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("map value %d expected %s but got %s", idx, valTI.Name, vt.Name))
		}

		if err := validateRange(expr, k, keyTI); err != nil {
			return NilValue{}, err
		}

		if err := validateRange(expr, v, valTI); err != nil {
			return NilValue{}, err
		}

		elems[MapKey(k)] = v
		keys[MapKey(k)] = k
	}

	return MapValue{
		Entries:   elems,
		Keys:      keys,
		KeyType:   keyTI,
		ValueType: valTI,
	}, nil
}

type cachedCase struct {
	clause  *parser.SelectCaseClause
	ch      *Channel
	sendVal Value
	hasVal  bool
	hasChan bool
}

func (i *Interpreter) runCase(c *parser.SelectCaseClause, recvVal Value) (ControlSignal, error) {
	switch c.Op.(type) {

	case *parser.SendExpression:
		return i.EvalBlock(c.Body, true, nil)

	case *parser.ReceiveExpression:
		var vars map[string]Value
		if c.AssignName != nil {
			vars = map[string]Value{
				c.AssignName.Value: recvVal,
			}
		}
		return i.EvalBlock(c.Body, true, vars)
	}

	return SignalNone{}, nil
}

func (i *Interpreter) evalCall(e *parser.FuncCall) (Value, error) {
	if ident, ok := e.Callee.(*parser.Identifier); ok {
		if ti, ok := i.TypeEnv[ident.Value]; ok {
			if len(e.Args) != 1 {
				return NilValue{}, NewRuntimeError(e, "type cast expects 1 arg")
			}
			return i.evalTypeCast(ti.TypeInfo, e.Args[0], e)
		}
	}

	return i.evalFuncCall(e)
}

func (i *Interpreter) evalTypeCast(target *TypeInfo, arg parser.Expression, node parser.Node) (Value, error) {
	val, err := i.evalOne(arg)
	if err != nil {
		return NilValue{}, err
	}

	v := UnwrapFully(val)

	if ev, ok := v.(EnumValue); ok {
		inner := ev.Variant.Value
		return i.evalTypeCastValue(target, inner, node)
	}

	switch target.Kind {
	case TypeInt:
		var val int

		switch v := v.(type) {
		case IntValue:
			val = v.V
		case FloatValue:
			val = int(v.V)
		default:
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("cannot cast '%s' to '%s'", i.TypeInfoFromValue(v).Name, target.Name))
		}

		return IntValue{V: val}, nil
	case TypeFloat:
		var val float64

		switch v := v.(type) {
		case IntValue:
			val = float64(v.V)
		case FloatValue:
			val = v.V
		default:
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("cannot cast '%s' to '%s'", i.TypeInfoFromValue(v).Name, target.Name))
		}

		return FloatValue{V: val}, nil
	case TypeString:
		var val string

		switch v := v.(type) {
		case StringValue:
			val = v.V
		default:
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("cannot cast '%s' to '%s'", i.TypeInfoFromValue(v).Name, target.Name))
		}

		return StringValue{V: val}, nil
	case TypeBool:
		var val bool

		switch v := v.(type) {
		case BoolValue:
			val = v.V
		default:
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("cannot cast '%s' to '%s'", i.TypeInfoFromValue(v).Name, target.Name))
		}

		return BoolValue{V: val}, nil
	case TypeArray:
		var val ArrayValue

		switch v := v.(type) {
		case ArrayValue:
			val = v
		default:
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("cannot cast '%s' to '%s'", i.TypeInfoFromValue(v).Name, target.Name))
		}

		return val, nil
	case TypeFixedArray:
		var val ArrayValue

		switch v := v.(type) {
		case ArrayValue:
			val = v

			if val.Capacity != target.Size {
				return NilValue{}, NewRuntimeError(node, fmt.Sprintf("cannot cast '%s' to '%s'", i.TypeInfoFromValue(v).Name, target.Name))
			}
		default:
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("cannot cast '%s' to '%s'", i.TypeInfoFromValue(v).Name, target.Name))
		}

		return val, nil
	case TypeMap:
		switch v := v.(type) {
		case MapValue:
			return v, nil
		default:
			return NilValue{}, NewRuntimeError(node,
				fmt.Sprintf("cannot cast '%s' to '%s'", i.TypeInfoFromValue(v).Name, target.Name))
		}
	case TypeStruct:
		switch v := v.(type) {
		case *StructValue:
			return v, nil
		default:
			return NilValue{}, NewRuntimeError(node,
				fmt.Sprintf("cannot cast '%s' to '%s'", i.TypeInfoFromValue(v).Name, target.Name))
		}
	case TypePointer:
		switch v := v.(type) {
		case *PointerValue:
			return v, nil
		default:
			return NilValue{}, NewRuntimeError(node,
				fmt.Sprintf("cannot cast '%s' to '%s'", i.TypeInfoFromValue(v).Name, target.Name))
		}
	case TypeFunc:
		switch v := v.(type) {
		case *Func:
			return v, nil
		default:
			return NilValue{}, NewRuntimeError(node,
				fmt.Sprintf("cannot cast '%s' to '%s'", i.TypeInfoFromValue(v).Name, target.Name))
		}
	case TypeChannel:
		switch v := v.(type) {
		case *Channel:
			return v, nil
		default:
			return NilValue{}, NewRuntimeError(node,
				fmt.Sprintf("cannot cast '%s' to '%s'", i.TypeInfoFromValue(v).Name, target.Name))
		}
	case TypeInterface:
		if !implementsInterface(i.TypeInfoFromValue(v), target) {
			return NilValue{}, NewRuntimeError(node,
				fmt.Sprintf("type '%s' does not implement interface '%s'",
					i.TypeInfoFromValue(v).Name, target.Name))
		}

		return InterfaceValue{
			Value:    v,
			TypeInfo: i.TypeInfoFromValue(v),
		}, nil
	case TypeNamed:
		base := target.Underlying

		casted, err := i.evalTypeCast(base, arg, node)
		if err != nil {
			return NilValue{}, err
		}

		if sv, ok := casted.(*StructValue); ok {
			sv.TypeName = target
			return sv, nil
		}

		return NamedValue{
			TypeName: target,
			Value:    casted,
		}, nil

	default:
		return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown type cast: %s", target.Name))
	}
}

func (i *Interpreter) evalTypeCastValue(target *TypeInfo, val Value, node parser.Node) (Value, error) {
	v := UnwrapFully(val)

	switch target.Kind {
	case TypeInt:
		var val int

		switch v := v.(type) {
		case IntValue:
			val = v.V
		case FloatValue:
			val = int(v.V)
		default:
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("cannot cast '%s' to '%s'", i.TypeInfoFromValue(v).Name, target.Name))
		}

		return IntValue{V: val}, nil
	case TypeFloat:
		var val float64

		switch v := v.(type) {
		case IntValue:
			val = float64(v.V)
		case FloatValue:
			val = v.V
		default:
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("cannot cast '%s' to '%s'", i.TypeInfoFromValue(v).Name, target.Name))
		}

		return FloatValue{V: val}, nil
	case TypeString:
		var val string

		switch v := v.(type) {
		case StringValue:
			val = v.V
		case EnumValue:
			ev, err := extractEnumValue(node, v, TypeInt)
			if err != nil {
				return NilValue{}, err
			}
			return ev, nil
		default:
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("cannot cast '%s' to '%s'", i.TypeInfoFromValue(v).Name, target.Name))
		}

		return StringValue{V: val}, nil
	case TypeBool:
		var val bool

		switch v := v.(type) {
		case BoolValue:
			val = v.V
		default:
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("cannot cast '%s' to '%s'", i.TypeInfoFromValue(v).Name, target.Name))
		}

		return BoolValue{V: val}, nil
	case TypeArray:
		var val ArrayValue

		switch v := v.(type) {
		case ArrayValue:
			val = v
		default:
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("cannot cast '%s' to '%s'", i.TypeInfoFromValue(v).Name, target.Name))
		}

		return val, nil
	case TypeFixedArray:
		var val ArrayValue

		switch v := v.(type) {
		case ArrayValue:
			val = v

			if val.Capacity != target.Size {
				return NilValue{}, NewRuntimeError(node, fmt.Sprintf("cannot cast '%s' to '%s'", i.TypeInfoFromValue(v).Name, target.Name))
			}
		default:
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("cannot cast '%s' to '%s'", i.TypeInfoFromValue(v).Name, target.Name))
		}

		return val, nil
	case TypeMap:
		switch v := v.(type) {
		case MapValue:
			return v, nil
		default:
			return NilValue{}, NewRuntimeError(node,
				fmt.Sprintf("cannot cast '%s' to '%s'", i.TypeInfoFromValue(v).Name, target.Name))
		}
	case TypeStruct:
		switch v := v.(type) {
		case *StructValue:
			return v, nil
		default:
			return NilValue{}, NewRuntimeError(node,
				fmt.Sprintf("cannot cast '%s' to '%s'", i.TypeInfoFromValue(v).Name, target.Name))
		}
	case TypePointer:
		switch v := v.(type) {
		case *PointerValue:
			return v, nil
		default:
			return NilValue{}, NewRuntimeError(node,
				fmt.Sprintf("cannot cast '%s' to '%s'", i.TypeInfoFromValue(v).Name, target.Name))
		}
	case TypeFunc:
		switch v := v.(type) {
		case *Func:
			return v, nil
		default:
			return NilValue{}, NewRuntimeError(node,
				fmt.Sprintf("cannot cast '%s' to '%s'", i.TypeInfoFromValue(v).Name, target.Name))
		}
	case TypeChannel:
		switch v := v.(type) {
		case *Channel:
			return v, nil
		default:
			return NilValue{}, NewRuntimeError(node,
				fmt.Sprintf("cannot cast '%s' to '%s'", i.TypeInfoFromValue(v).Name, target.Name))
		}
	case TypeInterface:
		if !implementsInterface(i.TypeInfoFromValue(v), target) {
			return NilValue{}, NewRuntimeError(node,
				fmt.Sprintf("type '%s' does not implement interface '%s'",
					i.TypeInfoFromValue(v).Name, target.Name))
		}

		return InterfaceValue{
			Value:    v,
			TypeInfo: i.TypeInfoFromValue(v),
		}, nil
	case TypeNamed:
		base := target.Underlying

		casted, err := i.evalTypeCastValue(base, v, node)
		if err != nil {
			return NilValue{}, err
		}

		if sv, ok := casted.(*StructValue); ok {
			sv.TypeName = target
			return sv, nil
		}

		return NamedValue{
			TypeName: target,
			Value:    casted,
		}, nil

	default:
		return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown type cast: %s", target.Name))
	}
}

func (i *Interpreter) evalArgs(args []parser.Expression) ([]Value, error) {
	var values []Value

	for _, arg := range args {
		if spread, ok := arg.(*parser.PostfixExpression); ok && spread.Operator == "..." {

			v, err := i.evalOne(spread.Left)
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

		v, err := i.evalOne(arg)
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
	val, err := i.evalOne(expr.Callee)
	if err != nil {
		return NilValue{}, err
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
		return NilValue{}, NewRuntimeError(expr, fmt.Sprintf("expected 'function' but got '%s'", UnwrapAlias(i.TypeInfoFromValue(val)).Name))
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

			val, err = i.paramWithType(callNode, param.Name.Value, val, expected)
			if err != nil {
				return NilValue{}, err
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

			actual := UnwrapAlias(i.TypeInfoFromValue(v))
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

	prevTypeEnv := i.TypeEnv
	if fn.TypeEnv != nil {
		i.TypeEnv = fn.TypeEnv
	}

	sig, err := i.EvalBlock(fn.Body, false, nil)

	deferErr := i.runDefers(newEnv)

	i.TypeEnv = prevTypeEnv
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
			expectedTI := UnwrapAlias(expectedType)

			if expectedTI.Name == "error" {
				if _, isNil := actual.(NilValue); isNil {
					continue
				}
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

		if len(ret.Values) == 0 {
			return NilValue{}, nil
		}
		return ret.Values[0], nil
	}

	return NilValue{}, nil
}

func (i *Interpreter) evalIndexExpression(node parser.Expression, left, idx Value) (EvalResult, error) {
	if nv, ok := left.(InterfaceValue); ok {
		return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(node, fmt.Sprintf("cannot index value of type '%s' without type assertion", nv.TypeInfo.Name))
	}

	idx = UnwrapFully(idx)
	left = UnwrapFully(left)

	typ := i.TypeInfoFromValue(left)

	switch typ.Kind {
	case TypeArray, TypeFixedArray:
		arr, ok := left.(ArrayValue)

		idxVal, ok := idx.(IntValue)
		if !ok {
			return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(node, "index must be int")
		}

		idx := idxVal.V

		if idx < 0 || idx >= len(arr.Elements) {
			return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(node, fmt.Sprintf("index: %d, out of bounds", idx))
		}

		elem := arr.Elements[idx]

		elemType := UnwrapAlias(i.TypeInfoFromValue(elem))

		if !TypesAssignable(elemType, arr.ElemType) {
			return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(node,
				fmt.Sprintf("array element expected %s but got %s",
					arr.ElemType.Name, elemType.Name))
		}

		if err := validateRange(node, elem, arr.ElemType); err != nil {
			return EvalResult{[]Value{NilValue{}}, nil}, err
		}

		return EvalResult{[]Value{copyValue(elem)}, nil}, nil

	case TypeString:
		idxVal, ok := idx.(IntValue)
		if !ok {
			return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(node, "index must be int")
		}

		idx := idxVal.V

		if idx < 0 || idx >= len(left.(StringValue).V) {
			return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(node, fmt.Sprintf("index: %d, out of bounds", idx))
		}

		r := []rune(left.(StringValue).V)
		return EvalResult{[]Value{StringValue{V: string(r[idx])}}, nil}, nil

	case TypeMap:
		mv := left.(MapValue)

		keyType := UnwrapAlias(i.TypeInfoFromValue(idx))

		if mv.KeyType.Kind == TypeInterface {
			if !isComparableValue(idx) {
				return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(
					node,
					"value of this type cannot be used as map key",
				)
			}
		} else {
			if !TypesAssignable(keyType, mv.KeyType) {
				return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(
					node,
					fmt.Sprintf(
						"map index expected %s but got %s",
						mv.KeyType.Name,
						keyType.Name,
					),
				)
			}

			if err := validateRange(node, idx, mv.KeyType); err != nil {
				return EvalResult{[]Value{NilValue{}}, nil}, err
			}
		}

		val, ok := mv.Entries[MapKey(idx)]
		if !ok {
			return EvalResult{[]Value{NilValue{}, BoolValue{false}}, nil}, nil
		}

		valType := UnwrapAlias(i.TypeInfoFromValue(val))

		if !TypesAssignable(valType, mv.ValueType) {
			return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(node,
				fmt.Sprintf("map value expected %s but got %s",
					mv.ValueType.Name, valType.Name))
		}

		if err := validateRange(node, val, mv.ValueType); err != nil {
			return EvalResult{[]Value{NilValue{}}, nil}, err
		}

		return EvalResult{[]Value{val, BoolValue{true}}, nil}, nil

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
		}

		typeStr, ok := types[typ.Kind]

		if ok {
			return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(node, fmt.Sprintf("indexing is not allowed with type: '%s'", typeStr))
		}

		return EvalResult{[]Value{NilValue{}}, nil}, NewRuntimeError(node, fmt.Sprintf("indexing is not allowed with type: %d", typ.Kind))
	}
}

func (i *Interpreter) evalSliceExpression(node parser.Expression, left, startVal, endVal Value) (Value, error) {
	if iv, ok := left.(InterfaceValue); ok {
		return NilValue{}, NewRuntimeError(node,
			fmt.Sprintf("cannot slice value of type '%s' without type assertion", iv.TypeInfo.Name))
	}

	left = UnwrapFully(left)
	typ := i.TypeInfoFromValue(left)

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

	startVal = UnwrapFully(startVal)
	endVal = UnwrapFully(endVal)

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
	if left == nil {
		return NilValue{}, NewRuntimeError(node, "nil value in member expression")
	}

	if iv, ok := left.(InterfaceValue); ok {
		if iv.Value == nil {
			return NilValue{}, NewRuntimeError(node, "nil interface value")
		}
		return i.evalMemberExpression(node, iv.Value, field)
	}

	if nv, ok := left.(NamedValue); ok {
		return i.evalMemberExpression(node, nv.Value, field)
	}

	orig := left

	origType := UnwrapAlias(i.TypeInfoFromValue(orig))
	recvType := UnwrapAlias(i.TypeInfoFromValue(left))
	ptrType := i.pointerTo(recvType)

	if fn, ok := i.Env.GetMethod(origType, field); ok {
		return BoundMethodValue{
			Receiver: orig,
			Func:     fn,
		}, nil
	}

	if fn, ok := i.Env.GetMethod(ptrType, field); ok {
		if ptr, ok := orig.(*PointerValue); ok {
			return BoundMethodValue{
				Receiver: ptr,
				Func:     fn,
			}, nil
		}

		tmp := &Variable{Value: orig}
		return BoundMethodValue{
			Receiver: &PointerValue{
				Target:   VariableTarget{Var: tmp},
				ElemType: recvType,
			},
			Func: fn,
		}, nil
	}

	if ptr, ok := left.(*PointerValue); ok {
		val, err := ptr.Target.Get(i)
		if err != nil {
			return NilValue{}, err
		}

		if _, ok := val.(NilValue); ok || ptr.Target == nil {
			return NilValue{}, NewRuntimeError(node, "nil pointer dereference")
		}
		left = val
	}

	switch obj := left.(type) {

	case ModuleValue:
		if typ, ok := obj.TypeEnv[field]; ok {
			return typ, nil
		}
		val, ok, _ := obj.Env.Get(field)
		if !ok {
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown '%s'", field))
		}
		return val, nil

	case *StructValue:
		structTI := obj.TypeName
		if structTI.Kind == TypeNamed {
			structTI = structTI.Underlying
		}
		if structTI.Opaque {
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("type '%s' is opaque and its fields cannot be accessed", structTI.Name))
		}
		val, ok := obj.Fields[field]
		if !ok {
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown field %s", field))
		}
		expectedType, ok := structTI.Fields[field]
		if !ok {
			return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown field %s", field))
		}
		// skip type check if type info is missing
		actualTI := UnwrapAlias(i.TypeInfoFromValue(val))
		expectedTI := UnwrapAlias(expectedType)
		if actualTI != nil && expectedTI != nil {
			if !TypesAssignable(actualTI, expectedTI) {
				return NilValue{}, NewRuntimeError(node,
					fmt.Sprintf("field '%s' expected '%s' but got '%s'",
						field, expectedType.Name, actualTI.Name))
			}
		}
		return val, nil

	case TypeValue:
		if obj.TypeInfo.Kind != TypeEnum {
			return NilValue{}, NewRuntimeError(node,
				fmt.Sprintf("type '%s' has no members", obj.TypeInfo.Name))
		}

		variant, ok := obj.TypeInfo.Variants[field]
		if !ok {
			return NilValue{}, NewRuntimeError(node,
				fmt.Sprintf("unknown enum variant '%s.%s'",
					obj.TypeInfo.Name, field))
		}

		return EnumValue{
			Enum:    obj.TypeInfo,
			Variant: variant,
		}, nil
	}

	return NilValue{}, NewRuntimeError(node,
		fmt.Sprintf("member expression expects enums or structs, but got '%s'",
			i.TypeInfoFromValue(left).Name))
}

func (i *Interpreter) evalInfix(node *parser.InfixExpression, left Value, op string, right Value) (Value, error) {
	left = UnwrapUntyped(left)
	right = UnwrapUntyped(right)

	liv, lok := left.(InterfaceValue)
	riv, rok := right.(InterfaceValue)

	if lok {
		if _, ok := right.(NilValue); ok {
			return evalInterfaceNilInfix(node, liv, op)
		}
	}

	if rok {
		if _, ok := left.(NilValue); ok {
			return evalInterfaceNilInfix(node, riv, op)
		}
	}

	if lok && rok {

		if liv.Value == nil && riv.Value == nil {
			switch op {
			case "==":
				return BoolValue{V: true}, nil
			case "!=":
				return BoolValue{V: false}, nil
			}
		}

		if liv.Value == nil || riv.Value == nil {
			switch op {
			case "==":
				return BoolValue{V: false}, nil
			case "!=":
				return BoolValue{V: true}, nil
			}
		}

		return i.evalInfix(node, liv.Value, op, riv.Value)
	}

	if lok {
		if liv.Value == nil {
			switch op {
			case "==":
				return BoolValue{V: false}, nil
			case "!=":
				return BoolValue{V: true}, nil
			}
		}

		return NilValue{}, NewRuntimeError(
			node,
			fmt.Sprintf("cannot use '%s' in operations, assert a type first",
				liv.TypeInfo.Name),
		)
	}

	if rok {
		if riv.Value == nil {
			switch op {
			case "==":
				return BoolValue{V: false}, nil
			case "!=":
				return BoolValue{V: true}, nil
			}
		}

		return NilValue{}, NewRuntimeError(
			node,
			fmt.Sprintf("cannot use '%s' in operations, assert a type first",
				riv.TypeInfo.Name),
		)
	}

	if _, ok := left.(NilValue); ok {
		return evalNilInfix(node, op, right)
	}

	if _, ok := right.(NilValue); ok {
		return evalNilInfix(node, op, left)
	}

	lnv, lok := left.(NamedValue)
	rnv, rok := right.(NamedValue)

	if lok || rok {

		if !lok || !rok || lnv.TypeName != rnv.TypeName {
			return NilValue{}, NewRuntimeError(
				node,
				"cannot operate on mismatched named types (try casting)",
			)
		}

		ul := UnwrapFully(left)
		ur := UnwrapFully(right)

		res, err := i.evalInfix(node, ul, op, ur)
		if err != nil {
			return NilValue{}, err
		}

		return NamedValue{
			TypeName: lnv.TypeName,
			Value:    res,
		}, nil
	}

	if left.Type() == INT && right.Type() == FLOAT {
		return evalFloatInfix(node,
			FloatValue{V: float64(left.(IntValue).V)},
			op,
			right.(FloatValue))
	}

	if left.Type() == FLOAT && right.Type() == INT {
		return evalFloatInfix(node,
			left.(FloatValue),
			op,
			FloatValue{V: float64(right.(IntValue).V)})
	}

	if left.Type() == POINTER && right.Type() == NIL {
		return evalNilInfix(node, op, left.(*PointerValue))
	}

	if left.Type() == NIL && right.Type() == POINTER {
		return evalNilInfix(node, op, right.(*PointerValue))
	}

	if left.Type() != right.Type() {
		return NilValue{}, NewRuntimeError(
			node,
			fmt.Sprintf("type mismatch: '%s' %s '%s'",
				left.Type(), op, right.Type()),
		)
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
		return evalEnumInfix(node, left.(EnumValue), op, right.(EnumValue))

	case POINTER:
		return evalPointerInfix(node, left.(*PointerValue), op, right.(*PointerValue))

	case STRUCT:
		return evalStructInfix(node, left.(*StructValue), op, right.(*StructValue))

	case ARR:
		return evalArrayInfix(node, left.(ArrayValue), op, right.(ArrayValue))
	}

	return NilValue{}, NewRuntimeError(
		node,
		fmt.Sprintf("unsupported operand types: %s %s %s",
			i.TypeInfoFromValue(left).Name,
			op,
			i.TypeInfoFromValue(right).Name),
	)
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

		return IntValue{V: left.V / right.V}, nil

	case "%":
		if right.V == 0 {
			return NilValue{}, NewRuntimeError(node, "undefined: mod by zero")
		}

		return IntValue{V: left.V % right.V}, nil
	case "|":
		return IntValue{V: left.V | right.V}, nil
	case "&":
		return IntValue{V: left.V & right.V}, nil
	case ">>":
		return IntValue{V: left.V >> right.V}, nil
	case "<<":
		return IntValue{V: left.V << right.V}, nil
	case "^":
		return IntValue{V: left.V ^ right.V}, nil
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
	case "<=":
		return BoolValue{V: left.V <= right.V}, nil
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

func evalInterfaceNilInfix(node *parser.InfixExpression, left InterfaceValue, op string) (Value, error) {
	isNil := left.Value == NilValue{} || left.Value.Type() == NIL

	switch op {
	case "==":
		return BoolValue{V: isNil}, nil
	case "!=":
		return BoolValue{V: !isNil}, nil
	}

	return NilValue{}, NewRuntimeError(node, fmt.Sprintf("invalid operator: interface %s nil", op))
}

func evalEnumInfix(node *parser.InfixExpression, left EnumValue, op string, right EnumValue) (Value, error) {
	if left.Enum != right.Enum {
		return NilValue{}, NewRuntimeError(
			node,
			fmt.Sprintf("cannot compare different enums: %s and %s", left.Enum.Name, right.Enum.Name),
		)
	}

	lv := left.Variant.Value
	rv := right.Variant.Value

	switch op {
	case "==":
		return BoolValue{V: valuesEqual(lv, rv)}, nil
	case "!=":
		return BoolValue{V: !valuesEqual(lv, rv)}, nil
	case "<", ">", "<=", ">=":
		return compareOrdered(node, lv, rv, op)
	default:
		return NilValue{}, NewRuntimeError(
			node,
			fmt.Sprintf("invalid operator: %s %s %s", left.Enum.Name, op, right.Enum.Name),
		)
	}
}

func evalPointerInfix(node *parser.InfixExpression, left Value, op string, right Value) (Value, error) {
	switch op {
	case "==":
		return BoolValue{V: left.(*PointerValue).Target == right.(*PointerValue).Target}, nil
	case "!=":
		return BoolValue{V: left.(*PointerValue).Target != right.(*PointerValue).Target}, nil
	default:
		return NilValue{}, NewRuntimeError(node, fmt.Sprintf("invalid operator: %s %s %s", left.String(), op, left.String()))
	}
}

func evalArrayInfix(node *parser.InfixExpression, left ArrayValue, op string, right ArrayValue) (Value, error) {
	switch op {
	case "==":
		if len(left.Elements) != len(right.Elements) {
			return BoolValue{V: false}, nil
		}

		for i := 0; i < len(left.Elements); i++ {
			if !valuesEqual(left.Elements[i], right.Elements[i]) {
				return BoolValue{V: false}, nil
			}
		}

		return BoolValue{V: true}, nil
	case "!=":
		res, err := evalArrayInfix(node, left, "==", right)
		if err != nil {
			return NilValue{}, err
		}

		return BoolValue{V: !res.(BoolValue).V}, nil
	default:
		return NilValue{}, NewRuntimeError(node, fmt.Sprintf("invalid operator: %s %s %s", left.String(), op, right.String()))
	}
}

func evalStructInfix(node *parser.InfixExpression, left *StructValue, op string, right *StructValue) (Value, error) {
	switch op {
	case "==":
		if left.TypeName != right.TypeName {
			return BoolValue{V: false}, nil
		}

		for k, lv := range left.Fields {
			rv := right.Fields[k]

			if !valuesEqual(lv, rv) {
				return BoolValue{V: false}, nil
			}
		}

		return BoolValue{V: true}, nil

	case "!=":
		res, err := evalStructInfix(node, left, "==", right)
		if err != nil {
			return NilValue{}, err
		}
		return BoolValue{V: !res.(BoolValue).V}, nil

	default:
		return NilValue{}, NewRuntimeError(
			node,
			fmt.Sprintf("invalid operator: %s %s %s", left.String(), op, right.String()),
		)
	}
}

func (i *Interpreter) evalPrefix(node *parser.PrefixExpression, op string, right Value) (Value, error) {
	right = UnwrapFully(right)

	switch op {

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
		switch expr := node.Right.(type) {

		case *parser.Identifier:
			v, ok := i.Env.GetVar(expr.Value)
			if !ok {
				return NilValue{}, NewRuntimeError(node, "undefined variable")
			}

			target := VariableTarget{Name: expr.Value, Var: v}

			val, err := target.Get(i)

			if err != nil {
				return NilValue{}, err
			}

			ti := i.TypeInfoFromValue(val)
			if ti.Kind == TypePointer {
				ti = ti.Elem
			}

			return &PointerValue{
				Target:   target,
				ElemType: ti,
			}, nil

		case *parser.MemberExpression:
			ptr, err := i.evalAddressableMember(expr)
			if err != nil {
				return NilValue{}, err
			}
			return ptr, nil

		case *parser.IndexExpression:
			ptr, err := i.evalAddressableIndex(expr)
			if err != nil {
				return NilValue{}, err
			}
			return ptr, nil

		case *parser.CompositeLiteral:
			val, err := i.evalOne(expr)
			if err != nil {
				return NilValue{}, err
			}

			tmp := &Variable{Value: val}
			target := VariableTarget{Var: tmp}

			ti := i.TypeInfoFromValue(val)
			if ti.Kind == TypePointer {
				ti = ti.Elem
			}

			return &PointerValue{
				Target:   target,
				ElemType: ti,
			}, nil

		default:
			return NilValue{}, NewRuntimeError(node, "cannot take address of expression")
		}

	case "*":
		ptr, ok := right.(*PointerValue)
		if !ok {
			return NilValue{}, NewRuntimeError(node, "cannot dereference non-pointer")
		}

		if ptr.Target == nil {
			return NilValue{}, NewRuntimeError(node, "nil pointer dereference")
		}

		return ptr.Target.Get(i)
	}

	return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown prefix operator: %s", node.Operator))
}

func (i *Interpreter) evalAddressableMember(node *parser.MemberExpression) (*PointerValue, error) {
	left, err := i.evalOne(node.Left)
	if err != nil {
		return nil, err
	}

	if ptr, ok := left.(*PointerValue); ok {
		left, err = ptr.Target.Get(i)
		if err != nil {
			return nil, err
		}
	}

	sv, ok := left.(*StructValue)
	if !ok {
		return nil, NewRuntimeError(node, "cannot take address of non-struct field")
	}

	val, ok := sv.Fields[node.Field.Value]
	if !ok {
		return nil, NewRuntimeError(node, "unknown field")
	}

	ti := i.TypeInfoFromValue(val)
	if ti.Kind == TypePointer {
		ti = ti.Elem
	}

	tmp := &Variable{Value: val}

	return &PointerValue{
		Target:   VariableTarget{Var: tmp},
		ElemType: ti,
	}, nil
}

func (i *Interpreter) evalAddressableIndex(expr *parser.IndexExpression) (*PointerValue, error) {
	target, err := i.resolveAssignableTarget(expr)
	if err != nil {
		return nil, err
	}
	val, err := target.Get(i)
	if err != nil {
		return nil, err
	}
	ti := UnwrapAlias(i.TypeInfoFromValue(val))
	return &PointerValue{
		Target:   target,
		ElemType: ti,
	}, nil
}

func (i *Interpreter) evalPostfix(node *parser.PostfixExpression, left Value, op string) (Value, error) {
	switch op {
	case "++", "--":
		target, err := i.resolveAssignableTarget(node.Left)
		if err != nil {
			return NilValue{}, NewRuntimeError(node, err.Error())
		}

		cur, err := target.Get(i)
		if err != nil {
			return NilValue{}, NewRuntimeError(node, err.Error())
		}

		one := IntValue{V: 1}

		var infixOp string
		if op == "++" {
			infixOp = "+"
		} else {
			infixOp = "-"
		}

		res, err := i.evalInfix(
			&parser.InfixExpression{
				NodeBase: node.NodeBase,
				Operator: infixOp,
			},
			cur,
			infixOp,
			one,
		)
		if err != nil {
			return NilValue{}, err
		}

		err = target.Set(i, res)
		if err != nil {
			return NilValue{}, NewRuntimeError(node, err.Error())
		}

		return cur, nil
	}

	return NilValue{}, NewRuntimeError(node, fmt.Sprintf("unknown postfix operator: %s", node.Operator))
}

func isTruthy(val Value) (bool, error) {
	val = UnwrapFully(val)
	b, ok := val.(BoolValue)
	if !ok {
		return false, fmt.Errorf("condition must be boolean")
	}
	return b.V, nil
}
