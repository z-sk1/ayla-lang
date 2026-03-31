package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/z-sk1/ayla-lang/token"
)

type Node interface {
	Pos() (int, int)
	Format(*Formatter) string
}

type TypeNode interface {
	Node
	typeNode()
}

type Statement interface {
	Node
}

type Expression interface {
	Node
}

type Formatter struct {
	Indent int
}

func (f *Formatter) identStr() string {
	return strings.Repeat("    ", f.Indent)
}

func FormatProgram(stmts []Statement) string {
	f := &Formatter{}
	var out strings.Builder
	prevLine := 0

	for i, stmt := range stmts {
		line, _ := stmt.Pos()

		if i > 0 {
			diff := line - prevLine
			if diff < 1 {
				diff = 1
			}
			if diff > 2 {
				diff = 2
			}

			out.WriteString(strings.Repeat("\n", diff))
		}

		out.WriteString(stmt.Format(f))
		prevLine = line
	}

	return out.String()
}

func formatBlock(f *Formatter, stmts []Statement) string {
	var out strings.Builder

	out.WriteString("{\n")
	f.Indent++

	for _, s := range stmts {
		out.WriteString(f.identStr())
		out.WriteString(s.Format(f))
		out.WriteString("\n")
	}

	f.Indent--
	out.WriteString(f.identStr())
	out.WriteString("}")

	return out.String()
}

func (f *Formatter) formatExprList(exprs []Expression) string {
	parts := make([]string, 0, len(exprs))

	for _, e := range exprs {
		parts = append(parts, e.Format(f))
	}

	return strings.Join(parts, ", ")
}

type NodeBase struct {
	Token token.Token
}

func (n *NodeBase) Pos() (int, int) {
	return n.Token.Line, n.Token.Column
}

const (
	_ int = iota
	LOWEST
	LOR         // ||
	LAND        // &&
	BITOR       // |
	BITXOR      // ^
	BITAND      // &
	EQUALS      // == !=
	LESSGREATER // < >
	SHIFT       // << >>
	SUM         // + -
	PRODUCT     // * /
	PREFIX      // !x -z
	MEMBER      // p.x
	CALL        // ()
	INDEX       // []
	POSTFIX     // ...
)

var precedences = map[token.TokenType]int{
	token.LOR:  LOR,
	token.LAND: LAND,

	token.OR:  BITOR,
	token.XOR: BITXOR,
	token.AND: BITAND,

	token.EQ:  EQUALS,
	token.NEQ: EQUALS,

	token.LT:  LESSGREATER,
	token.GT:  LESSGREATER,
	token.LTE: LESSGREATER,
	token.GTE: LESSGREATER,

	token.SHL: SHIFT,
	token.SHR: SHIFT,

	token.PLUS: SUM,
	token.SUB:  SUM,

	token.MUL:   PRODUCT,
	token.SLASH: PRODUCT,
	token.MOD:   PRODUCT,

	token.DOT:      MEMBER,
	token.LPAREN:   CALL,
	token.LBRACKET: INDEX,

	token.ELLIPSIS: POSTFIX,
	token.INC:      POSTFIX,
	token.DEC:      POSTFIX,
}

type VarStatement struct {
	NodeBase
	Name     *Identifier
	Type     TypeNode // if no type defaults to nil, and then automatically chooses type
	Value    Expression
	Lifetime Expression
}

func (v *VarStatement) Format(f *Formatter) string {
	out := "egg " + v.Name.Format(f)

	if v.Lifetime != nil {
		out += "<" + v.Lifetime.Format(f) + ">"
	}

	if v.Type != nil {
		out += " " + v.Type.Format(f)
	}

	if v.Value != nil {
		out += " = " + v.Value.Format(f)
	}

	return out
}

type VarStatementBlock struct {
	NodeBase
	Decls []Statement
}

