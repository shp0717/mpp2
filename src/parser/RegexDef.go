package parser
import (
	"regexp"
	"fmt"
	"strings"
)

type Regex struct {
	Pattern string
}

type RegexMatch struct {
	Regex *Regex
	Text  string
	Match []string
}

func (r *Regex) Match(text string) *RegexMatch {
	matches := regexp.MustCompile(r.Pattern).FindStringSubmatch(text)
	if matches == nil {
		return nil
	}
	return &RegexMatch{
		Regex: r,
		Text:  text,
		Match: matches,
	}
}

func (r *Regex) IsMatch(text string) bool {
	return regexp.MustCompile(r.Pattern).MatchString(text)
}

func (r *Regex) ReplaceAll(text string, replacement string) string {
	return regexp.MustCompile(r.Pattern).ReplaceAllString(text, replacement)
}

func (r *Regex) ReplaceAllFunc(text string, replacementFunc func(string) string) string {
	return regexp.MustCompile(r.Pattern).ReplaceAllStringFunc(text, replacementFunc)
}

func (rm *RegexMatch) Group(index int) string {
	if index < 0 || index >= len(rm.Match) {
		return ""
	}
	return rm.Match[index]
}

// Values in () / [] / {} / "" are base-16 encoded
var (  // Define statements and base regex patterns for the language
	// Matches a single line comment, starting with '//' and continuing until the end of the line.
	LineCommentRegex = Regex{ Pattern: `(//.*)` }

	// Matches a block comment, starting with '/*' and ending with '*/', allowing for any characters in between, including newlines.
	BlockCommentRegex = Regex{ Pattern: `(/\*[\s\S]*?\*/)` }

	// Matches an in-line value, which can be part of a larger expression
	FullNameRegex = Regex{ Pattern: `([a-zA-Z0-9_.()\[\]{}"]+)` }

	// Matches an operation value, which can be part of a larger expression
	InLineValueRegex = Regex{ Pattern: `([a-zA-Z0-9_.()\[\]{}"+\-*/%<>=!&|^?: ]+)` }

	// Matches a lazy operation value, which can be part of a larger expression
	InLineLazyValueRegex = Regex{ Pattern: `([a-zA-Z0-9_.()\[\]{}"+\-*/%<>=!&|^?: ]+?)` }

	// Matches a base-16 encoded value
	Base16ValueRegex = Regex{ Pattern: `([a-fA-F0-9]*)` }

	// Matches a function call, capturing the function name and its arguments (Base-16 encoded)
	FunctionCallRegex = Regex{ Pattern: fmt.Sprintf(`^%s\s*\(%s\)$`, FullNameRegex.Pattern, Base16ValueRegex.Pattern) }

	// Matches a validate name
	ValidNameRegex = Regex{ Pattern: `([a-zA-Z_][a-zA-Z0-9_]*)` }

	// Matches a if statement, capturing the condition and the body (Base-16 encoded), allowing else if and else statements in back of the if statement
	// Syntax: if (cond) {b16} ...(May have else if and else statements, but we will only define the regex for the if statement, and we will parse the else if and else statements in the parser)
	IfStatementRegex = Regex{ Pattern: fmt.Sprintf(`^if\s*\(%s\)\s*\{%s\}`, Base16ValueRegex.Pattern, Base16ValueRegex.Pattern) }

	// Matches an else if statement, capturing the condition and the body (Base-16 encoded), allowing else if and else statements in back of the else if statement
	ElseIfStatementRegex = Regex{ Pattern: fmt.Sprintf(`^else\s+if\s*\(%s\)\s*\{%s\}`, Base16ValueRegex.Pattern, Base16ValueRegex.Pattern) }

	// Matches an else statement, capturing the body (Base-16 encoded)
	ElseStatementRegex = Regex{ Pattern: fmt.Sprintf(`^else\s*\{%s\}$`, Base16ValueRegex.Pattern) }

	// Matches a C-style for loop, capturing the initialization, condition, and increment (Base-16 encoded)
	ForLoopRegex = Regex{ Pattern: fmt.Sprintf(`^for\s*\(%s\)\s*\{%s\}$`, Base16ValueRegex.Pattern, Base16ValueRegex.Pattern) }

	// Matches a for-each loop, capturing the header (Base-16 encoded) and the body (Base-16 encoded)
	ForEachRegex = Regex{ Pattern: fmt.Sprintf(`^for\s*\(%s\)\s*\{%s\}$`, Base16ValueRegex.Pattern, Base16ValueRegex.Pattern) }

	// Matches a while loop, capturing the condition and the body (Base-16 encoded)
	WhileLoopRegex = Regex{ Pattern: fmt.Sprintf(`^while\s*\(%s\)\s*\{%s\}$`, InLineValueRegex.Pattern, Base16ValueRegex.Pattern) }

	// Matches a do-while loop, capturing the body and the condition (Base-16 encoded)
	DoWhileLoopRegex = Regex{ Pattern: fmt.Sprintf(`^do\s*\{%s\}\s*while\s*\(%s\)$`, Base16ValueRegex.Pattern, InLineValueRegex.Pattern) }

	// Matches a function definition, capturing the function name, parameters (Base-16 encoded), and body (Base-16 encoded)
	FuncDefRegex = Regex{ Pattern: fmt.Sprintf(`^func\s+%s\s*\(%s\)\s*\{%s\}$`, FullNameRegex.Pattern, Base16ValueRegex.Pattern, Base16ValueRegex.Pattern) }

	// Matches an anonymous function definition, capturing the parameters (Base-16 encoded) and body (Base-16 encoded)
	AnonyFuncDefRegex = Regex{ Pattern: fmt.Sprintf(`^func\s*\(%s\)\s*\{%s\}$`, Base16ValueRegex.Pattern, Base16ValueRegex.Pattern) }

	// Matches a return statement, capturing the return value
	ReturnStatementRegex = Regex{ Pattern: fmt.Sprintf(`^return\b%s$`, InLineValueRegex.Pattern) }

	// Matches a set statement, capturing the settable name and the value
	SetStatementRegex = Regex{ Pattern: fmt.Sprintf(`^%s\s*=\s*%s$`, FullNameRegex.Pattern, InLineValueRegex.Pattern) }

	// Matches a delete statement, capturing the deletable name
	DeleteStatementRegex = Regex{ Pattern: fmt.Sprintf(`^delete\s+%s$`, FullNameRegex.Pattern) }

	// Matches a get statement, capturing the gettable name and the key / index (Base-16 encoded)
	GetItemRegex = Regex{ Pattern: fmt.Sprintf(`^%s\[%s\]$`, FullNameRegex.Pattern, Base16ValueRegex.Pattern) }

	// Matches a get statement, capturing the gettable name and the attribute (Base-16 encoded)
	GetAttrRegex = Regex{ Pattern: fmt.Sprintf(`^%s\.%s$`, FullNameRegex.Pattern, ValidNameRegex.Pattern) }

	// Matches a break statement
	BreakStatementRegex = Regex{ Pattern: `^break$` }

	// Matches a continue statement
	ContinueStatementRegex = Regex{ Pattern: `^continue$` }

	// Matches a label definition, capturing the label name
	LabelDefRegex = Regex{ Pattern: fmt.Sprintf(`^%s:$`, ValidNameRegex.Pattern) }

	// Matches a goto statement, capturing the label name
	GotoStatementRegex = Regex{ Pattern: fmt.Sprintf(`^goto\s+%s$`, ValidNameRegex.Pattern) }

	// Matches an import statement, capturing the module name (Base-16 encoded)
	ImportStatementRegex = Regex{ Pattern: fmt.Sprintf(`^import\s*"%s"$`, Base16ValueRegex.Pattern) }

	// Matches the entry point of the program, which is a line that starts with '!Meow++'
	EntryPointRegex = Regex{ Pattern: `^!Meow\+\+$` }
)

