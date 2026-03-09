package runtime

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	goruntime "runtime"
	"strings"
	"unsafe"

	"github.com/peterh/liner"
)

var BUILTIN_VARS, BUILTIN_MODULES map[string]Obj = map[string]Obj{}, map[string]Obj{}

var InitList = []func(){
	InitMagicRefs,
	InitBuiltinVars,
	InitBuiltinModules,
}

func Init() {
	for _, initFunc := range InitList {
		initFunc()
	}
}

func AddInitFunc(initFunc func()) error {
	InitList = append(InitList, initFunc)
	return nil
}

func InitBuiltinVars() {
	numberFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for float(), got %d", len(args))
		}
		return ToNumber(args[0])
	})
	if err != nil {
		panic(err)
	}
	stringFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for str(), got %d", len(args))
		}
		return ToString(args[0])
	})
	if err != nil {
		panic(err)
	}
	arrayFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for array(), got %d", len(args))
		}
		return ToArray(args[0])
	})
	if err != nil {
		panic(err)
	}
	mapFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for map(), got %d", len(args))
		}
		return ToMap(args[0])
	})
	if err != nil {
		panic(err)
	}
	lenFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for len(), got %d", len(args))
		}
		return Length(args[0])
	})
	if err != nil {
		panic(err)
	}
	typeFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for type(), got %d", len(args))
		}
		return NewString(args[0].Type)
	})
	if err != nil {
		panic(err)
	}
	setMagicFunc, err := NewFunction(func(args []Obj) (Obj, error) {  // args: [obj, magicName, magicFunc]
		if len(args) != 3 {
			return nil, fmt.Errorf("RuntimeError: expected 3 arguments for set_magic(), got %d", len(args))
		}
		if args[1].Type != "string" {
			return nil, fmt.Errorf("TypeError: expected string for magic name in set_magic(), got %s", args[1].Type)
		}
		if args[2].Type != "function" {
			return nil, fmt.Errorf("TypeError: expected function for magic function in set_magic(), got %s", args[2].Type)
		}
		obj := args[0]
		magicName, ok := args[1].Value.(string)
		if !ok {
			return nil, fmt.Errorf("TypeError: expected string for magic name in set_magic(), got %T", args[1].Value)
		}
		magicFunc, ok := args[2].Value.(func([]Obj) (Obj, error))
		if !ok {
			return nil, fmt.Errorf("TypeError: expected function for magic function in set_magic(), got %T", args[2].Value)
		}
		if obj.Magic == nil {
			obj.Magic = make(map[string]ObjMagic)
		}
		obj.Magic[magicName] = func (self Obj, args []Obj) (Obj, error) {
			return magicFunc(append([]Obj{self}, args...))
		}
		return obj, nil
	})
	if err != nil {
		panic(err)
	}
	setTypeFunc, err := NewFunction(func(args []Obj) (Obj, error) {  // args: [obj, typeName]
		if len(args) != 2 {
			return nil, fmt.Errorf("RuntimeError: expected 2 arguments for set_type(), got %d", len(args))
		}
		if args[1].Type != "string" {
			return nil, fmt.Errorf("TypeError: expected string for type name in set_type(), got %s", args[1].Type)
		}
		obj := args[0]
		typeName, ok := args[1].Value.(string)
		if !ok {
			return nil, fmt.Errorf("TypeError: expected string for type name in set_type(), got %T", args[1].Value)
		}
		obj.Type = typeName
		return obj, nil
	})
	if err != nil {
		panic(err)
	}
	exitFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) > 1 {
			return nil, fmt.Errorf("RuntimeError: expected at most 1 argument for exit(), got %d", len(args))
		}
		exitCode := 0
		if len(args) == 1 {
			numObj, err := ToNumber(args[0])
			if err != nil {
				return nil, fmt.Errorf("RuntimeError: error converting argument to number for exit(): %v", err)
			}
			exitCode = int(numObj.Value.(float64))
		}
		os.Exit(exitCode)
		return nil, nil
	})
	if err != nil {
		panic(err)
	}
	stdoutObj := &MeowppObject{
		Type:  "special",
		Value: "<stdout>",
		Magic: map[string]ObjMagic{
			"op<<": func(self Obj, args []Obj) (Obj, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("RuntimeError: expected 1 argument for << operator, got %d", len(args))
				}
				stringify, err := ToString(args[0])
				if err != nil {
					return nil, fmt.Errorf("RuntimeError: error converting argument to string for stdout << operator: %v", err)
				}
				fmt.Print(stringify.Value)
				return self, nil
			},
			"string": func(self Obj, args []Obj) (Obj, error) {
				return NewString("<stdout>")
			},
		},
	}
	stdinObj := &MeowppObject{
		Type:  "special",
		Value: "<stdin>",
		Magic: map[string]ObjMagic{
			"op>>": func(self Obj, args []Obj) (Obj, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("RuntimeError: expected 1 argument for >> operator, got %d", len(args))
				}
				reader := liner.NewLiner()
				defer reader.Close()
				input, err := reader.Prompt("")
				if err != nil {
					return nil, fmt.Errorf("RuntimeError: error reading from stdin: %v", err)
				}
				res, err := NewString(input)
				if err != nil {
					return nil, fmt.Errorf("RuntimeError: error creating string from stdin input: %v", err)
				}
				CopyTo(args[0], res)
				return self, nil
			},
			"string": func(self Obj, args []Obj) (Obj, error) {
				return NewString("<stdin>")
			},
		},
	}
	stderrObj := &MeowppObject{
		Type:  "special",
		Value: "<stderr>",
		Magic: map[string]ObjMagic{
			"op<<": func(self Obj, args []Obj) (Obj, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("RuntimeError: expected 1 argument for << operator, got %d", len(args))
				}
				stringify, err := ToString(args[0])
				if err != nil {
					return nil, fmt.Errorf("RuntimeError: error converting argument to string for stderr << operator: %v", err)
				}
				fmt.Fprint(os.Stderr, stringify.Value)
				return self, nil
			},
			"string": func(self Obj, args []Obj) (Obj, error) {
				return NewString("<stderr>")
			},
		},
	}
	BUILTIN_VARS["stdout"] = stdoutObj
	BUILTIN_VARS["stdin"] = stdinObj
	BUILTIN_VARS["stderr"] = stderrObj
	BUILTIN_VARS["number"] = numberFunc
	BUILTIN_VARS["string"] = stringFunc
	BUILTIN_VARS["array"] = arrayFunc
	BUILTIN_VARS["map"] = mapFunc
	BUILTIN_VARS["len"] = lenFunc
	BUILTIN_VARS["type"] = typeFunc
	BUILTIN_VARS["set_magic"] = setMagicFunc
	BUILTIN_VARS["set_type"] = setTypeFunc
	BUILTIN_VARS["exit"] = exitFunc
}