func formatVarDeclNoKeyword(stmt Statement, f *Formatter) string {
	switch v := stmt.(type) {

	case *VarStatement:
		out := v.Name.Format(f)
		if v.Type != nil {
			out += " " + v.Type.Format(f)
		}
		if v.Value != nil {
			out += " = " + v.Value.Format(f)
		}
		return out

	case *MultiVarStatement:
		var out strings.Builder

		names := make([]string, len(v.Names))
		for i, n := range v.Names {
			names[i] = n.Format(f)
		}

		out.WriteString(strings.Join(names, ", "))

		if v.Type != nil {
			out.WriteString(" ")
			out.WriteString(v.Type.Format(f))
		}

		if len(v.Values) > 0 {
			vals := make([]string, len(v.Values))
			for i, val := range v.Values {
				vals[i] = val.Format(f)
			}
			out.WriteString(" = ")
			out.WriteString(strings.Join(vals, ", "))
		}

		return out.String()

	case *ConstStatement:
		out := v.Name.Format(f)
		if v.Type != nil {
			out += " " + v.Type.Format(f)
		}
		if v.Value != nil {
			out += " = " + v.Value.Format(f)
		}
		return out

	case *MultiConstStatement:
		var out strings.Builder

		names := make([]string, len(v.Names))
		for i, n := range v.Names {
			names[i] = n.Format(f)
		}

		out.WriteString(strings.Join(names, ", "))

		if v.Type != nil {
			out.WriteString(" ")
			out.WriteString(v.Type.Format(f))
		}

		if len(v.Values) > 0 {
			vals := make([]string, len(v.Values))
			for i, val := range v.Values {
				vals[i] = val.Format(f)
			}
			out.WriteString(" = ")
			out.WriteString(strings.Join(vals, ", "))
		}

		return out.String()
	}

	return stmt.Format(f)
}

func (v *VarStatementBlock) Format(f *Formatter) string {
	var out strings.Builder

	out.WriteString("egg (\n")

	f.Indent++

	for _, d := range v.Decls {
		out.WriteString(f.identStr())
		out.WriteString(formatVarDeclNoKeyword(d, f))
		out.WriteString("\n")
	}

	f.Indent--

	out.WriteString(f.identStr())
	out.WriteString(")")

	return out.String()
}

type VarStatementNoKeyword struct {
	NodeBase
	Name     *Identifier
	Value    Expression
	Lifetime Expression
}

func (v *VarStatementNoKeyword) Format(f *Formatter) string {
	out := v.Name.Format(f)

	if v.Value != nil {
		out += " = " + v.Value.Format(f)
	}

	if v.Lifetime != nil {
		out += "<" + v.Lifetime.Format(f) + ">"
	}

	return out
}

type MultiVarStatement struct {
	NodeBase
	Names    []*Identifier
	Type     TypeNode
	Values   []Expression
	Lifetime Expression
}

func (m *MultiVarStatement) Format(f *Formatter) string {
	names := []string{}
	for _, n := range m.Names {
		names = append(names, n.Format(f))
	}

	out := "egg " + strings.Join(names, ", ")

	if m.Lifetime != nil {
		out += "<" + m.Lifetime.Format(f) + ">"
	}

	if m.Type != nil {
		out += " " + m.Type.Format(f)
	}

	if len(m.Values) > 0 {
		out += " = " + f.formatExprList(m.Values)
	}

	return out
}

type MultiVarStatementNoKeyword struct {
	NodeBase
	Names    []*Identifier
	Values   []Expression
	Lifetime Expression
}

func (m *MultiVarStatementNoKeyword) Format(f *Formatter) string {
	names := []string{}
	for _, n := range m.Names {
		names = append(names, n.Format(f))
	}

	out := strings.Join(names, ", ")

	if len(m.Values) > 0 {
		out += " = " + f.formatExprList(m.Values)
	}

	if m.Lifetime != nil {
		out += "<" + m.Lifetime.Format(f) + ">"
	}

	return out
}

