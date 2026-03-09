# Meow++ (MPP2)

Meow++ is a small interpreted language implemented in Go. This repository contains the MPP2 runtime, parser, REPL, AST exporter, and a native compilation path that wraps Meow++ source inside a generated Go program.

Current status: Version 2.0.0 Alpha  
Primary source extension: `.mpp`  
AST extension: `.mst` (JSON)  
Runtime model: dynamic objects with magic-method based operators and dispatch

## Overview

The implementation in this repository is centered around three layers:

- A parser that converts `.mpp` source into a JSON-like command tree.
- A runtime that evaluates commands and values through typed objects and magic methods.
- A CLI that can run source files, run prebuilt AST files, export AST, launch a REPL, and compile source into a native binary.

## Repository Layout

| Path | Purpose |
| --- | --- |
| `main.go` | CLI entry point and command dispatcher. |
| `parser/` | Statement splitting, regex definitions, and AST generation. |
| `runtime/` | Object system, evaluator, built-ins, operators, and control-flow execution. |
| `utils/` | Top-level helpers for running source, running AST, REPL, AST export, and compile. |

## CLI

The executable accepts either subcommands or a direct file path. The current implementation exposes the following commands:

```text
mpp2 help
mpp2 run <file.mpp>
mpp2 run <file.mst>
mpp2 ast <file.mpp> <file.mst>
mpp2 repl
mpp2 compile <file.mpp> <output_binary>
mpp2 version
```

You can also run a file directly without the `run` subcommand:

```text
mpp2 program.mpp
mpp2 program.mst
```

In a normal release package, this project is expected to be distributed as Go source files, a prebuilt executable, and this README. The README therefore documents the runtime interface and source layout, but does not assume an extra packaging script is included.

## Add The Executable To PATH

If you want to run `mpp2` from any terminal location, place the executable in a folder that is already on your `PATH`, or add the folder containing the executable to your `PATH`.

### macOS and Linux

A common approach is to move the executable into `/usr/local/bin` if you have permission:

```sh
sudo cp mpp2 /usr/local/bin/mpp2
sudo chmod +x /usr/local/bin/mpp2
```

If you prefer to keep the executable in your home directory, create a personal bin directory and add it to your shell configuration:

```sh
mkdir -p $HOME/.local/bin
cp mpp2 $HOME/.local/bin/mpp2
chmod +x $HOME/.local/bin/mpp2
```

Then add this line to your `~/.zshrc` or `~/.bashrc`:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

After reloading the shell, verify the command:

```sh
mpp2 help
```

### Windows

Place `mpp2.exe` in a permanent folder such as `C:\Tools\mpp2`, then add that folder to the user PATH in System Settings.

1. Open **System Properties**.
2. Open **Environment Variables**.
3. Select the user **Path** variable.
4. Add the folder containing `mpp2.exe`.
5. Open a new terminal and run `mpp2 help`.

You can also do it in PowerShell:

```powershell
$target = "$HOME\AppData\Local\mpp2"
New-Item -ItemType Directory -Force -Path $target
Copy-Item .\mpp2.exe "$target\mpp2.exe" -Force
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";$target", "User")
```

## Language Basics

Meow++ is expression-oriented and dynamically typed. The parser recognizes the following core statement forms:

- Assignment: `name = value`
- Compound assignment: `+=`, `-=`, `*=`, `/=`, `%=`, `^=`, `&&=`, `||=`, `^^=`
- Deletion: `delete target`
- Conditionals: `if`, `else if`, `else`
- Loops: C-style `for`, foreach-style `for`, `while`, `do ... while`
- Functions: named `func` and anonymous `func`
- Flow control: `return`, `break`, `continue`, labels, and `goto`
- Imports: source-file imports and built-in module imports through the same `import "..."` syntax

### Program entry

`!Meow++` is recommended on the first line, but it is optional.

It exists mainly for language compatibility. In Meow++ 1, `!Meow` was the required and unique entry point, and code before that entry point was only used for definitions and was not executed. Meow++ 2 keeps a non-mandatory entry marker in order to preserve that original language feature.

If a `!Meow++` entry marker is present, the program skips everything before the first `!Meow++`. That skipped portion is not executed and its definitions are not loaded.

### Literals and values

- Numbers: `1`, `3.14`, `0xFF`
- Strings: regular double-quoted strings in source
- Booleans: `true`, `false`
- Null: `null`
- Arrays: `[1, 2, 3]`
- Maps: `{"name": "Meow", "year": 2026}`
- Attribute access: `obj.attr`
- Indexing: `obj[key]`
- Function calls: `fn(a, b)`

