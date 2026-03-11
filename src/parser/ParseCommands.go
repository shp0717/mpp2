package parser

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"mpp2/meta"
)

var CommandParsers = []func(Line) ([]map[string]any, error, bool, bool){}

func ToSettable(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if GetItemRegex.IsMatch(raw) {
		itemMatch := GetItemRegex.Match(raw)
		current := itemMatch.Group(1)
		keyStr := itemMatch.Group(2)
		decodedKey, err := HexDecode(keyStr)
		if err != nil {
			return nil
		}
		parsedKey, err := ParseValue(decodedKey)
		if err != nil {
			return nil
		}
		key := parsedKey
		target, err := ParseValue(current)
		if err != nil {
			return nil
		}
		return map[string]any{
			"type": "item",
			"target": target,
			"key": key,
		}
	} else if GetAttrRegex.IsMatch(raw) {
		attrMatch := GetAttrRegex.Match(raw)
		current := attrMatch.Group(1)
		attrName := attrMatch.Group(2)
		key := attrName
		target, err := ParseValue(current)
		if err != nil {
			return nil
		}
		return map[string]any{
			"type": "item",
			"target": target,
			"key": key,
		}
	} else {
		return map[string]any{
			"type": "var",
			"name": raw,
		}
	}
}

func SetStatement(l string, v string) (map[string]any, error) {
	left := strings.TrimSpace(l)
	value, err := ParseValue(v)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"cmd": "set",
		"target": ToSettable(left),
		"value": value,
	}, nil
}

func GetExc(line Line) string {
	file := line.File
	lineNum := line.LineNum
	raw := line.Raw
	return fmt.Sprintf("  File \033[1;35m%s\033[0m, line \033[1m%d\033[0m:\n\033[1;33m%s\033[0m", file, lineNum, SetIndentation(raw, "    "))
}

func RaiseError(line Line, message string) error {
	exc := GetExc(line)
	return fmt.Errorf("Traceback (most recent call last):\n%s\n\033[1;31m%s\033[0m", exc, message)
}

func DecodeAll(encoded string) string {
	DoubleQuoted := Regex{Pattern: `"([0-9a-fA-F]*)"`}
	Parenthesized := Regex{Pattern: `\((0[0-9a-fA-F]*)\)`}
	SquareBracketed := Regex{Pattern: `\[(0[0-9a-fA-F]*)\]`}
	CurlyBracketed := Regex{Pattern: `\{(0[0-9a-fA-F]*)\}`}
	QuoteDecoder := func(match string) string {
		decoded, err := HexDecode(match)
		if err != nil {
			return ""
		}
		return decoded
	}
	BracketDecoder := func(match string) string {
		decoded, err := HexDecode(match)
		if err != nil {
			return ""
		}
		return DecodeAll(decoded)
	}
	encoded = DoubleQuoted.ReplaceAllFunc(encoded, QuoteDecoder)
	encoded = Parenthesized.ReplaceAllFunc(encoded, BracketDecoder)
	encoded = SquareBracketed.ReplaceAllFunc(encoded, BracketDecoder)
	encoded = CurlyBracketed.ReplaceAllFunc(encoded, BracketDecoder)
	return encoded
}