type ConstStatement struct {
	NodeBase
	Name     *Identifier
	Type     TypeNode // if no type defaults to nil, and then automatically chooses type
	Value    Expression
	Lifetime Expression
}

func (v *ConstStatement) Format(f *Formatter) string {
	out := "rock " + v.Name.Format(f)

	if v.Lifetime != nil {
		out += "<" + v.Lifetime.Format(f) + ">"
	}

	if v.Type != nil {
		out += " " + v.Type.Format(f)
	}

	if v.Value != nil {
		out += " = " + v.Value.Format(f)
	}

	return out
}

type ConstStatementBlock struct {
	NodeBase
	Decls []Statement
}

func (c *ConstStatementBlock) Format(f *Formatter) string {
	var out strings.Builder
	out.WriteString("rock (\n")

	f.Indent++
	for _, d := range c.Decls {
		out.WriteString(f.identStr())
		out.WriteString(formatVarDeclNoKeyword(d, f))
		out.WriteString("\n")
	}
	f.Indent--

	out.WriteString(")")
	return out.String()
}

type MultiConstStatement struct {
	NodeBase
	Names    []*Identifier
	Type     TypeNode
	Values   []Expression
	Lifetime Expression
}

func (m *MultiConstStatement) Format(f *Formatter) string {
	names := []string{}
	for _, n := range m.Names {
		names = append(names, n.Format(f))
	}

	out := "rock " + strings.Join(names, ", ")

	if m.Lifetime != nil {
		out += "<" + m.Lifetime.Format(f) + ">"
	}

	if m.Type != nil {
		out += " " + m.Type.Format(f)
	}

	if len(m.Values) > 0 {
		out += " = " + f.formatExprList(m.Values)
	}

	return out
}

type AssignmentStatement struct {
	NodeBase
	Targets []Expression
	Op      token.TokenType
	Values  []Expression
}

func (a *AssignmentStatement) Format(f *Formatter) string {
	targets := make([]string, 0, len(a.Targets))
	for _, t := range a.Targets {
		targets = append(targets, t.Format(f))
	}

	values := make([]string, 0, len(a.Values))
	for _, v := range a.Values {
		values = append(values, v.Format(f))
	}

	return fmt.Sprintf("%s %s %s",
		strings.Join(targets, ", "),
		a.Op,
		strings.Join(values, ", "),
	)
}

type EnumStatement struct {
	NodeBase
	Name     *Identifier
	Variants []*Identifier
}

func (e *EnumStatement) Format(f *Formatter) string {
	var out strings.Builder

	out.WriteString("enum ")
	out.WriteString(e.Name.Format(f))
	out.WriteString(" {\n")

	f.Indent++
	for _, v := range e.Variants {
		out.WriteString(f.identStr())
		out.WriteString(v.Format(f))
		out.WriteString("\n")
	}
	f.Indent--

	out.WriteString(f.identStr())
	out.WriteString("}")

	return out.String()
}

type TypeStatement struct {
	NodeBase
	Name  *Identifier
	Type  TypeNode
	Alias bool
}

func (t *TypeStatement) Format(f *Formatter) string {
	if t.Alias {
		return fmt.Sprintf(
			"type %s = %s",
			t.Name.Format(f),
			t.Type.Format(f),
		)
	}

	return fmt.Sprintf(
		"type %s %s",
		t.Name.Format(f),
		t.Type.Format(f),
	)
}

type StructType struct {
	NodeBase
	Fields []*StructField
}

func (*StructType) typeNode() {}

func (s *StructType) Format(f *Formatter) string {
	var out strings.Builder

	out.WriteString("struct {\n")

	f.Indent++
	for _, field := range s.Fields {
		out.WriteString(f.identStr())
		out.WriteString(field.Name.Format(f))
		out.WriteString(" ")
		out.WriteString(field.Type.Format(f))
		out.WriteString("\n")
	}
	f.Indent--

	out.WriteString(f.identStr())
	out.WriteString("}")

	return out.String()
}

type IdentType struct {
	NodeBase
	Name *Identifier
}