### Operators

The current parser and runtime implement these operator families:

- Arithmetic: `+`, `-`, `*`, `/`, `%`, `^`
- Comparison: `<`, `>`, `<=`, `>=`, `==`, `!=`
- Logical: `&&`, `||`, `^^`, unary `!`
- Unary numeric: unary `+` and unary `-`
- Ternary: `cond ? a : b`
- Shift-like magic operators: `<<` and `>>`
- Increment and decrement expressions are also recognized by the parser.

The `<<` and `>>` operators are especially important in this implementation because standard stream objects use them for output and input.

### Example

```text
!Meow++
import "math"

target = math.random.int(1, 100)
stdout << "Guess a number between 1 and 100!\n"
guess = ""

while (true) {
    stdout << "> "
    stdin >> guess
    guess_num = number(guess)
    if (guess_num < target) {
        stdout << "  Too low!\n"
    } else if (guess_num > target) {
        stdout << "  Too high!\n"
    } else {
        stdout << "  Correct!\n"
        break
    }
}
```

## Built-in Types

The runtime exposes these core object types:

- `number`
- `string`
- `bool`
- `null`
- `array`
- `map`
- `function`
- `subruntime`

Behavior is implemented through per-type magic methods, so operators, conversions, attribute access, indexing, and calls are all dispatched dynamically through the runtime object model.

## Built-in Variables and Functions

| Name | Description |
| --- | --- |
| `stdout` | Writable stream object. Uses `<<` to print. |
| `stdin` | Readable stream object. Uses `>>` to read input into a target. |
| `stderr` | Error output stream. Uses `<<` to print to stderr. |
| `number(x)` | Convert a value to a number. |
| `string(x)` | Convert a value to a string. |
| `array(x)` | Convert a value to an array. |
| `map(x)` | Convert a value to a map. |
| `len(x)` | Return the length of a value. |
| `type(x)` | Return the runtime type name. |
| `set_magic(obj, name, fn)` | Attach a custom magic method to an object. |
| `set_type(obj, name)` | Override the runtime type string of an object. |
| `exit(code?)` | Exit the process with an optional numeric code. |

## Built-in Modules

Built-in modules are imported with the same syntax as file modules, for example `import "math"`.

| Module | Available members in the current implementation |
| --- | --- |
| `builtins` | Exports the built-in variables and functions as a module namespace. |
| `unicode` | `upper`, `lower`, `ords`, `chrs` |
| `fileio` | `write`, `read`, `exists` |
| `memory` | `raw`, `size`, `usage` |
| `format` | `sprintf`, `sscanf` |
| `stdio` | `print`, `println`, `input` |
| `math` | `random.random`, `random.uniform`, `random.int`, `random.gauss`, `sin`, `cos`, `tan`, `asin`, `acos`, `atan`, `pi`, `e`, `sqrt`, `log`, `ln`, `log10`, `log2` |

## AST Workflow

The parser can export a source file into a `.mst` file. That file is plain JSON and can be executed directly by the runtime.

```sh
cd src
mpp2 ast ../test.mpp ./test.mst
mpp2 run ./test.mst
```

This matches the implementation in `utils.ParseMppAst` and `utils.RunMppAst`.

## REPL

The REPL keeps a persistent runtime, accepts multi-line input while brackets remain open, and rolls back runtime state for failed evaluations. Use:

```sh
cd src
mpp2 repl
```

## Native Compilation

The `compile` command does not compile Meow++ directly to machine code. Instead, it generates a temporary Go program that embeds the original source and runs it through the Meow++ runtime, then builds that Go program as a native executable.

**Important:** when working from the Go source tree, the compile path depends on the Meow++ runtime source being available under the application environment directory.

```sh
cd src
mpp2 compile ../test.mpp ../test_binary
```

## Notes and Limitations

- This repository is currently marked as Alpha.
- The documented behavior here follows the implementation in the Go source, not an external language spec.
- Built-in and user-defined behavior both rely heavily on runtime magic methods.
- Release bundles typically only need the executable, the Go source tree, and this README.
- If you rebuild or use `compile` from source, the runtime source tree still needs to be present in the Meow++ environment directory.

[GitHub Repository](https://github.com/shp0717/mpp2)  
[中文說明](./readme-zh.md)