var (  // Define operation regex patterns for the language
	// Matches an unary operation, capturing the operator and the operand
	UnaryRegex = Regex{ Pattern: fmt.Sprintf(`^(\+|-|!)\s*%s$`, InLineValueRegex.Pattern) }

	// Matches a prefix increment or decrement operation, capturing the operator and the operand
	ExprFirstRegex = Regex{ Pattern: fmt.Sprintf(`^(\+\+|--)\s*%s$`, FullNameRegex.Pattern)}

	// Matches a postfix increment or decrement operation, capturing the operator and the operand
	ExprLastRegex = Regex{ Pattern: fmt.Sprintf(`^%s\s*(\+\+|--)$`, FullNameRegex.Pattern) }

	// Matches an addition or subtraction operation, capturing the left and right operands, and the operator
	AddSubRegex = Regex{ Pattern: fmt.Sprintf(`^%s\s*(\+|-)\s*%s$`, InLineValueRegex.Pattern, InLineValueRegex.Pattern) }

	// Matches a multiplication, division, or modulus operation, capturing the left and right operands, and the operator
	MulDivModRegex = Regex{ Pattern: fmt.Sprintf(`^%s\s*(\*|/|%%)\s*%s$`, InLineValueRegex.Pattern, InLineValueRegex.Pattern) }

	// Matches a power operation, capturing the left and right operands
	PowRegex = Regex{ Pattern: fmt.Sprintf(`^%s\s*\^\s*%s$`, InLineLazyValueRegex.Pattern, InLineValueRegex.Pattern) }

	// Matches a logical AND operation, capturing the left and right operands
	LogicalAndRegex = Regex{ Pattern: fmt.Sprintf(`^%s\s*&&\s*%s$`, InLineValueRegex.Pattern, InLineValueRegex.Pattern) }

	// Matches a logical OR operation, capturing the left and right operands
	LogicalOrRegex = Regex{ Pattern: fmt.Sprintf(`^%s\s*\|\|\s*%s$`, InLineValueRegex.Pattern, InLineValueRegex.Pattern) }

	// Matches a logical XOR operation, capturing the left and right operands
	LogicalXorRegex = Regex{ Pattern: fmt.Sprintf(`^%s\s*\^\^\s*%s$`, InLineValueRegex.Pattern, InLineValueRegex.Pattern) }

	// Matches a equal or not equal operation, capturing the left and right operands, and the operator
	EqualNotEqualRegex = Regex{ Pattern: fmt.Sprintf(`^%s\s*(==|!=)\s*%s$`, InLineValueRegex.Pattern, InLineValueRegex.Pattern) }

	// Matches a comparison operation, capturing the left and right operands, and the operator
	ComparisonRegex = Regex{ Pattern: fmt.Sprintf(`^%s\s*(<=|>=|<|>)\s*%s$`, InLineValueRegex.Pattern, InLineValueRegex.Pattern) }

	// Matches an operation like +=, -=, *=, /=, %=, ^=, &&=, ||=, ^^=, capturing the left and right operands, operators
	OperatorSetRegex = Regex{ Pattern: fmt.Sprintf(`^%s\s*(\+|-|\*|/|%%|\^|&&|\|\||\^\^)=\s*%s$`, FullNameRegex.Pattern, InLineValueRegex.Pattern) }

	// Matches a ternary conditional operation, capturing the condition, true expression, and false expression
	TernaryRegex = Regex{ Pattern: fmt.Sprintf(`^%s\s*\?\s*%s\s*:\s*%s$`, InLineLazyValueRegex.Pattern, InLineValueRegex.Pattern, InLineValueRegex.Pattern) }

	// Matches a shift operation, capturing the left and right operands, and the operator
	ShiftRegex = Regex{ Pattern: fmt.Sprintf(`^%s\s*(<<|>>)\s*%s$`, InLineValueRegex.Pattern, InLineValueRegex.Pattern) }
)