func (*IdentType) typeNode() {}

func (t *IdentType) Format(f *Formatter) string {
	return t.Name.Format(f)
}

type RangeType struct {
	NodeBase
	Base TypeNode
	Min  Expression
	Max  Expression
}

func (*RangeType) typeNode() {}

func (r *RangeType) Format(f *Formatter) string {
	return fmt.Sprintf(
		"%s<%s..%s>",
		r.Base.Format(f),
		r.Min.Format(f),
		r.Max.Format(f),
	)
}

type QualifiedType struct {
	NodeBase
	Module *Identifier
	Name   *Identifier
}

func (*QualifiedType) typeNode() {}

func (q *QualifiedType) Format(f *Formatter) string {
	return fmt.Sprintf(
		"%s.%s",
		q.Module.Format(f),
		q.Name.Format(f),
	)
}

type ArrayType struct {
	NodeBase
	Elem TypeNode
	Size Expression
}

func (*ArrayType) typeNode() {}

func (a *ArrayType) Format(f *Formatter) string {
	if a.Size != nil {
		return fmt.Sprintf("[%s]%s",
			a.Size.Format(f),
			a.Elem.Format(f),
		)
	}
	return "[]" + a.Elem.Format(f)
}

type MapType struct {
	NodeBase
	Key   TypeNode
	Value TypeNode
}

func (*MapType) typeNode() {}

func (m *MapType) Format(f *Formatter) string {
	return fmt.Sprintf("map[%s]%s",
		m.Key.Format(f),
		m.Value.Format(f),
	)
}

type InterfaceType struct {
	NodeBase
	Methods []*FuncType
}

func (*InterfaceType) typeNode() {}

func (i *InterfaceType) Format(f *Formatter) string {
	var out strings.Builder

	out.WriteString("interface {\n")

	f.Indent++
	for _, m := range i.Methods {
		out.WriteString(f.identStr())
		out.WriteString(m.Format(f))
		out.WriteString("\n")
	}
	f.Indent--

	out.WriteString(f.identStr())
	out.WriteString("}")

	return out.String()
}

type FuncType struct {
	NodeBase
	Name    *Identifier
	Params  []TypeNode
	Returns []TypeNode
}

func (*FuncType) typeNode() {}

func (ft *FuncType) Format(f *Formatter) string {
	params := []string{}
	for _, p := range ft.Params {
		params = append(params, p.Format(f))
	}

	out := ""

	if ft.Name != nil {
		out += ft.Name.Format(f)
	}

	out += "(" + strings.Join(params, ", ") + ")"

	if len(ft.Returns) > 0 {
		ret := []string{}
		for _, r := range ft.Returns {
			ret = append(ret, r.Format(f))
		}
		out += " " + strings.Join(ret, ", ")
	}

	return out
}

type PointerType struct {
	NodeBase
	Base TypeNode
}

func (*PointerType) typeNode() {}

func (p *PointerType) Format(f *Formatter) string {
	return "*" + p.Base.Format(f)
}

type SpawnStatement struct {
	NodeBase
	Body []Statement
}

func (s *SpawnStatement) Format(f *Formatter) string {
	return "spawn " + formatBlock(f, s.Body)
}

type IfStatement struct {
	NodeBase
	Condition   Expression
	Consequence []Statement
	Alternative []Statement // optional else block
}

func (i *IfStatement) Format(f *Formatter) string {
	out := fmt.Sprintf(
		"ayla %s %s",
		i.Condition.Format(f),
		formatBlock(f, i.Consequence),
	)

	if len(i.Alternative) > 0 {
		out += " elen " + formatBlock(f, i.Alternative)
	}

	return out
}

type Param struct {
	NodeBase
	Type     TypeNode
	Name     *Identifier
	Variadic bool
}

type FuncStatement struct {
	NodeBase
	Name        *Identifier
	Params      []*Param
	Body        []Statement
	ReturnTypes []TypeNode
}