func IfStatement(line Line) (map[string]any, error) {
	text := line.Text
	linenum := line.LineNum
	conds := []map[string]any{}
	if IfStatementRegex.IsMatch(text) {
		match := IfStatementRegex.Match(text)
		all := match.Match[0]
		encodedCondition := match.Group(1)
		decodedCondition, err := HexDecode(encodedCondition)
		if err != nil {
			return nil, err
		}
		stmtLine := Line{
			Text: decodedCondition,
			LineNum: linenum,
			File: line.File,
			Raw: DecodeAll(encodedCondition),
			Lines: 0,
		}
		parsedCondition, err := ParseValue(decodedCondition)
		if err != nil {
			return nil, err
		}
		encodedBodyStr := match.Group(2)
		bodyStr, err := HexDecode(encodedBodyStr)
		if err != nil {
			return nil, err
		}
		bodyLines, err := SplitStatements(bodyStr, ";", linenum, line.File, false)
		if err != nil {
			return nil, err
		}
		for _, line := range bodyLines {
			linenum += line.Lines
		}
		bodyCmds, err := ParseAll(bodyLines)
		if err != nil {
			return nil, err
		}
		ifStmt := map[string]any{
			"cond": parsedCondition,
			"body": bodyCmds,
			"exc": GetExc(stmtLine),
		}
		conds = append(conds, ifStmt)
		text = strings.TrimPrefix(text, all)
		text = strings.TrimSpace(text)
	} else {
		return nil, RaiseError(line, "Invalid if statement syntax")
	}
	for match := ElseIfStatementRegex.Match(text); match != nil; match = ElseIfStatementRegex.Match(text) {
		encodedCondition := match.Group(1)
		all := match.Match[0]
		decodedCondition, err := HexDecode(encodedCondition)
		if err != nil {
			return nil, err
		}
		stmtLine := Line{
			Text: decodedCondition,
			LineNum: linenum,
			File: line.File,
			Raw: DecodeAll(encodedCondition),
			Lines: 0,
		}
		parsedCondition, err := ParseValue(decodedCondition)
		if err != nil {
			return nil, err
		}
		encodedBodyStr := match.Group(2)
		bodyStr, err := HexDecode(encodedBodyStr)
		if err != nil {
			return nil, err
		}
		bodyLines, err := SplitStatements(bodyStr, ";", linenum, line.File, false)
		if err != nil {
			return nil, err
		}
		for _, line := range bodyLines {
			linenum += line.Lines
		}
		bodyCmds, err := ParseAll(bodyLines)
		if err != nil {
			return nil, err
		}
		elseIfStmt := map[string]any{
			"cond": parsedCondition,
			"body": bodyCmds,
			"exc": GetExc(stmtLine),
		}
		conds = append(conds, elseIfStmt)
		text = strings.TrimPrefix(text, all)
		text = strings.TrimSpace(text)
	}
	if ElseStatementRegex.IsMatch(text) {
		match := ElseStatementRegex.Match(text)
		all := match.Match[0]
		encodedBodyStr := match.Group(1)
		bodyStr, err := HexDecode(encodedBodyStr)
		if err != nil {
			return nil, err
		}
		bodyLines, err := SplitStatements(bodyStr, ";", linenum, line.File, false)
		if err != nil {
			return nil, err
		}
		bodyCmds, err := ParseAll(bodyLines)
		if err != nil {
			return nil, err
		}
		elseStmt := map[string]any{
			"cond": true,
			"body": bodyCmds,
			"exc": "",
		}
		conds = append(conds, elseStmt)
		text = strings.TrimPrefix(text, all)
		text = strings.TrimSpace(text)
	}
	if text != "" {
		return nil, RaiseError(line, "SyntaxError: unexpected text after if statement")
	}
	return map[string]any{
		"cmd": "if_chunk",
		"conds": conds,
	}, nil
}

func ForLoop(line Line) (map[string]any, error) {
	match := ForLoopRegex.Match(line.Text)
	if match == nil {
		return nil, RaiseError(line, "ParserError: invalid for loop syntax")
	}
	encodedHeader := match.Group(1)
	headerStr, err := HexDecode(encodedHeader)
	if err != nil {
		return nil, err
	}
	headerLine := Line{
		Text: headerStr,
		LineNum: line.LineNum,
		File: line.File,
		Raw: DecodeAll(encodedHeader),
		Lines: 0,
	}
	headerParts := strings.Split(headerStr, ";")
	if len(headerParts) != 3 {
		return nil, RaiseError(line, "SyntaxError: for loop header must have 3 parts separated by ';'")
	}
	initStr := headerParts[0]
	initLines, err := SplitStatements(initStr, ",", line.LineNum, line.File, false)
	if err != nil {
		return nil, err
	}
	initCmds, err := ParseAll(initLines)
	if err != nil {
		return nil, err
	}
	condStr := headerParts[1]
	parsedCond, err := ParseValue(condStr)
	if err != nil {
		return nil, err
	}
	incrStr := headerParts[2]
	incrLines, err := SplitStatements(incrStr, ",", line.LineNum, line.File, false)
	if err != nil {
		return nil, err
	}
	incrCmds, err := ParseAll(incrLines)
	if err != nil {
		return nil, err
	}
	bodyEncoded := match.Group(2)
	bodyStr, err := HexDecode(bodyEncoded)
	if err != nil {
		return nil, err
	}
	bodyLines, err := SplitStatements(bodyStr, ";", line.LineNum, line.File, false)
	if err != nil {
		return nil, err
	}
	bodyCmds, err := ParseAll(bodyLines)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"cmd": "for_loop",
		"init": initCmds,
		"cond": parsedCond,
		"incr": incrCmds,
		"body": bodyCmds,
		"exc": GetExc(headerLine),
	}, nil
}

