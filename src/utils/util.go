package utils

import (
	"mpp2/parser"
	"mpp2/runtime"
	"os"
	"os/exec"
	"fmt"
	"encoding/json"
	"time"
	"path/filepath"
	"errors"

	"mpp2/meta"
	"github.com/peterh/liner"
)

func Init() {
	runtime.Init()
}

func RunAst(ast []map[string]any) error {
	rt := runtime.NewRuntime(
		map[string]runtime.Obj{},
		ast,
	)
	rt.GotoEntryPoint()
	err := rt.Run()
	if err != nil {
		return err
	}
	return nil
}

func RunSource(code string, file string) error {
	ast, err := parser.Parse(code, file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return RunAst(ast)
}

func RunMppSource(file string) error {
	code, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", file, err)
		return err
	}
	return RunSource(string(code), file)
}

func RunMppAst(file string) error {
	code, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", file, err)
		return err
	}
	var ast []map[string]any
	err = json.Unmarshal(code, &ast)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON from file %s: %v\n", file, err)
		return err
	}
	return RunAst(ast)
}

func countBrackets(s string) int {
	count := 0
	for _, char := range s {
		switch char {
		case '(', '{', '[':
			count++
		case ')', '}', ']':
			count--
		}
	}
	return count
}

func Repl() {
	fmt.Printf("Meow++ REPL (version %s, %s %s) on %s/%s\n", meta.VERSION, meta.GOVER, meta.COMPILER, meta.OS, meta.ARCH)
	fmt.Println("Type 'exit()' to quit the REPL.")
	text := ""
	bracketCount := 0
	runtime := runtime.NewRuntime(
		map[string]runtime.Obj{},
		[]map[string]any{},
	)
	reader := liner.NewLiner()
	inputIndex := 1
	defer reader.Close()
	for {
		var prompt string
		if text == "" {
			prompt = ">>> "
		} else {
			prompt = "... "
		}
		input, err := reader.Prompt(prompt)
		if err != nil {
			time.Sleep(200 * time.Millisecond) // Prevent busy loop on Ctrl+D
			fmt.Println()
			break
		}
		if input != "" {
			reader.AppendHistory(input)
		}
		text += input + "\n"
		bracketCount += countBrackets(input)
		if bracketCount == 0 {
			rawRuntime := runtime
			ast, err := parser.Parse(text, fmt.Sprintf("<stdin:%d>", inputIndex))
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				text = ""
				continue
			} else {
				inputIndex++
			}
			runtime.Cmds = append(runtime.Cmds, ast...)
			err = runtime.Run()
			if err != nil {
				runtime = rawRuntime
			}
			text = ""
		}
	}
}

func ParseMppAst(input_file string, output_file string) error {
	code, err := os.ReadFile(input_file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", input_file, err)
		return err
	}
	ast, err := parser.Parse(string(code), input_file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	jsonData, err := json.MarshalIndent(ast, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding AST to JSON: %v\n", err)
		return err
	}
	err = os.WriteFile(output_file, jsonData, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to file %s: %v\n", output_file, err)
		return err
	}
	return nil
}

func copyDir(src string, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dst, relPath)
		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		} else {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(destPath, data, info.Mode())
		}
	})
}

func transpile(input_file string) (string, error) {
	tempFolder := filepath.Join(os.TempDir(), "mpp2_transpiler")
	err := os.MkdirAll(tempFolder, 0755) // Ensure temp folder exists
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temporary folder: %v\n", err)
		return "", err
	}
	template := `package main
import "os"
import "mpp2/utils"
func main() {
	utils.Init()
	sourceCode := %q
	file := %q
	err := utils.RunSource(sourceCode, file)
	if err != nil {
		os.Exit(1)
	}
}
`
	absInputFile, err := filepath.Abs(input_file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting absolute path of %s: %v\n", input_file, err)
		return "", err
	}
	code, err := os.ReadFile(absInputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", absInputFile, err)
		return "", err
	}
	transpiledCode := fmt.Sprintf(template, string(code), absInputFile)
	err = os.MkdirAll(tempFolder, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temporary folder: %v\n", err)
		return "", err
	}
	// Copy source file folder to temp folder
	sourceDir := meta.SOURCEPATH
	err = copyDir(sourceDir, tempFolder)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error copying source files to temporary folder: %v\n", err)
		return "", err
	}
	// Copy runtime package to temp folder if it exists
	runtimeSrc := "./runtime"
	if _, err := os.Stat(runtimeSrc); !os.IsNotExist(err) {
		err = copyDir(runtimeSrc, filepath.Join(tempFolder, "runtime"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error copying runtime package to temporary folder: %v\n", err)
			return "", err
		}
	}
	// Copy parser package to temp folder if it exists
	parserSrc := "./parser"
	if _, err := os.Stat(parserSrc); !os.IsNotExist(err) {
		err = copyDir(parserSrc, filepath.Join(tempFolder, "parser"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error copying parser package to temporary folder: %v\n", err)
			return "", err
		}
	}
	// Copy checker package to temp folder if it exists
	checkerSrc := "./checker"
	if _, err := os.Stat(checkerSrc); !os.IsNotExist(err) {
		err = copyDir(checkerSrc, filepath.Join(tempFolder, "checker"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error copying checker package to temporary folder: %v\n", err)
			return "", err
		}
	}
	// Run go mod tidy in temp folder to ensure dependencies are resolved
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tempFolder
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running 'go mod tidy' in temporary folder: %v\n", err)
		return "", err
	}
	// Write transpiled code to main.go in temp folder
	tempFile := filepath.Join(tempFolder, "main.go")
	err = os.WriteFile(tempFile, []byte(transpiledCode), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing transpiled code to file: %v\n", err)
		return "", err
	}
	return tempFile, nil
}

func Compile(input_file string, output_file string) error {
	mainFile, err := transpile(input_file)
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "build", "-o", output_file, mainFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()

	var exitErr *exec.ExitError
	var pathErr *os.PathError
	var execErr *exec.Error

	if errors.As(err, &exitErr) {
		fmt.Fprintf(os.Stderr, "Compilation failed with exit code %d\n", exitErr.ExitCode())
	} else if errors.As(err, &pathErr) {
		fmt.Fprintf(os.Stderr, "File error: %v\n", pathErr)
	} else if errors.As(err, &execErr) {
		fmt.Fprintf(os.Stderr, "Execution error: %v\n", execErr)
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "Unexpected error: %v\n", err)
	}

	return err
}