func (fn *FuncStatement) Format(f *Formatter) string {
	params := []string{}

	for _, p := range fn.Params {
		params = append(params,
			p.Name.Format(f)+" "+p.Type.Format(f),
		)
	}

	out := fmt.Sprintf(
		"fun %s(%s)",
		fn.Name.Format(f),
		strings.Join(params, ", "),
	)

	if len(fn.ReturnTypes) > 0 {
		types := []string{}
		for _, t := range fn.ReturnTypes {
			types = append(types, t.Format(f))
		}

		out += " " + strings.Join(types, ", ")
	}

	out += " " + formatBlock(f, fn.Body)

	return out
}

type FuncCall struct {
	NodeBase
	Callee Expression
	Args   []Expression
}

func (c *FuncCall) Format(f *Formatter) string {
	return c.Callee.Format(f) + "(" + f.formatExprList(c.Args) + ")"
}

type FuncLiteral struct {
	NodeBase
	Params      []*Param
	Body        []Statement
	ReturnTypes []TypeNode
}

func (fl *FuncLiteral) Format(f *Formatter) string {
	params := []string{}
	for _, p := range fl.Params {
		params = append(params,
			p.Name.Format(f)+" "+p.Type.Format(f),
		)
	}

	out := "fun(" + strings.Join(params, ", ") + ")"

	if len(fl.ReturnTypes) > 0 {
		ret := []string{}
		for _, t := range fl.ReturnTypes {
			ret = append(ret, t.Format(f))
		}
		out += " " + strings.Join(ret, ", ")
	}

	out += " " + formatBlock(f, fl.Body)

	return out
}

type Receiver struct {
	NodeBase
	Type TypeNode
	Name *Identifier
}

type MethodStatement struct {
	NodeBase
	Name        *Identifier
	Receiver    *Receiver
	Params      []*Param
	Body        []Statement
	ReturnTypes []TypeNode
}

func (m *MethodStatement) Format(f *Formatter) string {
	params := []string{}
	for _, p := range m.Params {
		params = append(params,
			p.Name.Format(f)+" "+p.Type.Format(f),
		)
	}

	out := fmt.Sprintf(
		"fun (%s %s) %s(%s)",
		m.Receiver.Name.Format(f),
		m.Receiver.Type.Format(f),
		m.Name.Format(f),
		strings.Join(params, ", "),
	)

	if len(m.ReturnTypes) > 0 {
		ret := []string{}
		for _, t := range m.ReturnTypes {
			ret = append(ret, t.Format(f))
		}
		out += " " + strings.Join(ret, ", ")
	}

	out += " " + formatBlock(f, m.Body)

	return out
}

type ForStatement struct {
	NodeBase
	Init      Statement  // egg i = 0;
	Condition Expression // i < 5;
	Post      Statement  // i = i + 1
	Body      []Statement
}

func (fs *ForStatement) Format(f *Formatter) string {
	init := ""
	cond := ""
	post := ""

	if fs.Init != nil {
		init = fs.Init.Format(f)
	}
	if fs.Condition != nil {
		cond = fs.Condition.Format(f)
	}
	if fs.Post != nil {
		post = fs.Post.Format(f)
	}

	return fmt.Sprintf(
		"four %s; %s; %s %s",
		init,
		cond,
		post,
		formatBlock(f, fs.Body),
	)
}

type ForRangeStatement struct {
	NodeBase
	Key   *Identifier
	Value *Identifier
	Expr  Expression
	Body  []Statement
}

func (fr *ForRangeStatement) Format(f *Formatter) string {
	key := ""
	val := ""

	if fr.Key != nil {
		key = fr.Key.Format(f)
	}
	if fr.Value != nil {
		val = ", " + fr.Value.Format(f)
	}

	return fmt.Sprintf(
		"four %s%s := range %s %s",
		key,
		val,
		fr.Expr.Format(f),
		formatBlock(f, fr.Body),
	)
}