func ForEach(line Line) (map[string]any, error) {
	match := ForEachRegex.Match(line.Text)
	if match == nil {
		return nil, RaiseError(line, "ParserError: invalid for-each loop syntax")
	}
	encodedHeader := match.Group(1)
	headerStr, err := HexDecode(encodedHeader)
	if err != nil {
		return nil, err
	}
	headerLine := Line{
		Text: headerStr,
		LineNum: line.LineNum,
		File: line.File,
		Raw: DecodeAll(encodedHeader),
		Lines: 0,
	}
	headerParts := strings.SplitN(headerStr, ":", 2)
	if len(headerParts) != 2 {
		return nil, RaiseError(line, "SyntaxError: for-each loop header must have 2 parts separated by ':'")
	}
	iterableStr := headerParts[1]
	parsedIterable, err := ParseValue(iterableStr)
	if err != nil {
		return nil, err
	}
	target := headerParts[0]
	bodyEncoded := match.Group(2)
	bodyStr, err := HexDecode(bodyEncoded)
	if err != nil {
		return nil, err
	}
	bodyLines, err := SplitStatements(bodyStr, ";", line.LineNum, line.File, false)
	if err != nil {
		return nil, err
	}
	bodyCmds, err := ParseAll(bodyLines)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"cmd": "for_each",
		"target": ToSettable(target),
		"iterable": parsedIterable,
		"body": bodyCmds,
		"exc": GetExc(headerLine),
	}, nil
}

func WhileLoop(line Line) (map[string]any, error) {
	match := WhileLoopRegex.Match(line.Text)
	if match == nil {
		return nil, RaiseError(line, "ParserError: invalid while loop syntax")
	}
	encodedCondition := match.Group(1)
	decodedCondition, err := HexDecode(encodedCondition)
	if err != nil {
		return nil, err
	}
	condLine := Line{
		Text: decodedCondition,
		LineNum: line.LineNum,
		File: line.File,
		Raw: DecodeAll(encodedCondition),
		Lines: 0,
	}
	parsedCondition, err := ParseValue(decodedCondition)
	if err != nil {
		return nil, err
	}
	bodyEncoded := match.Group(2)
	bodyStr, err := HexDecode(bodyEncoded)
	if err != nil {
		return nil, err
	}
	bodyLines, err := SplitStatements(bodyStr, ";", line.LineNum, line.File, false)
	if err != nil {
		return nil, err
	}
	bodyCmds, err := ParseAll(bodyLines)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"cmd": "while_loop",
		"cond": parsedCondition,
		"body": bodyCmds,
		"exc": GetExc(condLine),
	}, nil
}

