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
	WALRUS = ":="
	PLUS   = "+"
	SUB    = "-"
	SLASH  = "/"
	MUL    = "*"
	MOD    = "%"

	PLUS_ASSIGN  = "+="
	SUB_ASSIGN   = "-="
	SLASH_ASSIGN = "/="
	MUL_ASSIGN   = "*="

	INC = "++"
	DEC = "--"

	EQ  = "=="
	NEQ = "!="
	LT  = "<"
	GT  = ">"
	LTE = "<="
	GTE = ">="

	BANG = "!"
	AND  = "&"
	OR   = "|"
	SHL  = "<<"
	SHR  = ">>"
	XOR  = "^"

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
	VAR      = "VAR"
	CONST    = "CONST"
	IMPORT   = "IMPORT"
	TYPE     = "TYPE"
	STRUCT   = "STRUCT"
	ENUM     = "ENUM"
	IF       = "IF"
	ELSE     = "ELSE"
	SWITCH   = "SWITCH"
	CASE     = "CASE"
	DEFAULT  = "DEFAULT"
	WITH     = "WITH"
	MAP      = "MAP"
	IN       = "IN"
	FUNC     = "FUNC"
	RETURN   = "RETURN"
	CONTINUE = "CONTINUE"
	DEFER    = "DEFER"
	SPAWN    = "SPAWN"
	FOR      = "FOR"
	RANGE    = "RANGE"
	WHILE    = "WHILE"
	BREAK    = "BREAK"
	TRUE     = "TRUE"
	FALSE    = "FALSE"
	NIL      = "NIL"

	INT_TYPE    = "INT_TYPE"
	FLOAT_TYPE  = "FLOAT_TYPE"
	STRING_TYPE = "STRING_TYPE"
	BOOL_TYPE   = "BOOL_TYPE"
	ANY_TYPE    = "ANY_TYPE"
	ERROR_TYPE  = "ERROR_TYPE"
)

var keywords = map[string]TokenType{
	"egg":       VAR,
	"rock":      CONST,
	"import":    IMPORT,
	"type":      TYPE,
	"struct":    STRUCT,
	"enum":      ENUM,
	"ayla":      IF,
	"elen":      ELSE,
	"choose":    SWITCH,
	"when":      CASE,
	"otherwise": DEFAULT,
	"with":      WITH,
	"map":       MAP,
	"in":        IN,
	"fun":       FUNC,
	"back":      RETURN,
	"defer":     DEFER,
	"spawn":     SPAWN,
	"int":       INT_TYPE,
	"float":     FLOAT_TYPE,
	"string":    STRING_TYPE,
	"bool":      BOOL_TYPE,
	"thing":     ANY_TYPE,
	"error":     ERROR_TYPE,
	"four":      FOR,
	"range":     RANGE,
	"why":       WHILE,
	"kitkat":    BREAK,
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