type WhileStatement struct {
	NodeBase
	Condition Expression // i < 5
	Body      []Statement
}

func (w *WhileStatement) Format(f *Formatter) string {
	return fmt.Sprintf(
		"why %s %s",
		w.Condition.Format(f),
		formatBlock(f, w.Body),
	)
}

type SwitchStatement struct {
	NodeBase
	Value   Expression
	Cases   []*CaseClause
	Default *DefaultClause
}

func (s *SwitchStatement) Format(f *Formatter) string {
	var out strings.Builder

	out.WriteString("choose ")
	out.WriteString(s.Value.Format(f))
	out.WriteString(" {\n")

	f.Indent++

	for _, c := range s.Cases {
		out.WriteString(f.identStr())
		out.WriteString(c.Format(f))
		out.WriteString("\n")
	}

	if s.Default != nil {
		out.WriteString(f.identStr())
		out.WriteString(s.Default.Format(f))
		out.WriteString("\n")
	}

	f.Indent--

	out.WriteString(f.identStr())
	out.WriteString("}")

	return out.String()
}

type CaseClause struct {
	NodeBase
	Exprs []Expression
	Body  []Statement
}

func (c *CaseClause) Format(f *Formatter) string {
	var out strings.Builder

	out.WriteString("when ")
	out.WriteString(f.formatExprList(c.Exprs))
	out.WriteString(" {\n")

	f.Indent++
	for _, s := range c.Body {
		out.WriteString(f.identStr())
		out.WriteString(s.Format(f))
		out.WriteString("\n")
		f.Indent--
		out.WriteString(f.identStr())
		f.Indent++
		out.WriteString("}\n")
	}
	f.Indent--

	return out.String()
}

type DefaultClause struct {
	NodeBase
	Body []Statement
}

func (d *DefaultClause) Format(f *Formatter) string {
	var out strings.Builder

	out.WriteString("default:\n")

	f.Indent++
	for _, s := range d.Body {
		out.WriteString(f.identStr())
		out.WriteString(s.Format(f))
		out.WriteString("\n")
	}
	f.Indent--

	return out.String()
}

type WithStatement struct {
	NodeBase
	Expr Expression
	Body []Statement
}

func (w *WithStatement) Format(f *Formatter) string {
	return fmt.Sprintf(
		"with %s %s",
		w.Expr.Format(f),
		formatBlock(f, w.Body),
	)
}

type BreakStatement struct {
	NodeBase
}

func (b *BreakStatement) Format(f *Formatter) string {
	return "kitkat"
}

type ContinueStatement struct {
	NodeBase
}

func (c *ContinueStatement) Format(f *Formatter) string {
	return "next"
}

type ReturnStatement struct {
	NodeBase
	Values []Expression
}

func (r *ReturnStatement) Format(f *Formatter) string {
	if len(r.Values) == 0 {
		return "back"
	}
	return "back " + f.formatExprList(r.Values)
}

type ImportStatement struct {
	NodeBase
	Name string
}

func (i *ImportStatement) Format(f *Formatter) string {
	return fmt.Sprintf("import %s", i.Name)
}

type DeferStatement struct {
	NodeBase
	Call *FuncCall
}

func (d *DeferStatement) Format(f *Formatter) string {
	return "defer " + d.Call.Format(f)
}

type CompositeLiteral struct {
	NodeBase
	Type     TypeNode              // works for Foo, []int, map[string]int, etc.
	Elements []Expression          // for slice/array
	Fields   map[string]Expression // for struct
	Pairs    []MapPair             // for map
}

func (c *CompositeLiteral) Format(f *Formatter) string {
	var out strings.Builder

	out.WriteString(c.Type.Format(f))
	out.WriteString("{")

	elems := []string{}

	for _, e := range c.Elements {
		elems = append(elems, e.Format(f))
	}

	for k, v := range c.Fields {
		elems = append(elems, k+": "+v.Format(f))
	}

	for _, p := range c.Pairs {
		elems = append(elems,
			fmt.Sprintf("%s: %s",
				p.Key.Format(f),
				p.Value.Format(f),
			))
	}

	out.WriteString(strings.Join(elems, ", "))
	out.WriteString("}")

	return out.String()
}