func DoWhileLoop(line Line) (map[string]any, error) {
	match := DoWhileLoopRegex.Match(line.Text)
	if match == nil {
		return nil, RaiseError(line, "ParserError: invalid do-while loop syntax")
	}
	bodyEncoded := match.Group(1)
	bodyStr, err := HexDecode(bodyEncoded)
	if err != nil {
		return nil, err
	}
	bodyLines, err := SplitStatements(bodyStr, ";", line.LineNum, line.File, false)
	if err != nil {
		return nil, err
	}
	bodyCmds, err := ParseAll(bodyLines)
	if err != nil {
		return nil, err
	}
	encodedCondition := match.Group(2)
	decodedCondition, err := HexDecode(encodedCondition)
	if err != nil {
		return nil, err
	}
	condLine := Line{
		Text: decodedCondition,
		LineNum: line.LineNum,
		File: line.File,
		Raw: DecodeAll(encodedCondition),
		Lines: 0,
	}
	parsedCondition, err := ParseValue(decodedCondition)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"cmd": "do_while_loop",
		"cond": parsedCondition,
		"body": bodyCmds,
		"exc": GetExc(condLine),
	}, nil
}

func FuncDef(line Line) (map[string]any, error) {
	match := FuncDefRegex.Match(line.Text)
	if match == nil {
		return nil, RaiseError(line, "ParserError: invalid function definition syntax")
	}
	name := match.Group(1)
	target := ToSettable(name)
	encodedParams := match.Group(2)
	paramsStr, err := HexDecode(encodedParams)
	if err != nil {
		return nil, err
	}
	infArgs := false
	params := []string{}
	if strings.HasPrefix(paramsStr, "...") {
		infArgs = true
		paramsStr = strings.TrimPrefix(paramsStr, "...")
	}
	paramStrs := strings.Split(paramsStr, ",")
	if len(paramStrs) > 0 && paramStrs[len(paramStrs)-1] == "" {
		paramStrs = paramStrs[:len(paramStrs)-1]
	}
	for _, param := range paramStrs {
		param = strings.TrimSpace(param)
		if param != "" {
			params = append(params, param)
		} else {
			return nil, RaiseError(line, "SyntaxError: invalid parameter name in function definition")
		}
	}
	bodyEncoded := match.Group(3)
	bodyStr, err := HexDecode(bodyEncoded)
	if err != nil {
		return nil, err
	}
	bodyLines, err := SplitStatements(bodyStr, ";", line.LineNum, line.File, false)
	if err != nil {
		return nil, err
	}
	bodyCmds, err := ParseAll(bodyLines)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"cmd": "set",
		"target": target,
		"value": map[string]any{
			"cmd": "func_def",
			"params": params,
			"inf_args": infArgs,
			"body": bodyCmds,
		},
		"exc": GetExc(line),
	}, nil
}

func AnonyFuncDef(line Line) (map[string]any, error) {
	match := AnonyFuncDefRegex.Match(line.Text)
	if match == nil {
		return nil, RaiseError(line, "ParserError: invalid anonymous function definition syntax")
	}
	encodedParams := match.Group(1)
	paramsStr, err := HexDecode(encodedParams)
	if err != nil {
		return nil, err
	}
	infArgs := false
	params := []string{}
	if strings.HasPrefix(paramsStr, "...") {
		infArgs = true
		paramsStr = strings.TrimPrefix(paramsStr, "...")
	}
	for _, param := range strings.Split(paramsStr, ",") {
		param = strings.TrimSpace(param)
		if param != "" {
			params = append(params, param)
		} else {
			return nil, RaiseError(line, "SyntaxError: invalid parameter name in function definition")
		}
	}
	bodyEncoded := match.Group(2)
	bodyStr, err := HexDecode(bodyEncoded)
	if err != nil {
		return nil, err
	}
	bodyLines, err := SplitStatements(bodyStr, ";", line.LineNum, line.File, false)
	if err != nil {
		return nil, err
	}
	bodyCmds, err := ParseAll(bodyLines)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"cmd": "func_def",
		"params": params,
		"inf_args": infArgs,
		"body": bodyCmds,
	}, nil
}

