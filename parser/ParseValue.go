package parser


import (
	"strconv"
	"strings"
	"encoding/hex"
	"fmt"
)


func HexDecode(hexStr string) (string, error) {
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", fmt.Errorf("SyntaxError: invalid syntax in hexadecimal encoding: %s", hexStr)
	}
	return string(bytes), nil
}


func UnquoteString(s string) (string, error) {
	if !strings.HasPrefix(s, "\"") || !strings.HasSuffix(s, "\"") {
		return "", fmt.Errorf("string is not properly quoted: %s", s)
	}
	hexDecoded, err := HexDecode(s[1:len(s)-1])
	if err != nil {
		return "", err
	}
	return hexDecoded, nil
}


/*
Parsing steps:

1. Parentheses

2. Square Brackets

3. Curly Braces

4. Double Quotes

5. ?:

6. ||

7. &&

8. ^^

9. == !=

10. < > <= >=

11. << >>

12. + -

13. * / %

14. ! + - (unary)

15. **

16. Function Calls

17. Get Key / Index

18. Get Attribute

19. Literal

20. Variable
*/
func ParseValue(value string) (any, error) {
	value = strings.TrimSpace(value)
	if AnonyFuncDefRegex.IsMatch(value) {
		line := Line{
			Text: value,
			LineNum: 1,
			File: "<function>",
			Raw: value,
			Lines: 0,
		}
		cmd, err := AnonyFuncDef(line)
		if err != nil {
			return nil, err
		}
		return cmd, nil
	}
	// 1. Parentheses
	if ParenthesesRegex.IsMatch(value) {
		match := ParenthesesRegex.Match(value)
		encoded := match.Group(1)
		decoded, err := HexDecode(encoded)
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in parentheses")
		}
		return ParseValue(decoded)
	}
	// 2. Square Brackets
	if SquareBracketsRegex.IsMatch(value) {
		encoded := SquareBracketsRegex.Match(value).Group(1)
		decoded, err := HexDecode(encoded)
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in square brackets")
		}
		args := strings.Split(decoded, ",")
		if len(args) > 0 && strings.TrimSpace(args[len(args)-1]) == "" {
			args = args[:len(args)-1]
		}
		parseArgs := make([]any, len(args))
		for i := range args {
			trimmed := strings.TrimSpace(args[i])
			if trimmed == "" && i != len(args)-1 {
				return nil, fmt.Errorf("SyntaxError: empty value in square brackets")
			}
			parsedArg, err := ParseValue(trimmed)
			if err != nil {
				return nil, fmt.Errorf("SyntaxError: invalid syntax in square brackets: %v", err)
			}
			parseArgs[i] = parsedArg
		}
		return parseArgs, nil
	}
	// 3. Curly Braces
	if CurlyBracesRegex.IsMatch(value) {
		encoded := CurlyBracesRegex.Match(value).Group(1)
		decoded, err := HexDecode(encoded)
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in curly braces")
		}
		pairs := strings.Split(decoded, ",")
		if strings.TrimSpace(pairs[len(pairs)-1]) == "" {
			pairs = pairs[:len(pairs)-1]
		}
		keys := []any{}
		values := []any{}
		for _, pair := range pairs {
			kv := strings.SplitN(pair, ":", 2)
			if len(kv) != 2 {
				return nil, fmt.Errorf("SyntaxError: invalid key-value pair in curly braces")
			}
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			var parsedKey any
			if strings.HasPrefix(key, "\"") && strings.HasSuffix(key, "\"") {
				unquotedKey, err := UnquoteString(key)
				if err != nil {
					return nil, fmt.Errorf("SyntaxError: invalid syntax in curly braces key: %v", err)
				}
				parsedKey = unquotedKey
			} else if strings.HasPrefix(key, "[") && strings.HasSuffix(key, "]") {
				encodedKey := key[1:len(key)-1]
				decodedKey, err := HexDecode(encodedKey)
				if err != nil {
					return nil, fmt.Errorf("SyntaxError: invalid syntax in curly braces key: %v", err)
				}
				parsedKey, err = ParseValue(decodedKey)
				if err != nil {
					return nil, fmt.Errorf("SyntaxError: invalid syntax in curly braces key: %v", err)
				}
			} else {
				parsedKey = key
			}
			parsedValue, err := ParseValue(value)
			if err != nil {
				return nil, fmt.Errorf("SyntaxError: invalid syntax in curly braces value: %v", err)
			}
			keys = append(keys, parsedKey)
			values = append(values, parsedValue)
		}
		return map[string]any{
			"cmd": "map",
			"keys": keys,
			"values": values,
		}, nil
	}
	// 5. ?:
	if TernaryRegex.IsMatch(value) {
		match := TernaryRegex.Match(value)
		condition, err := ParseValue(match.Group(1))
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in ternary condition: %v", err)
		}
		trueExpr, err := ParseValue(match.Group(2))
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in ternary true expression: %v", err)
		}
		falseExpr, err := ParseValue(match.Group(3))
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in ternary false expression: %v", err)
		}
		return map[string]any{
			"cmd": "ternary",
			"cond": condition,
			"true_expr": trueExpr,
			"false_expr": falseExpr,
		}, nil
	}
	// 6. ||
	if LogicalOrRegex.IsMatch(value) {
		match := LogicalOrRegex.Match(value)
		left, err := ParseValue(match.Group(1))
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in logical OR left operand: %v", err)
		}
		right, err := ParseValue(match.Group(2))
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in logical OR right operand: %v", err)
		}
		return map[string]any{
			"cmd": "operation",
			"operator": "||",
			"left": left,
			"right": right,
		}, nil
	}
	// 7. &&
	if LogicalAndRegex.IsMatch(value) {
		match := LogicalAndRegex.Match(value)
		left, err := ParseValue(match.Group(1))
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in logical AND left operand: %v", err)
		}
		right, err := ParseValue(match.Group(2))
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in logical AND right operand: %v", err)
		}
		return map[string]any{
			"cmd": "operation",
			"operator": "&&",
			"left": left,
			"right": right,
		}, nil
	}
	// 8. ^^
	if LogicalXorRegex.IsMatch(value) {
		match := LogicalXorRegex.Match(value)
		left, err := ParseValue(match.Group(1))
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in logical XOR left operand: %v", err)
		}
		right, err := ParseValue(match.Group(2))
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in logical XOR right operand: %v", err)
		}
		return map[string]any{
			"cmd": "operation",
			"operator": "^^",
			"left": left,
			"right": right,
		}, nil
	}
	// 9. == !=
	if EqualNotEqualRegex.IsMatch(value) {
		match := EqualNotEqualRegex.Match(value)
		left, err := ParseValue(match.Group(1))
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in equality left operand: %v", err)
		}
		right, err := ParseValue(match.Group(3))
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in equality right operand: %v", err)
		}
		operator := match.Group(2)
		return map[string]any{
			"cmd": "operation",
			"operator": operator,
			"left": left,
			"right": right,
		}, nil
	}
	// 10. < > <= >=
	if ComparisonRegex.IsMatch(value) {
		match := ComparisonRegex.Match(value)
		leftRaw := match.Group(1)
		if !strings.HasSuffix(leftRaw, "<") && !strings.HasSuffix(leftRaw, ">") {
			left, err := ParseValue(leftRaw)
			if err == nil {
				right, err := ParseValue(match.Group(3))
				if err == nil {
					operator := match.Group(2)
					return map[string]any{
						"cmd": "operation",
						"operator": operator,
						"left": left,
						"right": right,
					}, nil
				}
			}
		}
	}
	// 11. << >>
	if ShiftRegex.IsMatch(value) {
		match := ShiftRegex.Match(value)
		left, err := ParseValue(match.Group(1))
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in shift left operand: %v", err)
		}
		right, err := ParseValue(match.Group(3))
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in shift right operand: %v", err)
		}
		operator := match.Group(2)
		return map[string]any{
			"cmd": "operation",
			"operator": operator,
			"left": left,
			"right": right,
		}, nil
	}
	// 12. + -
	if AddSubRegex.IsMatch(value) {
		match := AddSubRegex.Match(value)
		left, err := ParseValue(match.Group(1))
		if err == nil {
			right, err := ParseValue(match.Group(3))
			if err == nil {
				operator := match.Group(2)
				return map[string]any{
					"cmd": "operation",
					"operator": operator,
					"left": left,
					"right": right,
				}, nil
			}
		}
	}
	// 13. * / %
	if MulDivModRegex.IsMatch(value) {
		match := MulDivModRegex.Match(value)
		left, err := ParseValue(match.Group(1))
		if err == nil {
			right, err := ParseValue(match.Group(3))
			if err == nil {
				operator := match.Group(2)
				return map[string]any{
					"cmd": "operation",
					"operator": operator,
					"left": left,
					"right": right,
				}, nil
			}
		}
	}
	// 14. ! + - (unary)
	if ExprFirstRegex.IsMatch(value) {
		match := ExprFirstRegex.Match(value)
		operator := match.Group(1)
		operand := match.Group(2)
		operandValue, err := ParseValue(operand)
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in unary operation operand: %v", err)
		}
		return map[string]any{
			"cmd": "self_incr_decr",
			"operator": operator,
			"target": ToSettable(operand),
			"operand": operandValue,
			"position": "prefix",
		}, nil
	}
	if ExprLastRegex.IsMatch(value) {
		match := ExprLastRegex.Match(value)
		operator := match.Group(2)
		operand := match.Group(1)
		operandValue, err := ParseValue(operand)
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in unary operation operand: %v", err)
		}
		return map[string]any{
			"cmd": "self_incr_decr",
			"operator": operator,
			"target": ToSettable(operand),
			"operand": operandValue,
			"position": "postfix",
		}, nil
	}
	if UnaryRegex.IsMatch(value) {
		match := UnaryRegex.Match(value)
		operator := match.Group(1)
		operand, err := ParseValue(match.Group(2))
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in unary operation operand: %v", err)
		}
		return map[string]any{
			"cmd": "unary",
			"operator": operator,
			"operand": operand,
		}, nil
	}
	// 15. ^
	if PowRegex.IsMatch(value) {
		match := PowRegex.Match(value)
		left, err := ParseValue(match.Group(1))
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in power left operand: %v", err)
		}
		right, err := ParseValue(match.Group(2))
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in power right operand: %v", err)
		}
		return map[string]any{
			"cmd": "operation",
			"operator": "^",
			"left": left,
			"right": right,
		}, nil
	}
	// 16. Function Calls
	if FunctionCallRegex.IsMatch(value) {
		match := FunctionCallRegex.Match(value)
		funcName := match.Group(1)
		encodedArgs := match.Group(2)
		decodedArgs, err := HexDecode(encodedArgs)
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in function call arguments")
		}
		args := strings.Split(decodedArgs, ",")
		if len(args) > 0 && strings.TrimSpace(args[len(args)-1]) == "" {
			args = args[:len(args)-1]
		}
		parseArgs := make([]map[string]any, len(args))
		for i := range args {
			trimmed := strings.TrimSpace(args[i])
			if trimmed == "" && i != len(args)-1 {
				return nil, fmt.Errorf("SyntaxError: empty argument in function call")
			}
			unpack := false
			if strings.HasSuffix(trimmed, "...") {
				unpack = true
				trimmed = strings.TrimSuffix(trimmed, "...")
			}
			argValue, err := ParseValue(trimmed)
			if err != nil {
				return nil, fmt.Errorf("SyntaxError: invalid syntax in function call argument: %v", err)
			}
			parseArgs[i] = map[string]any{
				"value": argValue,
				"unpack": unpack,
			}
		}
		function, err := ParseValue(funcName)
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in function call: %v", err)
		}
		return map[string]any{
			"cmd": "function_call",
			"function": function,
			"args": parseArgs,
		}, nil
	}
	// 17. Get Key / Index
	if GetItemRegex.IsMatch(value) {
		match := GetItemRegex.Match(value)
		object, err := ParseValue(match.Group(1))
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in object for key/index access: %v", err)
		}
		encodedKey := match.Group(2)
		decodedKey, err := HexDecode(encodedKey)
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in key/index access")
		}
		key, err := ParseValue(decodedKey)
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in key/index access: %v", err)
		}
		return map[string]any{
			"cmd": "get_item",
			"object": object,
			"key": key,
		}, nil
	}
	// 18. Get Attribute
	if GetAttrRegex.IsMatch(value) {
		match := GetAttrRegex.Match(value)
		object, err := ParseValue(match.Group(1))
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in object for attribute access: %v", err)
		}
		attrName := match.Group(2)
		return map[string]any{
			"cmd": "get_item",
			"object": object,
			"key": attrName,
		}, nil
	}
	// 19. Literal
	if StringLiteralRegex.IsMatch(value) {
		match := StringLiteralRegex.Match(value)
		decoded, err := HexDecode(match.Group(1))
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in string literal: %v", err)
		}
		return decoded, nil
	}
	if HexLiteralRegex.IsMatch(value) {
		match := HexLiteralRegex.Match(value)
		decoded := 0
		for _, c := range match.Group(1) {
			decoded *= 16
			if c >= '0' && c <= '9' {
				decoded += int(c - '0')
			} else if c >= 'a' && c <= 'f' {
				decoded += int(c - 'a' + 10)
			} else if c >= 'A' && c <= 'F' {
				decoded += int(c - 'A' + 10)
			} else {
				return nil, fmt.Errorf("SyntaxError: invalid syntax in hexadecimal literal")
			}
		}
		return decoded, nil
	}
	if NumberLiteralRegex.IsMatch(value) {
		match := NumberLiteralRegex.Match(value)
		numStr := match.Group(1)
		if strings.Contains(numStr, ".") {
			floatVal, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return nil, fmt.Errorf("SyntaxError: invalid syntax in number literal")
			}
			return floatVal, nil
		} else {
			intVal, err := strconv.Atoi(numStr)
			if err != nil {
				return nil, fmt.Errorf("SyntaxError: invalid syntax in number literal")
			}
			return intVal, nil
		}
	}
	if BooleanLiteralRegex.IsMatch(value) {
		match := BooleanLiteralRegex.Match(value)
		boolStr := match.Group(1)
		boolVal, err := strconv.ParseBool(boolStr)
		if err != nil {
			return nil, fmt.Errorf("SyntaxError: invalid syntax in boolean literal")
		}
		return boolVal, nil
	}
	if NullLiteralRegex.IsMatch(value) {
		return nil, nil
	}
	// 20. Variable
	if ValidNameRegex.IsMatch(value) {
		return map[string]any{
			"cmd": "var",
			"name": value,
		}, nil
	}
	return nil, fmt.Errorf("SyntaxError: invalid syntax")
}
