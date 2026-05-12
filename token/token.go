package token

type TokenType string

type Token struct {
	Type                TokenType
	Literal             string
	Line                int
	Column              int
	HadWhitespaceBefore bool
}

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"
	NEWLINE = "NEWLINE"

	// identifiers
	IDENT  = "IDENT"
	INT    = "INT"
	STRING = "STRING"
	FLOAT  = "FLOAT"
	// operators
	ASSIGN = "="
	ARROW  = "<-"
	WALRUS = ":="

	PLUS  = "+"
	SUB   = "-"
	SLASH = "/"
	MUL   = "*"
	MOD   = "%"

	PLUS_ASSIGN  = "+="
	SUB_ASSIGN   = "-="
	SLASH_ASSIGN = "/="
	MUL_ASSIGN   = "*="
	MOD_ASSIGN   = "%="

	INC = "++"
	DEC = "--"

	AND = "&"
	OR  = "|"
	SHL = "<<"
	SHR = ">>"
	XOR = "^"

	AND_ASSIGN = "&="
	OR_ASSIGN  = "|="
	SHL_ASSIGN = "<<="
	SHR_ASSIGN = ">>="
	XOR_ASSIGN = "^="

	BANG = "!"
	EQ   = "=="
	NEQ  = "!="
	LT   = "<"
	GT   = ">"
	LTE  = "<="
	GTE  = ">="

	LAND = "&&"
	LOR  = "||"

	COMMA     = ","
	SEMICOLON = ";"
	COLON     = ":"
	DOT       = "."
	ELLIPSIS  = "..."
	DUODOT    = ".."

	LPAREN   = "("
	RPAREN   = ")"
	LBRACE   = "{"
	RBRACE   = "}"
	LBRACKET = "["
	RBRACKET = "]"

	// keywords
	VAR       = "VAR"
	CONST     = "CONST"
	IMPORT    = "IMPORT"
	TYPE      = "TYPE"
	STRUCT    = "STRUCT"
	ENUM      = "ENUM"
	INTERFACE = "INTERFACE"
	IF        = "IF"
	ELSE      = "ELSE"
	SWITCH    = "SWITCH"
	SELECT    = "SELECT"
	CASE      = "CASE"
	DEFAULT   = "DEFAULT"
	WITH      = "WITH"
	MAP       = "MAP"
	FUNC      = "FUNC"
	RETURN    = "RETURN"
	CONTINUE  = "CONTINUE"
	DEFER     = "DEFER"
	START     = "START"
	CHAN      = "CHAN"
	FOR       = "FOR"
	RANGE     = "RANGE"
	WHILE     = "WHILE"
	BREAK     = "BREAK"
	TRUE      = "TRUE"
	FALSE     = "FALSE"
	NIL       = "NIL"

	INT_TYPE    = "INT_TYPE"
	FLOAT_TYPE  = "FLOAT_TYPE"
	STRING_TYPE = "STRING_TYPE"
	BOOL_TYPE   = "BOOL_TYPE"
)

var keywords = map[string]TokenType{
	"say":       VAR,
	"keep":      CONST,
	"import":    IMPORT,
	"type":      TYPE,
	"struct":    STRUCT,
	"enum":      ENUM,
	"interface": INTERFACE,
	"ayla":      IF,
	"elen":      ELSE,
	"choose":    SWITCH,
	"select":    SELECT,
	"when":      CASE,
	"otherwise": DEFAULT,
	"with":      WITH,
	"map":       MAP,
	"fun":       FUNC,
	"give":      RETURN,
	"defer":     DEFER,
	"start":     START,
	"chan":      CHAN,
	"int":       INT_TYPE,
	"float":     FLOAT_TYPE,
	"string":    STRING_TYPE,
	"bool":      BOOL_TYPE,
	"for":       FOR,
	"range":     RANGE,
	"while":     WHILE,
	"snap":      BREAK,
	"next":      CONTINUE,
	"yes":       TRUE,
	"no":        FALSE,
	"nil":       NIL,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