func ImportStatement(line Line) (map[string]any, error) {
	match := ImportStatementRegex.Match(line.Text)
	if match == nil {
		return nil, RaiseError(line, "ParserError: invalid import statement syntax")
	}
	encodedPath := match.Group(1)
	path, err := HexDecode(encodedPath)
	if err != nil {
		return nil, err
	}
	file := filepath.Base(path)
	suffix := filepath.Ext(file)
	moduleName := strings.TrimSuffix(file, suffix)
	isAbsolute := filepath.IsAbs(path)
	absPath := path
	if !isAbsolute {
		currentFile := line.File
		currentDir := filepath.Dir(currentFile)
		path = filepath.Join(currentDir, path)
		absPath, err = filepath.Abs(path)
		if err != nil {
			return nil, err
		}
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		builtinLibraryPath := meta.GetEnvFile("lib", file)
		if _, err := os.Stat(builtinLibraryPath); os.IsNotExist(err) {
			return nil, RaiseError(line, fmt.Sprintf("FileNotFoundError: no such file '%s'", path))
		} else {
			absPath = builtinLibraryPath
		}
	}
	moduleContentBytes, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	moduleContent := string(moduleContentBytes)
	moduleCmds, err := Parse(moduleContent, absPath)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"cmd": "import",
		"name": moduleName,
		"content": moduleCmds,
	}, nil
}

func BuiltinImportStatement(line Line) (map[string]any, error) {
	match := ImportStatementRegex.Match(line.Text)
	if match == nil {
		return nil, RaiseError(line, "ParserError: invalid builtin import statement syntax")
	}
	encodedModuleName := match.Group(1)
	moduleName, err := HexDecode(encodedModuleName)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"cmd": "builtin_import",
		"name": moduleName,
	}, nil
}

