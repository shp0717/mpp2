package main

import (
	"mpp2/utils"
	"mpp2/meta"
	"os"
	"strings"
	"fmt"
)

func main() {
	utils.Init()
	if len(os.Args) < 2 {
		fmt.Println("Usage 1: mpp2 <file.mpp>")
		fmt.Println("Usage 2: mpp2 <file.mst>")
		fmt.Println("Usage 3: mpp2 <command> <arguments>")
		fmt.Println("         Use 'mpp2 help' for more information.")
		return
	}
	switch os.Args[1] {
	case "help":
		fmt.Println("List of commands:")
		fmt.Println("  mpp2 help                               - Show this help message")
		fmt.Println("  mpp2 run <file.mpp>                     - Run a .mpp source file")
		fmt.Println("  mpp2 run <file.mst>                     - Run a .mst AST file")
		fmt.Println("  mpp2 ast <file.mpp> <file.mst>          - Parse a .mpp file and save its AST to a .mst file")
		fmt.Println("  mpp2 repl                               - Start an interactive REPL session")
		fmt.Println("  mpp2 compile <file.mpp> <output_binary> - Compile a .mpp file to a native binary")
	case "run":
		if len(os.Args) != 3 {
			fmt.Fprintln(os.Stderr, "Usage: mpp2 run <file.mpp|file.mst>")
			os.Exit(1)
		}
		if strings.HasSuffix(os.Args[2], ".mpp") {
			err := utils.RunMppSource(os.Args[2])
			if err != nil {
				os.Exit(1)
			}
		} else if strings.HasSuffix(os.Args[2], ".mst") {
			err := utils.RunMppAst(os.Args[2])
			if err != nil {
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Unknown file type: %s\n", os.Args[2])
			os.Exit(1)
		}
	case "ast":
		if len(os.Args) != 4 {
			fmt.Fprintln(os.Stderr, "Usage: mpp2 ast <input.mpp> <output.mst>")
			os.Exit(1)
		}
		if !strings.HasSuffix(os.Args[2], ".mpp") {
			fmt.Fprintf(os.Stderr, "Input file must have .mpp extension: %s\n", os.Args[2])
			os.Exit(1)
		}
		inputFile := os.Args[2]
		outputFile := os.Args[3]
		utils.ParseMppAst(inputFile, outputFile)
	case "repl":
		if len(os.Args) != 2 {
			fmt.Fprintln(os.Stderr, "Usage: mpp2 repl")
			os.Exit(1)
		}
		utils.Repl()
	case "compile":
		if len(os.Args) != 4 {
			fmt.Fprintln(os.Stderr, "Usage: mpp2 compile <input.mpp> <output_binary>")
			os.Exit(1)
		}
		inputFile := os.Args[2]
		outputFile := os.Args[3]
		if !strings.HasSuffix(inputFile, ".mpp") {
			fmt.Fprintf(os.Stderr, "Input file must have .mpp extension: %s\n", inputFile)
			os.Exit(1)
		}
		err := utils.Compile(inputFile, outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Compilation failed: %v\n", err)
			os.Exit(1)
		}
	case "version":
		fmt.Printf("Meow++ %s, on %s/%s\n", meta.FULLVER, meta.OS, meta.ARCH)
	default:
		if len(os.Args) == 2 {
			if strings.HasSuffix(os.Args[1], ".mpp") {
				err := utils.RunMppSource(os.Args[1])
				if err != nil {
					os.Exit(1)
				}
			} else if strings.HasSuffix(os.Args[1], ".mst") {
				err := utils.RunMppAst(os.Args[1])
				if err != nil {
					os.Exit(1)
				}
			} else {
				fmt.Fprintf(os.Stderr, "Unknown file type: %s\n", os.Args[1])
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
			os.Exit(1)
		}
	}
}