type MapPair struct {
	Key   Expression
	Value Expression
}

type StructField struct {
	Name *Identifier
	Type TypeNode
}

type SliceExpression struct {
	NodeBase
	Left  Expression
	Start Expression
	End   Expression
}

func (s *SliceExpression) Format(f *Formatter) string {
	return fmt.Sprintf("%s[%s:%s]",
		s.Left.Format(f),
		s.Start.Format(f),
		s.End.Format(f),
	)
}

type IndexExpression struct {
	NodeBase
	Left  Expression
	Index Expression
}

func (i *IndexExpression) Format(f *Formatter) string {
	return fmt.Sprintf("%s[%s]", i.Left.Format(f), i.Index.Format(f))
}

type TypeAssertExpression struct {
	NodeBase
	Expr Expression
	Type TypeNode
}

func (t *TypeAssertExpression) Format(f *Formatter) string {
	return fmt.Sprintf("%s.(%s)",
		t.Expr.Format(f),
		t.Type.Format(f),
	)
}

type IntLiteral struct {
	NodeBase
	Value int
}

func (i IntLiteral) Format(f *Formatter) string {
	return strconv.Itoa(i.Value)
}

type FloatLiteral struct {
	NodeBase
	Value float64
}

func (fl FloatLiteral) Format(f *Formatter) string {
	return strconv.FormatFloat(fl.Value, 'f', -1, 64)
}

type StringLiteral struct {
	NodeBase
	Value string
}

func (s StringLiteral) Format(f *Formatter) string {
	return fmt.Sprintf(`"%s"`, s.Value)
}

type InterpolatedString struct {
	NodeBase
	Parts []Expression
}

func (i *InterpolatedString) Format(f *Formatter) string {
	var out strings.Builder

	out.WriteString(`"`)

	for _, p := range i.Parts {
		out.WriteString(p.Format(f))
	}

	out.WriteString(`"`)

	return out.String()
}

type BoolLiteral struct {
	NodeBase
	Value bool
}

func (b BoolLiteral) Format(f *Formatter) string {
	if b.Value {
		return "yes"
	}
	return "no"
}

type NilLiteral struct {
	NodeBase
}

func (n NilLiteral) Format(f *Formatter) string {
	return "nil"
}

type MemberExpression struct {
	NodeBase
	Left  Expression  // p
	Field *Identifier // x
}

func (m *MemberExpression) Format(f *Formatter) string {
	return fmt.Sprintf("%s.%s", m.Left.Format(f), m.Field.Format(f))
}

type Identifier struct {
	NodeBase
	Value string
}

func (i *Identifier) Format(f *Formatter) string {
	return i.Value
}

type ExpressionStatement struct {
	NodeBase
	Expression Expression
}

func (e *ExpressionStatement) Format(f *Formatter) string {
	return e.Expression.Format(f)
}

type InfixExpression struct {
	NodeBase
	Left     Expression
	Operator string
	Right    Expression
}

func (i *InfixExpression) Format(f *Formatter) string {
	return fmt.Sprintf("%s %s %s", i.Left.Format(f), i.Operator, i.Right.Format(f))
}

type PrefixExpression struct {
	NodeBase
	Operator string
	Right    Expression
}

func (p *PrefixExpression) Format(f *Formatter) string {
	return p.Operator + p.Right.Format(f)
}

type GroupedExpression struct {
	NodeBase
	Expression Expression
}

func (g *GroupedExpression) Format(f *Formatter) string {
	return fmt.Sprintf("(%s)", g.Expression.Format(f))
}

type PostfixExpression struct {
	NodeBase
	Left     Expression
	Operator string
}

func (p *PostfixExpression) Format(f *Formatter) string {
	return p.Left.Format(f) + p.Operator
}