func ParseCommand(line Line) ([]map[string]any, error) {
	if line.Text == "" {
		return []map[string]any{}, nil
	}
	exc := GetExc(line)
	text := line.Text
	if SetStatementRegex.IsMatch(text) {
		match := SetStatementRegex.Match(text)
		cmd, err := SetStatement(match.Group(1), match.Group(2))
		if err != nil {
			return nil, err
		}
		cmd["exc"] = exc
		return []map[string]any{cmd}, nil
	} else if OperatorSetRegex.IsMatch(text) {
		match := OperatorSetRegex.Match(text)
		left := match.Group(1)
		operator := match.Group(2)
		right := match.Group(3)
		parsedRight := fmt.Sprintf("(%s) %s (%s)", hex.EncodeToString([]byte(left)), operator, hex.EncodeToString([]byte(right)))
		cmd, err := SetStatement(left, parsedRight)
		if err != nil {
			return nil, err
		}
		cmd["exc"] = exc
		return []map[string]any{cmd}, nil
	} else if DeleteStatementRegex.IsMatch(text) {
		match := DeleteStatementRegex.Match(text)
		target := match.Group(1)
		cmd := map[string]any{
			"cmd": "delete",
			"target": ToSettable(target),
			"exc": exc,
		}
		return []map[string]any{cmd}, nil
	} else if IfStatementRegex.IsMatch(text) {
		cmd, err := IfStatement(line)
		if err != nil {
			return nil, err
		}
		cmd["exc"] = exc
		return []map[string]any{cmd}, nil
	} else if ForLoopRegex.IsMatch(text) {
		cmd, err := ForLoop(line)
		if err != nil {
			cmd, err = ForEach(line)
			if err != nil {
				return nil, err
			}
		}
		return []map[string]any{cmd}, nil
	} else if WhileLoopRegex.IsMatch(text) {
		cmd, err := WhileLoop(line)
		if err != nil {
			return nil, err
		}
		return []map[string]any{cmd}, nil
	} else if DoWhileLoopRegex.IsMatch(text) {
		cmd, err := DoWhileLoop(line)
		if err != nil {
			return nil, err
		}
		return []map[string]any{cmd}, nil
	} else if FuncDefRegex.IsMatch(text) {
		cmd, err := FuncDef(line)
		if err != nil {
			return nil, err
		}
		return []map[string]any{cmd}, nil
	} else if ReturnStatementRegex.IsMatch(text) {
		match := ReturnStatementRegex.Match(text)
		valueStr := match.Group(1)
		parsedValue, err := ParseValue(valueStr)
		if err != nil {
			return nil, err
		}
		cmd := map[string]any{
			"cmd": "return",
			"value": parsedValue,
		}
		cmd["exc"] = exc
		return []map[string]any{cmd}, nil
	} else if BreakStatementRegex.IsMatch(text) {
		cmd := map[string]any{
			"cmd": "break",
			"exc": exc,
		}
		return []map[string]any{cmd}, nil
	} else if ContinueStatementRegex.IsMatch(text) {
		cmd := map[string]any{
			"cmd": "continue",
			"exc": exc,
		}
		return []map[string]any{cmd}, nil
	} else if LabelDefRegex.IsMatch(text) {
		match := LabelDefRegex.Match(text)
		labelName := match.Group(1)
		cmd := map[string]any{
			"cmd": "label_def",
			"name": labelName,
		}
		cmd["exc"] = exc
		return []map[string]any{cmd}, nil
	} else if GotoStatementRegex.IsMatch(text) {
		match := GotoStatementRegex.Match(text)
		labelName := match.Group(1)
		cmd := map[string]any{
			"cmd": "goto",
			"name": labelName,
		}
		cmd["exc"] = exc
		return []map[string]any{cmd}, nil
	} else if ImportStatementRegex.IsMatch(text) {
		cmd, err := ImportStatement(line)
		if err != nil {
			cmd, err = BuiltinImportStatement(line)
			if err != nil {
				return nil, err
			}
		}
		cmd["exc"] = exc
		return []map[string]any{cmd}, nil
	} else if EntryPointRegex.IsMatch(text) {
		cmd := map[string]any{
			"cmd": "entry_point",
		}
		cmd["exc"] = exc
		return []map[string]any{cmd}, nil
	} else if GlobalRegex.IsMatch(text) {
		match := GlobalRegex.Match(text)
		varNamesStr := match.Group(1)
		varNames := []string{}
		for _, varName := range strings.Split(varNamesStr, ",") {
			varName = strings.TrimSpace(varName)
			if varName != "" {
				varNames = append(varNames, varName)
			} else {
				return nil, RaiseError(line, "SyntaxError: invalid variable name in global statement")
			}
		}
		cmd := map[string]any{
			"cmd": "global",
			"vars": varNames,
		}
		cmd["exc"] = exc
		return []map[string]any{cmd}, nil
	} else if NonlocalRegex.IsMatch(text) {
		match := NonlocalRegex.Match(text)
		varNamesStr := match.Group(1)
		varNames := []string{}
		for _, varName := range strings.Split(varNamesStr, ",") {
			varName = strings.TrimSpace(varName)
			if varName != "" {
				varNames = append(varNames, varName)
			} else {
				return nil, RaiseError(line, "SyntaxError: invalid variable name in nonlocal statement")
			}
		}
		cmd := map[string]any{
			"cmd": "nonlocal",
			"vars": varNames,
		}
		cmd["exc"] = exc
		return []map[string]any{cmd}, nil
	} else if parsed, err := ParseValue(text); err == nil {
		cmd := map[string]any{
			"cmd": "expr",
			"value": parsed,
		}
		cmd["exc"] = exc
		return []map[string]any{cmd}, nil
	}
	for _, parser := range CommandParsers {
		cmds, err, setExc, matched := parser(line)
		if !matched {
			continue
		}
		if err != nil {
			return nil, err
		}
		if setExc {
			for _, cmd := range cmds {
				cmd["exc"] = exc
			}
		}
		return cmds, nil
	}
	return nil, RaiseError(line, "SyntaxError: invalid syntax")
}

func ParseAll(lines []Line) ([]map[string]any, error) {
	var cmds = []map[string]any{}
	for _, line := range lines {
		lineCmds, err := ParseCommand(line)
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, lineCmds...)
	}
	return cmds, nil
}

func Parse(code string, file string) ([]map[string]any, error) {
	encoded, err := EncodeStrings(code)
	if err != nil {
		return nil, err
	}
	removed := DeleteComments(encoded)
	lines, err := SplitStatements(removed, ";", 1, file, true)
	if err != nil {
		return nil, err
	}
	cmds, err := ParseAll(lines)
	if err != nil {
		return nil, err
	}
	return cmds, nil
}
