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
	mathFloorFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for math.floor, got %d", len(args))
		}
		numArg, err := ToNumber(args[0])
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error converting argument to number for math.floor: %v", err)
		}
		floorVal := math.Floor(numArg.Value.(float64))
		return NewNumber(floorVal)
	})
	if err != nil {
		panic(err)
	}
	mathCeilFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for math.ceil, got %d", len(args))
		}
		numArg, err := ToNumber(args[0])
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error converting argument to number for math.ceil: %v", err)
		}
		ceilVal := math.Ceil(numArg.Value.(float64))
		return NewNumber(ceilVal)
	})
	if err != nil {
		panic(err)
	}
	mathRoundFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for math.round, got %d", len(args))
		}
		numArg, err := ToNumber(args[0])
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error converting argument to number for math.round: %v", err)
		}
		roundVal := math.Round(numArg.Value.(float64))
		return NewNumber(roundVal)
	})
	if err != nil {
		panic(err)
	}
	mathTruncFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for math.trunc, got %d", len(args))
		}
		numArg, err := ToNumber(args[0])
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error converting argument to number for math.trunc: %v", err)
		}
		truncVal := math.Trunc(numArg.Value.(float64))
		return NewNumber(truncVal)
	})
	if err != nil {
		panic(err)
	}
	mathAbsFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for math.abs, got %d", len(args))
		}
		numArg, err := ToNumber(args[0])
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error converting argument to number for math.abs: %v", err)
		}
		absVal := math.Abs(numArg.Value.(float64))
		return NewNumber(absVal)
	})
	if err != nil {
		panic(err)
	}
	mathMinFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("RuntimeError: expected at least 1 argument for math.min, got %d", len(args))
		}
		minVal := math.Inf(1)
		for _, arg := range args {
			numArg, err := ToNumber(arg)
			if err != nil {
				return nil, fmt.Errorf("RuntimeError: error converting argument to number for math.min: %v", err)
			}
			if numArg.Value.(float64) < minVal {
				minVal = numArg.Value.(float64)
			}
		}
		return NewNumber(minVal)
	})
	if err != nil {
		panic(err)
	}
	mathMaxFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("RuntimeError: expected at least 1 argument for math.max, got %d", len(args))
		}
		maxVal := math.Inf(-1)
		for _, arg := range args {
			numArg, err := ToNumber(arg)
			if err != nil {
				return nil, fmt.Errorf("RuntimeError: error converting argument to number for math.max: %v", err)
			}
			if numArg.Value.(float64) > maxVal {
				maxVal = numArg.Value.(float64)
			}
		}
		return NewNumber(maxVal)
	})
	if err != nil {
		panic(err)
	}
	mathExpFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for math.exp, got %d", len(args))
		}
		numArg, err := ToNumber(args[0])
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error converting argument to number for math.exp: %v", err)
		}
		expVal := math.Exp(numArg.Value.(float64))
		return NewNumber(expVal)
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
				"floor@0": mathFloorFunc,
				"ceil@0": mathCeilFunc,
				"round@0": mathRoundFunc,
				"trunc@0": mathTruncFunc,
				"abs@0": mathAbsFunc,
				"min@0": mathMinFunc,
				"max@0": mathMaxFunc,
				"exp@0": mathExpFunc,
			},
			[]map[string]any{},
		),
	)
	if err != nil {
		panic(err)
	}

	// makeclass module
	makeClassMagicMagics := map[string]ObjMagic{
		"call": func(self Obj, args []Obj) (Obj, error) {
			magic, ok := self.Value.(ObjMagic)
			if !ok {
				return nil, fmt.Errorf("RuntimeError: invalid self object for call magic, expected ObjMagic, got %T", self.Value)
			}
			return magic(self, args)
		},
		"repr": func(self Obj, args []Obj) (Obj, error) {
			magic, ok := self.Value.(ObjMagic)
			if !ok {
				return nil, fmt.Errorf("RuntimeError: invalid self object for repr magic, expected ObjMagic, got %T", self.Value)
			}
			return NewString(fmt.Sprintf("<magic object with call behavior %p>", magic))
		},
	}

	makeClassMagicsMagics := map[string]ObjMagic{
		"set_item": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("RuntimeError: expected 2 arguments for set_item, got %d", len(args))
			}
			key := args[0]
			newValue := args[1]
			magics, ok := self.Value.(map[string]ObjMagic)
			if !ok {
				return nil, fmt.Errorf("RuntimeError: invalid self object for set_item magic, expected map[string]ObjMagic, got %T", self.Value)
			}
			if newValue.Type != "magic" {
				return nil, fmt.Errorf("TypeError: expected magic object for new value in set_item magic, got %s", newValue.Type)
			}
			newMagic, ok := newValue.Value.(ObjMagic)
			if !ok {
				return nil, fmt.Errorf("TypeError: expected magic object for new value in set_item magic, got %T", newValue.Value)
			}
			magics[key.Value.(string)] = newMagic
			return nil, nil
		},
		"get_item": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for get_item, got %d", len(args))
			}
			key := args[0]
			if key.Type != "string" {
				return nil, fmt.Errorf("TypeError: expected string argument for key in get_item magic, got %s", key.Type)
			}
			magics, ok := self.Value.(map[string]ObjMagic)
			if !ok {
				return nil, fmt.Errorf("RuntimeError: invalid self object for get_item magic, expected map[string]ObjMagic, got %T", self.Value)
			}
			magic, ok := magics[key.Value.(string)]
			if !ok {
				return nil, fmt.Errorf("RuntimeError: no magic found for key in get_item magic: %s", key.Value.(string))
			}
			magicObj := &MeowppObject{
				Type: "magic",
				Value: magic,
				Magic: makeClassMagicMagics,
			}
			return magicObj, nil
		},
		"repr": func(self Obj, args []Obj) (Obj, error) {
			magics, ok := self.Value.(map[string]ObjMagic)
			if !ok {
				return nil, fmt.Errorf("RuntimeError: invalid self object for repr magic, expected map[string]ObjMagic, got %T", self.Value)
			}
			return NewString(fmt.Sprintf("<magics object with %d magics>", len(magics)))
		},
	}

	makeClassInfoMagics := map[string]ObjMagic{
		"get_item": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for get_item, got %d", len(args))
			}
			key := args[0]
			values, ok := self.Value.(map[string]Obj)
			if !ok {
				return nil, fmt.Errorf("RuntimeError: invalid self object for get_item magic, expected map[string]Obj, got %T", self.Value)
			}
			value, ok := values["value"]
			if !ok {
				return nil, fmt.Errorf("RuntimeError: invalid self object for get_item magic, expected Obj, got %T", self.Value)
			}
			switch key.Type {
			case "type":
				return NewString(value.Type)
			case "value":
				return value, nil
			case "magic":
				magicsObj := &MeowppObject{
					Type: "magics",
					Value: value.Magic,
					Magic: makeClassMagicsMagics,
				}
				return magicsObj, nil
			case "update":
				updateFunc := func(self Obj, args []Obj) (Obj, error) {
					if len(args) != 0 {
						return nil, fmt.Errorf("RuntimeError: expected 0 argument for update magic, got %d", len(args))
					}
					magics, ok := values["magic"].Value.(map[string]ObjMagic)
					if !ok {
						return nil, fmt.Errorf("RuntimeError: invalid magic object for update magic, expected map[string]ObjMagic, got %T", values["magic"].Value)
					}
					value.Magic = magics
					return nil, nil
				}
				return NewFunction(updateFunc)
			default:
				return nil, fmt.Errorf("RuntimeError: unsupported key for get_item: %s", key.Type)
			}
		},
		"set_item": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("RuntimeError: expected 2 arguments for set_item, got %d", len(args))
			}
			key := args[0]
			newValue := args[1]
			values, ok := self.Value.(map[string]Obj)
			value, ok := values["value"]
			if !ok {
				return nil, fmt.Errorf("RuntimeError: invalid self object for set_item magic, expected Obj, got %T", self.Value)
			}
			switch key.Type {
			case "type":
				value.Type = newValue.Value.(string)
				return nil, nil
			case "value":
				value.Value = newValue.Value
				return nil, nil
			case "magic":
				if newValue.Type != "magics" {
					return nil, fmt.Errorf("TypeError: expected magic object for magic key in set_item magic, got %s", newValue.Type)
				}
				values["magic"] = newValue
				magics, ok := newValue.Value.(map[string]ObjMagic)
				if !ok {
					return nil, fmt.Errorf("TypeError: expected magic object for magic key in set_item magic, got %T", newValue.Value)
				}
				value.Magic = magics
				return nil, nil
			default:
				return nil, fmt.Errorf("RuntimeError: unsupported key for set_item: %s", key.Type)
			}
		},
		"repr": func(self Obj, args []Obj) (Obj, error) {
			value, ok := self.Value.(Obj)
			if !ok {
				return nil, fmt.Errorf("RuntimeError: invalid self object for repr magic, expected Obj, got %T", self.Value)
			}
			representedValue, err := Repr(value)
			if err != nil {
				return nil, fmt.Errorf("RuntimeError: error in repr magic: %v", err)
			}
			return NewString(fmt.Sprintf("<info object with type '%s' and value %v>", value.Type, representedValue))
		},
	}

	makeClassGetInfoFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for makeclass.get_info, got %d", len(args))
		}
		obj := args[0]
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: error creating string for object type: %v", err)
		}
		infoObj := &MeowppObject{
			Type: "info",
			Value: map[string]Obj{
				"value": obj,
				"magic": &MeowppObject{
					Type: "magics",
					Value: obj.Magic,
					Magic: makeClassMagicsMagics,
				},
			},
			Magic: makeClassInfoMagics,
		}
		return infoObj, nil
	})
	if err != nil {
		panic(err)
	}

	makeClassMagicFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for makeclass.magic, got %d", len(args))
		}
		callMagicObj := args[0]
		if callMagicObj.Type != "function" {
			return nil, fmt.Errorf("TypeError: expected function argument for makeclass.magic call magic, got %s", callMagicObj.Type)
		}
		callMagicFunc, ok := callMagicObj.Value.(func([]Obj) (Obj, error))
		if !ok {
			return nil, fmt.Errorf("TypeError: expected function argument for makeclass.magic call magic, got %T", callMagicObj.Value)
		}
		newCallFunc := func(self Obj, args []Obj) (Obj, error) {
			allArgs := append([]Obj{self}, args...)
			return callMagicFunc(allArgs)
		}
		magicObj := &MeowppObject{
			Type: "magic",
			Value: newCallFunc,
			Magic: makeClassMagicMagics,
		}
		return magicObj, nil
	})
	if err != nil {
		panic(err)
	}

	makeClassModule, err := NewSubRuntime(
		NewRuntime(
			map[string]Obj{
				"get_info@0": makeClassGetInfoFunc,
				"Magic@0": makeClassMagicFunc,
			},
			[]map[string]any{},
		),
	)
	if err != nil {
		panic(err)
	}

	// array module
	arrayAppendFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("RuntimeError: expected 2 arguments for array.append, got %d", len(args))
		}
		arrayObj := args[0]
		elements := args[1:]
		if arrayObj.Type != "array" {
			return nil, fmt.Errorf("TypeError: expected array argument for array.append, got %s", arrayObj.Type)
		}
		array, ok := arrayObj.Value.([]Obj)
		if !ok {
			return nil, fmt.Errorf("RuntimeError: invalid array object for array.append, expected []Obj, got %T", arrayObj.Value)
		}
		newArray := append(array, elements...)
		arrayObj.Value = newArray
		return nil, nil
	})
	if err != nil {
		panic(err)
	}
	arrayPopFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("RuntimeError: expected 1 argument for array.pop, got %d", len(args))
		}
		arrayObj := args[0]
		if arrayObj.Type != "array" {
			return nil, fmt.Errorf("TypeError: expected array argument for array.pop, got %s", arrayObj.Type)
		}
		array, ok := arrayObj.Value.([]Obj)
		if !ok {
			return nil, fmt.Errorf("RuntimeError: invalid array object for array.pop, expected []Obj, got %T", arrayObj.Value)
		}
		if len(array) == 0 {
			return nil, fmt.Errorf("RuntimeError: cannot pop from empty array")
		}
		lastElement := array[len(array)-1]
		newArray := array[:len(array)-1]
		arrayObj.Value = newArray
		return lastElement, nil
	})
	if err != nil {
		panic(err)
	}
	arraySliceFunc, err := NewFunction(func(args []Obj) (Obj, error) {
		if len(args) < 3 || len(args) > 4 {
			return nil, fmt.Errorf("RuntimeError: expected 2-4 arguments for array.slice, got %d", len(args))
		}
		arrayObj := args[0]
		if arrayObj.Type != "array" {
			return nil, fmt.Errorf("TypeError: expected array argument for array.slice, got %s", arrayObj.Type)
		}
		array, ok := arrayObj.Value.([]Obj)
		if !ok {
			return nil, fmt.Errorf("RuntimeError: invalid array object for array.slice, expected []Obj, got %T", arrayObj.Value)
		}
		startIndex := 0
		endIndex := len(array)
		step := 1
		if len(args) >= 3 {
			if args[1] == nil {
				startIndex = 0
			} else {
				startNum, err := ToNumber(args[1])
				if err != nil {
					return nil, fmt.Errorf("RuntimeError: error converting start index to number for array.slice: %v", err)
				}
				startIndex = int(startNum.Value.(float64))
			}
			for startIndex < 0 {
				startIndex += len(array)
			}
			for startIndex >= len(array) {
				startIndex -= len(array)
			}
			if args[2] == nil {
				endIndex = len(array)
			} else {
				endNum, err := ToNumber(args[2])
				if err != nil {
					return nil, fmt.Errorf("RuntimeError: error converting end index to number for array.slice: %v", err)
				}
				endIndex = int(endNum.Value.(float64))
			}
			for endIndex < 0 {
				endIndex += len(array)
			}
			for endIndex >= len(array) {
				endIndex -= len(array)
			}
		}
		if len(args) == 4 {
			stepNum, err := ToNumber(args[3])
			if err != nil {
				return nil, fmt.Errorf("RuntimeError: error converting step to number for array.slice: %v", err)
			}
			step = int(stepNum.Value.(float64))
			if step == 0 {
				return nil, fmt.Errorf("RuntimeError: step cannot be zero in array.slice")
			}
			if step < 0 && startIndex < endIndex {
				return nil, fmt.Errorf("RuntimeError: step cannot be negative when start index is less than end index in array.slice")
			}
			if step > 0 && startIndex > endIndex {
				return nil, fmt.Errorf("RuntimeError: step cannot be positive when start index is greater than end index in array.slice")
			}
		}
		slicedArray := []Obj{}
		for i := startIndex; (step > 0 && i < endIndex) || (step < 0 && i > endIndex); i += step {
			slicedArray = append(slicedArray, array[i])
		}
		return NewArray(slicedArray)
	})
	arrayModule, err := NewSubRuntime(
		NewRuntime(
			map[string]Obj{
				"append@0": arrayAppendFunc,
				"pop@0": arrayPopFunc,
				"slice@0": arraySliceFunc,
			},
			[]map[string]any{},
		),
	)
	if err != nil {
		panic(err)
	}

	// register built-in modules
	BUILTIN_MODULES["builtins"]  = builtinsModule
	BUILTIN_MODULES["unicode"]   = unicodeModule
	BUILTIN_MODULES["fileio"]    = fileIoModule
	BUILTIN_MODULES["memory"]    = memoryModule
	BUILTIN_MODULES["format"]    = formatModule
	BUILTIN_MODULES["stdio"]     = stdIoModule
	BUILTIN_MODULES["math"]      = mathModule
	BUILTIN_MODULES["makeclass"] = makeClassModule
	BUILTIN_MODULES["array"]     = arrayModule
}
