package main

import (
	"fmt"
	"log"

	"github.com/z-sk1/ayla-lang/parser"
)

type SymbolKind int

const (
	SymVar SymbolKind = iota
	SymConst
	SymFunc
	SymParam
	SymType
	SymUserType
	SymStructField
)

type Symbol struct {
	Kind   SymbolKind
	Name   string
	Ident  *parser.Identifier // where it is declared
	Type   *parser.Identifier // optional: string, int, struct Foo, etc
	Value  parser.Expression
	Parent *Symbol // optional (struct, function)
}

type Scope struct {
	Parent  *Scope
	Symbols map[string]*Symbol
}

func NewScope(parent *Scope) *Scope {
	return &Scope{
		Parent:  parent,
		Symbols: make(map[string]*Symbol),
	}
}

func (s *Scope) Define(sym *Symbol) {
	if _, exists := s.Symbols[sym.Name]; exists {
		panic(fmt.Sprintf("redeclaration of %s", sym.Name))
	}
	s.Symbols[sym.Name] = sym
}

func (s *Scope) Resolve(name string) *Symbol {
	for scope := s; scope != nil; scope = scope.Parent {
		if sym, ok := scope.Symbols[name]; ok {
			return sym
		}
	}
	return nil
}

func BuildSymbols(stmts []parser.Statement) *Scope {
	root := NewScope(nil)

	for _, t := range []string{"int", "float", "string", "bool", "arr"} {
		root.Define(&Symbol{
			Kind: SymType,
			Name: t,
		})
	}

	buildInScope(root, stmts)
	return root
}

func buildInScope(scope *Scope, stmts []parser.Statement) {
	log.Printf("ENTER scope %p\n", scope)
	defer func() {
		if r := recover(); r != nil {
			log.Printf(
				"buildInScope panic (scope=%p): %#v",
				scope, r,
			)
		}
	}()

	for _, stmt := range stmts {
		if stmt == nil {
			continue
		}

		switch s := stmt.(type) {

		case *parser.VarStatement:
			if s.Name == nil {
				panic("VarStatement.Name is nil")
			}

			log.Printf("DEFINE %s in scope %p\n", s.Name.Value, scope)
			scope.Define(&Symbol{
				Kind:  SymVar,
				Name:  s.Name.Value,
				Ident: s.Name,
				Type:  s.Type,
				Value: s.Value,
			})

		case *parser.VarStatementNoKeyword:
			if s.Name == nil {
				panic("VarStatementNoKeyword.Name is nil")
			}

			log.Printf("DEFINE %s in scope %p\n", s.Name.Value, scope)
			scope.Define(&Symbol{
				Kind:  SymVar,
				Name:  s.Name.Value,
				Ident: s.Name,
				Value: s.Value,
			})

		case *parser.ConstStatement:
			if s.Name == nil {
				panic("ConstStatement.Name is nil")
			}

			log.Printf("DEFINE %s in scope %p\n", s.Name.Value, scope)
			scope.Define(&Symbol{
				Kind:  SymConst,
				Name:  s.Name.Value,
				Ident: s.Name,
				Type:  s.Type,
				Value: s.Value,
			})

		case *parser.MultiVarStatement:
			if s.Names == nil {
				panic("MultiVarStatement.Names is nil")
			}

			for _, name := range s.Names {
				log.Printf("DEFINE %s in scope %p\n", name.Value, scope)
				scope.Define(&Symbol{
					Kind:  SymVar,
					Name:  name.Value,
					Ident: name,
					Type:  s.Type,
					Value: s.Value,
				})
			}

		case *parser.MultiVarStatementNoKeyword:
			if s.Names == nil {
				panic("MultiVarStatementNoKeyword.Names is nil")
			}

			for _, name := range s.Names {
				log.Printf("DEFINE %s in scope %p\n", name.Value, scope)
				scope.Define(&Symbol{
					Kind:  SymVar,
					Name:  name.Value,
					Ident: name,
					Value: s.Value,
				})
			}

		case *parser.MultiConstStatement:
			if s.Names == nil {
				panic("MultiConstStatement.Names is nil")
			}

			for _, name := range s.Names {
				log.Printf("DEFINE %s in scope %p\n", name.Value, scope)
				scope.Define(&Symbol{
					Kind:  SymConst,
					Name:  name.Value,
					Ident: name,
					Type:  s.Type,
					Value: s.Value,
				})
			}

		case *parser.FuncStatement:
			if s.Name == nil {
				panic("FuncStatement.Name is nil")
			}

			fnSym := &Symbol{
				Kind:  SymFunc,
				Name:  s.Name.Value,
				Ident: s.Name,
			}
			log.Printf("DEFINE %s in scope %p\n", s.Name.Value, scope)
			scope.Define(fnSym)

			// function scope
			fnScope := NewScope(scope)

			// params
			for _, p := range s.Params {
				log.Printf("DEFINE %s in scope %p\n", p.Token.Literal, scope)
				fnScope.Define(&Symbol{
					Kind:  SymParam,
					Name:  p.Name.Value,
					Ident: p.Name,
					Type:  p.Type,
				})
			}

			buildInScope(fnScope, s.Body)

		case *parser.TypeStatement:
			if s.Name == nil {
				panic("TypeStatement.Name is nil")
			}

			var typeName string

			switch t := s.Type.(type) {
			case *parser.IdentType:
				typeName = t.Name

			case *parser.StructType:
				var fieldNames string
				for _, field := range t.Fields {
					fieldNames += field.Name.Value + "\n"
				}
				typeName = fmt.Sprintf("struct {\n%s}", fieldNames)

			default:
				panic(fmt.Sprintf("unknown TypeStatement.Type: %T", s.Type))
			}

			log.Printf("DEFINE %s in scope %p\n", s.Name.Value, scope)
			scope.Define(&Symbol{
				Kind:  SymUserType,
				Name:  s.Name.Value,
				Ident: s.Name,
				Type: &parser.Identifier{
					NodeBase: s.NodeBase,
					Value:    typeName,
				},
			})

		case *parser.ForStatement:
			loopScope := NewScope(scope)

			if s.Init != nil {
				buildInScope(loopScope, []parser.Statement{s.Init})
			}
			buildInScope(loopScope, s.Body)

		case *parser.WhileStatement:
			loopScope := NewScope(scope)
			buildInScope(loopScope, s.Body)

		case *parser.IfStatement:
			buildInScope(NewScope(scope), s.Consequence)
			if s.Alternative != nil {
				buildInScope(NewScope(scope), s.Alternative)
			}

		}
	}
}