func InitBuiltinModules() {  // Type: subruntime
	// builtins module
	builtinsModuleVars := make(map[string]Obj, len(BUILTIN_VARS))
	for name, value := range BUILTIN_VARS {
		builtinsModuleVars[fmt.Sprintf("%s@0", name)] = value
	}
	builtinsModule, err := NewSubRuntime(
		NewRuntime(
			builtinsModuleVars,
			[]map[string]any{},
		),
	)
	if err != nil {
		panic(err)
	}

	// unicode module
	unicodeUpperFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for unicode.upper, got %d", len(args))
		}
		if args[0].Type != "string" {
			return nil, fmt.Errorf("TypeError: expected string argument for unicode.upper, got %s", args[0].Type)
		}
		strVal, ok := args[0].Value.(string)
		if !ok {
			return nil, fmt.Errorf("TypeError: expected string argument for unicode.upper, got %T", args[0].Value)
		}
		return NewString(strings.ToUpper(strVal))
	})
	if err != nil {
		panic(err)
	}
	unicodeLowerFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for unicode.lower, got %d", len(args))
		}
		if args[0].Type != "string" {
			return nil, fmt.Errorf("TypeError: expected string argument for unicode.lower, got %s", args[0].Type)
		}
		strVal, ok := args[0].Value.(string)
		if !ok {
			return nil, fmt.Errorf("TypeError: expected string argument for unicode.lower, got %T", args[0].Value)
		}
		return NewString(strings.ToLower(strVal))
	})
	if err != nil {
		panic(err)
	}
	unicodeOrdsFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for unicode.ord, got %d", len(args))
		}
		if args[0].Type != "string" {
			return nil, fmt.Errorf("TypeError: expected string argument for unicode.ord, got %s", args[0].Type)
		}
		strVal, ok := args[0].Value.(string)
		if !ok {
			return nil, fmt.Errorf("TypeError: expected string argument for unicode.ord, got %T", args[0].Value)
		}
		resultList := []Obj{}
		for _, char := range strVal {
			charVal, err := NewNumber(float64(char))
			if err != nil {
				return nil, err
			}
			resultList = append(resultList, charVal)
		}
		return NewArray(resultList)
	})
	if err != nil {
		panic(err)
	}
	unicodeChrsFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for unicode.chr, got %d", len(args))
		}
		if args[0].Type != "array" {
			return nil, fmt.Errorf("TypeError: expected array argument for unicode.chrs, got %s", args[0].Type)
		}
		arrVal, ok := args[0].Value.([]Obj)
		if !ok {
			return nil, fmt.Errorf("TypeError: expected array argument for unicode.chrs, got %T", args[0].Value)
		}
		var sb strings.Builder
		for _, item := range arrVal {
			if item.Type != "number" {
				return nil, fmt.Errorf("TypeError: expected array of numbers for unicode.chrs, got an array containing type %s", item.Type)
			}
			numVal, ok := item.Value.(float64)
			if !ok {
				return nil, fmt.Errorf("TypeError: expected array of numbers for unicode.chrs, got an array containing type %T", item.Value)
			}
			if numVal < 0 || numVal > math.MaxInt32 {
				return nil, fmt.Errorf("RuntimeError: number out of valid Unicode code point range in unicode.chrs: %f", numVal)
			}
			sb.WriteRune(rune(numVal))
		}
		return NewString(sb.String())
	})
	if err != nil {
		panic(err)
	}
	unicodeModule, err := NewSubRuntime(
		NewRuntime(
			map[string]Obj{
				"upper@0": unicodeUpperFunc,
				"lower@0": unicodeLowerFunc,
				"ords@0": unicodeOrdsFunc,
				"chrs@0": unicodeChrsFunc,
			},
			[]map[string]any{},
		),
	)
	if err != nil {
		panic(err)
	}

	// fileio module
	fileIoWriteFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("RuntimeError: expected 2 arguments for fileio.write, got %d", len(args))
		}
		if args[0].Type != "string" {
			return nil, fmt.Errorf("TypeError: expected string argument for fileio.write filename, got %s", args[0].Type)
		}
		if args[1].Type != "array" {
			return nil, fmt.Errorf("TypeError: expected array of number argument for fileio.write content, got %s", args[1].Type)
		}
		filenameVal, ok := args[0].Value.(string)
		if !ok {
			return nil, fmt.Errorf("TypeError: expected string argument for fileio.write filename, got %T", args[0].Value)
		}
		contentArr, ok := args[1].Value.([]Obj)
		if !ok {
			return nil, fmt.Errorf("TypeError: expected array of number argument for fileio.write content, got %T", args[1].Value)
		}
		for _, item := range contentArr {
			if item.Type != "number" {
				return nil, fmt.Errorf("TypeError: expected array of number argument for fileio.write content, got an array containing type %s", item.Type)
			}
			numVal, ok := item.Value.(float64)
			if !ok {
				return nil, fmt.Errorf("TypeError: expected array of number argument for fileio.write content, got an array containing type %T", item.Value)
			}
			if !IsInteger(item) {
				return nil, fmt.Errorf("ValueError: non-integer number in array argument for fileio.write content: %f", numVal)
			}
			if numVal < 0 || numVal > math.MaxInt32 {
				return nil, fmt.Errorf("RuntimeError: number out of valid Unicode code point range in fileio.write content: %f", numVal)
			}
		}
		var bytesContent []byte
		for _, item := range contentArr {
			numVal := item.Value.(float64)
			bytesContent = append(bytesContent, byte(numVal))
		}
		err := os.WriteFile(filenameVal, bytesContent, 0644)
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error writing to file in fileio.write: %v", err)
		}
		return nil, nil
	})
	if err != nil {
		panic(err)
	}
	fileIoReadFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for fileio.read, got %d", len(args))
		}
		if args[0].Type != "string" {
			return nil, fmt.Errorf("TypeError: expected string argument for fileio.read filename, got %s", args[0].Type)
		}
		filenameVal, ok := args[0].Value.(string)
		if !ok {
			return nil, fmt.Errorf("TypeError: expected string argument for fileio.read filename, got %T", args[0].Value)
		}
		if _, err := os.Stat(filenameVal); os.IsNotExist(err) {
			return nil, fmt.Errorf("fileio.FileNotFoundError: file not found in fileio.read: %s", filenameVal)
		}
		bytesContent, err := os.ReadFile(filenameVal)
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error reading from file in fileio.read: %v", err)
		}
		contentArr := make([]Obj, len(bytesContent))
		for i, b := range bytesContent {
			numVal, err := NewNumber(float64(b))
			if err != nil {
				return nil, fmt.Errorf("RuntimeError: error converting byte to number in fileio.read content: %v", err)
			}
			contentArr[i] = numVal
		}
		return NewArray(contentArr)
	})
	if err != nil {
		panic(err)
	}
	fileIoExistsFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for fileio.exists, got %d", len(args))
		}
		if args[0].Type != "string" {
			return nil, fmt.Errorf("TypeError: expected string argument for fileio.exists filename, got %s", args[0].Type)
		}
		filenameVal, ok := args[0].Value.(string)
		if !ok {
			return nil, fmt.Errorf("TypeError: expected string argument for fileio.exists filename, got %T", args[0].Value)
		}
		if _, err := os.Stat(filenameVal); os.IsNotExist(err) {
			return NewBool(false)
		}
		return NewBool(true)
	})
	if err != nil {
		panic(err)
	}
	fileIoModule, err := NewSubRuntime(
		NewRuntime(
			map[string]Obj{
				"write@0": fileIoWriteFunc,
				"read@0": fileIoReadFunc,
				"exists@0": fileIoExistsFunc,
			},
			[]map[string]any{},
		),
	)
	if err != nil {
		panic(err)
	}

	// memory module
	memoryRawFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for memory.raw, got %d", len(args))
		}
		obj := args[0]
		// return an array of numbers representing the raw bytes of the object in memory
		var bytes []byte
		objPtr := unsafe.Pointer(&obj)
		objSize := unsafe.Sizeof(obj)
		for i := uintptr(0); i < objSize; i++ {
			byteVal := *(*byte)(unsafe.Pointer(uintptr(objPtr) + i))
			bytes = append(bytes, byteVal)
		}
		numArr := make([]Obj, len(bytes))
		for i, b := range bytes {
			numVal, err := NewNumber(float64(b))
			if err != nil {
				return nil, fmt.Errorf("RuntimeError: error converting byte to number in memory.raw: %v", err)
			}
			numArr[i] = numVal
		}
		return NewArray(numArr)
	})
	if err != nil {
		panic(err)
	}
	memorySizeFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for memory.size, got %d", len(args))
		}
		obj := args[0]
		objSize := unsafe.Sizeof(obj)
		return NewNumber(float64(objSize))
	})
	if err != nil {
		panic(err)
	}
	memoryUsageFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		var m goruntime.MemStats
		goruntime.ReadMemStats(&m)
		return NewNumber(float64(m.Alloc))
	})
	if err != nil {
		panic(err)
	}
	memoryModule, err := NewSubRuntime(
		NewRuntime(
			map[string]Obj{
				"raw@0": memoryRawFunc,
				"size@0": memorySizeFunc,
				"usage@0": memoryUsageFunc,
			},
			[]map[string]any{},
		),
	)
	if err != nil {
		panic(err)
	}

	// format module
	formatSprintfFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("RuntimeError: expected at least 1 argument for format.sprintf, got %d", len(args))
		}
		formatStrObj := args[0]
		if formatStrObj.Type != "string" {
			return nil, fmt.Errorf("TypeError: expected string argument for format.sprintf format string, got %s", formatStrObj.Type)
		}
		argsList := args[1:]
		argsArray, err := NewArray(argsList)
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error creating array of arguments for format.sprintf: %v", err)
		}
		formatted, err := Sprintf(formatStrObj, argsArray)
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error in format.sprintf: %v", err)
		}
		return NewString(formatted)
	})
	if err != nil {
		panic(err)
	}
	formatSscanfFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("RuntimeError: expected at least 2 arguments for format.sscanf, got %d", len(args))
		}
		inputStrObj := args[0]
		formatStrObj := args[1]
		if inputStrObj.Type != "string" {
			return nil, fmt.Errorf("TypeError: expected string argument for format.sscanf input string, got %s", inputStrObj.Type)
		}
		if formatStrObj.Type != "string" {
			return nil, fmt.Errorf("TypeError: expected string argument for format.sscanf format string, got %s", formatStrObj.Type)
		}
		scanned, err := Sscanf(inputStrObj, formatStrObj)
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error in format.sscanf: %v", err)
		}
		return scanned, nil
	})
	if err != nil {
		panic(err)
	}
	formatModule, err := NewSubRuntime(
		NewRuntime(
			map[string]Obj{
				"sprintf@0": formatSprintfFunc,
				"sscanf@0": formatSscanfFunc,
			},
			[]map[string]any{},
		),
	)
	if err != nil {
		panic(err)
	}

	// stdio module
	stdIoPrintFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		for _, arg := range args {
			stringify, err := ToString(arg)
			if err != nil {
				return nil, fmt.Errorf("RuntimeError: error converting argument to string for stdio.print: %v", err)
			}
			fmt.Print(stringify.Value)
		}
		return nil, nil
	})
	if err != nil {
		panic(err)
	}
	stdIoPrintlnFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		for _, arg := range args {
			stringify, err := ToString(arg)
			if err != nil {
				return nil, fmt.Errorf("RuntimeError: error converting argument to string for stdio.println: %v", err)
			}
			fmt.Print(stringify.Value)
		}
		fmt.Println()
		return nil, nil
	})
	if err != nil {
		panic(err)
	}
	stdIoInputFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) > 1 {
			return nil, fmt.Errorf("RuntimeError: expected at most 1 argument for stdio.input, got %d", len(args))
		}
		prompt := ""
		if len(args) == 1 {
			stringify, err := ToString(args[0])
			if err != nil {
				return nil, fmt.Errorf("RuntimeError: error converting argument to string for stdio.input prompt: %v", err)
			}
			prompt = stringify.Value.(string)
		}
		reader := liner.NewLiner()
		defer reader.Close()
		input, err := reader.Prompt(prompt)
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error reading input in stdio.input: %v", err)
		}
		return NewString(input)
	})
	if err != nil {
		panic(err)
	}
	stdIoModule, err := NewSubRuntime(
		NewRuntime(
			map[string]Obj{
				"print@0": stdIoPrintFunc,
				"println@0": stdIoPrintlnFunc,
				"input@0": stdIoInputFunc,
			},
			[]map[string]any{},
		),
	)
	if err != nil {
		panic(err)
	}

	// math.random module
	randomRandomFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("RuntimeError: expected 0 arguments for math.random.random, got %d", len(args))
		}
		randomVal := rand.Float64()
		return NewNumber(randomVal)
	})
	if err != nil {
		panic(err)
	}
	randomUniformFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("RuntimeError: expected 2 arguments for math.random.uniform, got %d", len(args))
		}
		minNum, err := ToNumber(args[0])
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error converting first argument to number for math.random.uniform: %v", err)
		}
		maxNum, err := ToNumber(args[1])
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error converting second argument to number for math.random.uniform: %v", err)
		}
		randomVal := minNum.Value.(float64) + rand.Float64()*(maxNum.Value.(float64)-minNum.Value.(float64))
		return NewNumber(randomVal)
	})
	if err != nil {
		panic(err)
	}
	randomIntFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("RuntimeError: expected 2 arguments for math.random.int, got %d", len(args))
		}
		minNum, err := ToNumber(args[0])
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error converting first argument to number for math.random.int: %v", err)
		}
		maxNum, err := ToNumber(args[1])
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error converting second argument to number for math.random.int: %v", err)
		}
		minInt := int(minNum.Value.(float64))
		maxInt := int(maxNum.Value.(float64))
		if minInt > maxInt {
			minInt, maxInt = maxInt, minInt  // swap to allow reversed range
		}
		randomVal := rand.Intn(maxInt-minInt+1) + minInt
		return NewNumber(float64(randomVal))
	})
	if err != nil {
		panic(err)
	}
	randomGaussFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("RuntimeError: expected 2 arguments for math.random.gauss, got %d", len(args))
		}
		meanNum, err := ToNumber(args[0])
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error converting first argument to number for math.random.gauss: %v", err)
		}
		stddevNum, err := ToNumber(args[1])
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error converting second argument to number for math.random.gauss: %v", err)
		}
		randomVal := rand.NormFloat64()*stddevNum.Value.(float64) + meanNum.Value.(float64)
		return NewNumber(randomVal)
	})
	if err != nil {
		panic(err)
	}
	randomModule, err := NewSubRuntime(
		NewRuntime(
			map[string]Obj{
				"random@0": randomRandomFunc,
				"uniform@0": randomUniformFunc,
				"int@0": randomIntFunc,
				"gauss@0": randomGaussFunc,
			},
			[]map[string]any{},
		),
	)
	if err != nil {
		panic(err)
	}

	// math module
	mathSinFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for math.sin, got %d", len(args))
		}
		numArg, err := ToNumber(args[0])
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error converting argument to number for math.sin: %v", err)
		}
		sinVal := math.Sin(numArg.Value.(float64))
		return NewNumber(sinVal)
	})
	if err != nil {
		panic(err)
	}
	mathCosFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for math.cos, got %d", len(args))
		}
		numArg, err := ToNumber(args[0])
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error converting argument to number for math.cos: %v", err)
		}
		cosVal := math.Cos(numArg.Value.(float64))
		return NewNumber(cosVal)
	})
	if err != nil {
		panic(err)
	}
	mathTanFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for math.tan, got %d", len(args))
		}
		numArg, err := ToNumber(args[0])
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error converting argument to number for math.tan: %v", err)
		}
		tanVal := math.Tan(numArg.Value.(float64))
		return NewNumber(tanVal)
	})
	if err != nil {
		panic(err)
	}
	mathAsinFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for math.asin, got %d", len(args))
		}
		numArg, err := ToNumber(args[0])
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error converting argument to number for math.asin: %v", err)
		}
		asinVal := math.Asin(numArg.Value.(float64))
		return NewNumber(asinVal)
	})
	if err != nil {
		panic(err)
	}
	mathAcosFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for math.acos, got %d", len(args))
		}
		numArg, err := ToNumber(args[0])
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error converting argument to number for math.acos: %v", err)
		}
		acosVal := math.Acos(numArg.Value.(float64))
		return NewNumber(acosVal)
	})
	if err != nil {
		panic(err)
	}
	mathAtanFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for math.atan, got %d", len(args))
		}
		numArg, err := ToNumber(args[0])
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error converting argument to number for math.atan: %v", err)
		}
		atanVal := math.Atan(numArg.Value.(float64))
		return NewNumber(atanVal)
	})
	if err != nil {
		panic(err)
	}
	mathPi, err := NewNumber(float64(math.Pi))
	if err != nil {
		panic(err)
	}
	mathE, err := NewNumber(float64(math.E))
	if err != nil {
		panic(err)
	}
	mathSqrtFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for math.sqrt, got %d", len(args))
		}
		numArg, err := ToNumber(args[0])
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error converting argument to number for math.sqrt: %v", err)
		}
		sqrtVal := math.Sqrt(numArg.Value.(float64))
		return NewNumber(sqrtVal)
	})
	if err != nil {
		panic(err)
	}
	mathLogFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("RuntimeError: expected 2 arguments for math.log, got %d", len(args))
		}
		numArg, err := ToNumber(args[0])
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error converting first argument to number for math.log: %v", err)
		}
		baseArg, err := ToNumber(args[1])
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error converting second argument to number for math.log: %v", err)
		}
		logVal := math.Log(numArg.Value.(float64)) / math.Log(baseArg.Value.(float64))
		return NewNumber(logVal)
	})
	if err != nil {
		panic(err)
	}
	mathLnFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for math.ln, got %d", len(args))
		}
		numArg, err := ToNumber(args[0])
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error converting argument to number for math.ln: %v", err)
		}
		lnVal := math.Log(numArg.Value.(float64))
		return NewNumber(lnVal)
	})
	if err != nil {
		panic(err)
	}
	mathLog10Func, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for math.log10, got %d", len(args))
		}
		numArg, err := ToNumber(args[0])
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error converting argument to number for math.log10: %v", err)
		}
		log10Val := math.Log10(numArg.Value.(float64))
		return NewNumber(log10Val)
	})
	if err != nil {
		panic(err)
	}
	mathLog2Func, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for math.log2, got %d", len(args))
		}
		numArg, err := ToNumber(args[0])
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error converting argument to number for math.log2: %v", err)
		}
		log2Val := math.Log2(numArg.Value.(float64))
		return NewNumber(log2Val)
	})
	if err != nil {
		panic(err)
	}
	mathModule, err := NewSubRuntime(
		NewRuntime(
			map[string]Obj{
				"random@0": randomModule,
				"sin@0": mathSinFunc,
				"cos@0": mathCosFunc,
				"tan@0": mathTanFunc,
				"asin@0": mathAsinFunc,
				"acos@0": mathAcosFunc,
				"atan@0": mathAtanFunc,
				"pi@0": mathPi,
				"e@0": mathE,
				"sqrt@0": mathSqrtFunc,
				"log@0": mathLogFunc,
				"ln@0": mathLnFunc,
				"log10@0": mathLog10Func,
				"log2@0": mathLog2Func,
			},
			[]map[string]any{},
		),
	)
	if err != nil {
		panic(err)
	}

	BUILTIN_MODULES["builtins"] = builtinsModule
	BUILTIN_MODULES["unicode"]  = unicodeModule
	BUILTIN_MODULES["fileio"]   = fileIoModule
	BUILTIN_MODULES["memory"]   = memoryModule
	BUILTIN_MODULES["format"]   = formatModule
	BUILTIN_MODULES["stdio"]    = stdIoModule
	BUILTIN_MODULES["math"]     = mathModule
}
