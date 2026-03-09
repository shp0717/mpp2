package parser


import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"strconv"
)


type Line struct {
	Raw string
	Text string
	LineNum int
	File string
	Lines int
}


type Bracket struct {
	Type string
	Start int
}


func RemoveIndentation(s string) string {
	lines := strings.Split(s, "\n")
	var minIndent *string = nil
	var indentRegex = regexp.MustCompile(`^(\s*)(\S.*)$`)
	for _, line := range lines {
		if matches := indentRegex.FindStringSubmatch(strings.TrimSpace(line)); matches != nil {
			indent := matches[1]
			trimmedLine := matches[2]
			if trimmedLine != "" {
				if minIndent == nil {
					minIndent = &indent
				} else {
					for i := 0; i < len(indent) && i < len(*minIndent); i++ {
						if indent[i] != (*minIndent)[i] {
							*minIndent = (*minIndent)[:i]
							break
						}
					}
				}
			}
		}
	}
	if minIndent != nil && *minIndent != "" {
		for i, line := range lines {
			if strings.HasPrefix(line, *minIndent) {
				lines[i] = line[len(*minIndent):]
			}
		}
	}
	return strings.Join(lines, "\n")
}


func AddIndentation(s string, indent string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}


func SetIndentation(s string, indent string) string {
	return AddIndentation(RemoveIndentation(s), indent)
}


func EncodeStrings(raw string) (string, error) {
	index := 0
	var result strings.Builder
	var stringValue strings.Builder
	inString := false
	escaping := false
	for index < len(raw) {
		char := raw[index]
		if inString {
			stringValue.WriteByte(char)
			if escaping {
				escaping = false
			} else if char == '\\' {
				escaping = true
			} else if char == '"' {
				inString = false
				unpacked, err := strconv.Unquote(stringValue.String())
				if err != nil {
					return "", fmt.Errorf("SyntaxError: invalid syntax")
				}
				encoded := hex.EncodeToString([]byte(unpacked))
				result.WriteString(`"` + encoded + `"`)
				stringValue.Reset()
			}
		} else {
			if char == '"' && !inString {
				inString = true
				stringValue.WriteByte('"')
			} else {
				result.WriteByte(char)
			}
		}
		index++
	}
	return result.String(), nil
}


func DecodeStrings(raw string) (string, error) {
	var result strings.Builder
	var stringValue strings.Builder
	inString := false
	index := 0
	for index < len(raw) {
		char := raw[index]
		if inString {
			stringValue.WriteByte(char)
			if char == '"' {
				inString = false
				encoded := stringValue.String()[1 : len(stringValue.String())-1]
				decodedBytes, err := hex.DecodeString(encoded)
				if err != nil {
					return "", fmt.Errorf("SyntaxError: invalid syntax")
				}
				encodedString, err := json.Marshal(string(decodedBytes))
				if err != nil {
					return "", fmt.Errorf("SyntaxError: invalid syntax")
				}
				result.Write(encodedString)
				stringValue.Reset()
			}
		} else if char == '"' {
			inString = true
			stringValue.WriteByte('"')
		} else {
			result.WriteByte(char)
		}
		index++
	}
	return result.String(), nil
}


func DeleteComments(raw string) string {
	// Remove block comments first
	raw = BlockCommentRegex.ReplaceAllFunc(raw, func(comment string) string {
		lines := 0
		for _, char := range comment {
			if char == '\n' {
				lines++
			}
		}
		return strings.Repeat("\n", lines)
	})
	// Remove line comments
	raw = LineCommentRegex.ReplaceAll(raw, "")
	return raw
}


func SplitStatements(raw string, delimiter string, startLine int, file string, encodeParentheses bool) ([]Line, error) {
	lines := []Line{}
	currentLineNum := startLine
	currentStatement := strings.Builder{}
	rawStatement := strings.Builder{}
	brackets := map[rune]rune{
		'(': ')',
		'[': ']',
		'{': '}',
	}
	bracketStack := []Bracket{}
	index := 0
	for index < len(raw) {
		char := rune(raw[index])
		rawStatement.WriteRune(char)
		currentStatement.WriteRune(char)
		if encodeParentheses {
			for open, close := range brackets {
				switch char {
				case open:
					bracketStack = append(bracketStack, Bracket{Type: string(open), Start: currentStatement.Len() - 1})
				case close:
					if len(bracketStack) > 0 && bracketStack[len(bracketStack)-1].Type == string(open) {
						replaceStart := bracketStack[len(bracketStack)-1].Start + 1
						replaceEnd := currentStatement.Len() - 1
						bracketStack = bracketStack[:len(bracketStack)-1]
						statement := currentStatement.String()
						encoded := hex.EncodeToString([]byte(statement[replaceStart:replaceEnd]))
						currentStatement.Reset()
						currentStatement.WriteString(statement[:replaceStart])
						currentStatement.WriteString(encoded)
						currentStatement.WriteString(statement[replaceEnd:])
					}
				}
			}
		}
		if char == '\n' {
			currentLineNum++
		}
		if len(bracketStack) == 0 {
			if strings.HasSuffix(currentStatement.String(), delimiter) || strings.HasSuffix(currentStatement.String(), "\n") || index == len(raw)-1 {
				totalLines := strings.Count(currentStatement.String(), "\n")
				// Remove the delimiter from the end of the statement
				if strings.HasSuffix(currentStatement.String(), delimiter) || strings.HasSuffix(currentStatement.String(), "\n") {
					currentStatementStr := currentStatement.String()
					currentStatement.Reset()
					currentStatement.WriteString(currentStatementStr[:len(currentStatementStr)-len(delimiter)])
				}
				// Remove leading empty lines from rawStatement
				rawStatementLines := strings.Split(rawStatement.String(), "\n")
				removedLines := []string{}
				for i, line := range rawStatementLines {
					if strings.TrimSpace(line) == "" {
						continue
					} else {
						removedLines = rawStatementLines[i:]
						break
					}
				}
				removedLinesStr := strings.Join(removedLines, "\n")
				// Decode string from rawStatement
				decodedRawStatement, err := DecodeStrings(rawStatement.String())
				if err != nil {
					return nil, fmt.Errorf("SyntaxError: invalid syntax")
				}
				rawStatement.Reset()
				rawStatement.WriteString(decodedRawStatement)
				// Clear empty lines from rawStatement
				rawStatementLines = strings.Split(rawStatement.String(), "\n")
				cleanedRawStatement := []string{}
				for _, line := range rawStatementLines {
					if strings.TrimSpace(line) != "" {
						cleanedRawStatement = append(cleanedRawStatement, line)
					}
				}
				rawStatement.Reset()
				rawStatement.WriteString(strings.Join(cleanedRawStatement, "\n"))
				lines = append(lines, Line{
					Raw: rawStatement.String(),
					Text: strings.TrimSpace(currentStatement.String()),
					LineNum: currentLineNum - strings.Count(removedLinesStr, "\n"),
					File: file,
					Lines: totalLines,
				})
				currentStatement.Reset()
				rawStatement.Reset()
			}
		}
		index++
	}
	return lines, nil
}