var (  // Define literal regex patterns for the language
	// Matches a string literal, capturing the content inside double quotes (Base-16 encoded)
	StringLiteralRegex = Regex{ Pattern: `^"([a-fA-F0-9]*)"$` }

	// Matches a decimal number literal or float number, capturing the number
	NumberLiteralRegex = Regex{ Pattern: `^(\d+(\.\d+)?)$` }

	// Matches a hexadecimal number literal, capturing the digits
	HexLiteralRegex = Regex{ Pattern: `^0x([a-fA-F0-9]+)$` }

	// Matches a boolean literal, capturing the value (true or false)
	BooleanLiteralRegex = Regex{ Pattern: `^(true|false)$` }

	// Matches a null literal
	NullLiteralRegex = Regex{ Pattern: `^null$` }

	// Matches a value in parentheses, capturing the content inside the parentheses (Base-16 encoded)
	ParenthesesRegex = Regex{ Pattern: fmt.Sprintf(`^\(%s\)$`, Base16ValueRegex.Pattern) }

	// Matches a value in square brackets, capturing the content inside the square brackets (Base-16 encoded)
	SquareBracketsRegex = Regex{ Pattern: fmt.Sprintf(`^\[%s\]$`, Base16ValueRegex.Pattern) }

	// Matches a value in curly braces, capturing the content inside the curly braces (Base-16 encoded)
	CurlyBracesRegex = Regex{ Pattern: fmt.Sprintf(`^\{%s\}$`, Base16ValueRegex.Pattern) }
)

var ReservedKeywords = map[string]struct{}{
	"true": {},
	"false": {},
	"null": {},
	"if": {},
	"else": {},
	"for": {},
	"while": {},
	"do": {},
	"func": {},
	"return": {},
	"break": {},
	"continue": {},
	"goto": {},
	"import": {},
}

func ValidateName(name string) bool {
	for _, part := range strings.Split(name, ".") {
		if _, isReserved := ReservedKeywords[part]; !ValidNameRegex.IsMatch(part) || isReserved {
			return false
		}
	}
	return true
}
