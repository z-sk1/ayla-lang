package token

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
}

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

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

	LPAREN   = "("
	RPAREN   = ")"
	LBRACE   = "{"
	RBRACE   = "}"
	LBRACKET = "["
	RBRACKET = "]"

	// keywords
	VAR      = "VAR"
	CONST    = "CONST"
	PRINT    = "PRINT"
	IF       = "IF"
	ELSE     = "ELSE"
	FUNC     = "FUNC"
	RETURN   = "RETURN"
	CONTINUE = "CONTINUE"
	FOR      = "FOR"
	WHILE    = "WHILE"
	BREAK    = "BREAK"
	TRUE     = "TRUE"
	FALSE    = "FALSE"

	SCANLN = "SCANLN"

	INT_TYPE    = "INT_TYPE"
	FLOAT_TYPE  = "FLOAT_TYPE"
	STRING_TYPE = "STRING_TYPE"
	BOOL_TYPE   = "BOOL_TYPE"
)

var keywords = map[string]TokenType{
	"egg":  VAR,
	"rock": CONST,

	"ayla": IF,
	"elen": ELSE,

	"blueprint": FUNC,
	"back":      RETURN,

	"int":    INT_TYPE,
	"float":  FLOAT_TYPE,
	"string": STRING_TYPE,
	"bool":   BOOL_TYPE,

	"four":   FOR,
	"why":    WHILE,
	"kitkat": BREAK,
	"next":   CONTINUE,

	"yes": TRUE,
	"no":  FALSE,

	"explode": PRINT,
	"tsaln":   SCANLN,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
