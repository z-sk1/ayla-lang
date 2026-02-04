package token

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
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
	ASSIGN   = "="
	PLUS     = "+"
	MINUS    = "-"
	SLASH    = "/"
	ASTERISK = "*"
	MOD      = "%"
	WALRUS   = "WALRUS"

	EQ     = "EQ"
	NOT_EQ = "NOT_EQ"
	LT     = "<"
	GT     = ">"
	LTE    = "LTE"
	GTE    = "GTE"

	BANG = "!"
	AND  = "AND"
	OR   = "OR"

	COMMA     = ","
	SEMICOLON = ";"
	COLON     = ":"
	DOT       = "."

	LPAREN   = "("
	RPAREN   = ")"
	LBRACE   = "{"
	RBRACE   = "}"
	LBRACKET = "["
	RBRACKET = "]"

	// keywords
	VAR      = "VAR"
	CONST    = "CONST"
	TYPE     = "TYPE"
	STRUCT   = "STRUCT"
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
)

var keywords = map[string]TokenType{
	"egg":       VAR,
	"rock":      CONST,
	"type":      TYPE,
	"struct":    STRUCT,
	"ayla":      IF,
	"elen":      ELSE,
	"decide":    SWITCH,
	"when":      CASE,
	"otherwise": DEFAULT,
	"with":      WITH,
	"map":       MAP,
	"in":        IN,
	"fun":       FUNC,
	"back":      RETURN,
	"spawn":     SPAWN,
	"int":       INT_TYPE,
	"float":     FLOAT_TYPE,
	"string":    STRING_TYPE,
	"bool":      BOOL_TYPE,
	"thing":     ANY_TYPE,
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
