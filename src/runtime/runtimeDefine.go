package runtime

import (
	"fmt"
)

type Runtime struct {
	Vars map[string]Obj
	Loops []*Loop
	Functions []*Function
	Scopes []int
	ScopeCounter int
	CmdIndex int
	Cmds []map[string]any
	Globals map[int][]string
	Nonlocals map[int][]string
	Labels map[string]int
	CurrentExc string
	ExcStack []string
}

type Loop struct {
	status string  // "running", "break", "continue"
}

type Function struct {
	status string  // "running", "return"
	value  Obj
}

func ToSliceOfMapStringAny(raw []any) ([]map[string]any, bool) {
	result := make([]map[string]any, len(raw))
	for i, item := range raw {
		mapped, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		result[i] = mapped
	}
	return result, true
}

func NewRuntime(vars map[string]Obj, cmds []map[string]any) Runtime {
	return Runtime{
		Vars: vars,
		Loops: []*Loop{},
		Functions: []*Function{},
		Scopes: []int{0},
		ScopeCounter: 1,
		Globals: make(map[int][]string),
		Nonlocals: make(map[int][]string),
		Labels: make(map[string]int),
		CurrentExc: "",
		ExcStack: []string{},
		Cmds: cmds,
	}
}

func ChunkRuntime(Cmds []map[string]any, RawRuntime Runtime) Runtime {
	vars := RawRuntime.Vars
	loops := RawRuntime.Loops
	functions := RawRuntime.Functions
	scopes := RawRuntime.Scopes
	scopeCounter := RawRuntime.ScopeCounter
	globals := RawRuntime.Globals
	nonlocals := RawRuntime.Nonlocals
	excStack := RawRuntime.ExcStack
	cmdIndex := 0

	return Runtime{
		Vars: vars,
		Loops: loops,
		Functions: functions,
		Scopes: scopes,
		ScopeCounter: scopeCounter,
		CmdIndex: cmdIndex,
		Cmds: Cmds,
		Globals: globals,
		Nonlocals: nonlocals,
		Labels: make(map[string]int),
		CurrentExc: "",
		ExcStack: excStack,
	}
}

func SyncRuntime(RawRuntime *Runtime, newRuntime Runtime) {
	RawRuntime.Vars = newRuntime.Vars
	RawRuntime.Loops = newRuntime.Loops
	RawRuntime.Functions = newRuntime.Functions
	RawRuntime.Scopes = newRuntime.Scopes
	RawRuntime.ScopeCounter = newRuntime.ScopeCounter
	RawRuntime.Globals = newRuntime.Globals
	RawRuntime.Nonlocals = newRuntime.Nonlocals
	RawRuntime.Labels = newRuntime.Labels
	RawRuntime.CurrentExc = newRuntime.CurrentExc
	RawRuntime.ExcStack = newRuntime.ExcStack
}

func (rt *Runtime) PushScope() {
	rt.Scopes = append(rt.Scopes, rt.ScopeCounter)
	rt.ScopeCounter++
}

func (rt *Runtime) PopScope() {
	if len(rt.Scopes) > 1 {
		rt.Scopes = rt.Scopes[:len(rt.Scopes)-1]
	}
}

func (rt *Runtime) CurrentScope() int {
	return rt.Scopes[len(rt.Scopes)-1]
}

func (rt *Runtime) GetVar(name string) (Obj, bool) {
	if globals, ok := rt.Globals[rt.CurrentScope()]; ok {
		for _, g := range globals {
			if g == name {
				val, exists := rt.Vars[fmt.Sprintf("%s@0", name)]
				return val, exists
			}
		}
	}
	if nonlocals, ok := rt.Nonlocals[rt.CurrentScope()]; ok {
		for _, n := range nonlocals {
			if n == name {
				for i := len(rt.Scopes) - 2; i >= 0; i-- {
					scopeID := rt.Scopes[i]
					key := fmt.Sprintf("%s@%d", name, scopeID)
					if val, ok := rt.Vars[key]; ok {
						return val, true
					}
				}
				return nil, false
			}
		}
	}
	for i := len(rt.Scopes) - 1; i >= 0; i-- {
		scopeID := rt.Scopes[i]
		key := fmt.Sprintf("%s@%d", name, scopeID)
		if val, ok := rt.Vars[key]; ok {
			return val, true
		}
	}
	if val, ok := BUILTIN_VARS[name]; ok {
		return val, true
	}
	return nil, false
}

func (rt *Runtime) SetVar(name string, value Obj) {
	isGlobal := false
	isNonlocal := false
	globals, ok := rt.Globals[rt.CurrentScope()]
	if ok {
		for _, g := range globals {
			if g == name {
				isGlobal = true
				break
			}
		}
	}
	nonlocals, ok := rt.Nonlocals[rt.CurrentScope()]
	if ok {
		for _, n := range nonlocals {
			if n == name {
				isNonlocal = true
				break
			}
		}
	}

	var scopeID int
	if isGlobal {
		scopeID = 0
	} else if isNonlocal {
		for i := len(rt.Scopes) - 2; i >= 0; i-- {
			scopeID = rt.Scopes[i]
			key := fmt.Sprintf("%s@%d", name, scopeID)
			if _, ok := rt.Vars[key]; ok {
				break
			}
		}
	} else {
		scopeID = rt.CurrentScope()
	}

	key := fmt.Sprintf("%s@%d", name, scopeID)
	rt.Vars[key] = value
}

func (rt *Runtime) DeleteVar(name string) error {
	if globals, ok := rt.Globals[rt.CurrentScope()]; ok {
		for _, g := range globals {
			if g == name {
				key := fmt.Sprintf("%s@0", name)
				if _, ok := rt.Vars[key]; ok {
					delete(rt.Vars, key)
					return nil
				}
				return fmt.Errorf("NameError: name '%s' is not defined", name)
			}
		}
	}
	if nonlocals, ok := rt.Nonlocals[rt.CurrentScope()]; ok {
		for _, n := range nonlocals {
			if n == name {
				for i := len(rt.Scopes) - 2; i >= 0; i-- {
					scopeID := rt.Scopes[i]
					key := fmt.Sprintf("%s@%d", name, scopeID)
					if _, ok := rt.Vars[key]; ok {
						delete(rt.Vars, key)
						return nil
					}
				}
				return fmt.Errorf("NameError: name '%s' is not defined", name)
			}
		}
	}
	for i := len(rt.Scopes) - 1; i >= 0; i-- {
		scopeID := rt.Scopes[i]
		key := fmt.Sprintf("%s@%d", name, scopeID)
		if _, ok := rt.Vars[key]; ok {
			delete(rt.Vars, key)
			return nil
		}
	}
	return fmt.Errorf("NameError: name '%s' is not defined", name)
}

func (rt *Runtime) PushLoop(loop *Loop) {
	rt.Loops = append(rt.Loops, loop)
}

func (rt *Runtime) RemoveLoop(loop *Loop) {
	for i := len(rt.Loops) - 1; i >= 0; i-- {
		if rt.Loops[i] == loop {
			rt.Loops = rt.Loops[:i]
			break
		}
	}
}

func (rt *Runtime) PushFunction(function *Function) {
	rt.Functions = append(rt.Functions, function)
}

func (rt *Runtime) RemoveFunction(function *Function) {
	for i := len(rt.Functions) - 1; i >= 0; i-- {
		if rt.Functions[i] == function {
			rt.Functions = rt.Functions[:i]
			break
		}
	}
}

func (rt *Runtime) RaiseError(err error) {
	errMsg := "Traceback (most recent call last):\n"
	for _, exc := range rt.ExcStack {
		errMsg += exc + "\n"
	}
	if rt.CurrentExc != "" {
		errMsg += fmt.Sprintf("%s\n", rt.CurrentExc)
	}
	errMsg += fmt.Sprintf("\033[1;31m%s\033[0m\n", err.Error())
	fmt.Print(errMsg)
}

func (rt *Runtime) DefineFunction(args []string, body []map[string]any, infArgs bool) func([]Obj) (Obj, error) {
	currentScopes := rt.Scopes
	return func(callArgs []Obj) (Obj, error) {
		if !infArgs && len(callArgs) != len(args) {
			return nil, fmt.Errorf("TypeError: expected %d arguments, got %d", len(args), len(callArgs))
		} else if infArgs && len(callArgs) < len(args) {
			return nil, fmt.Errorf("TypeError: expected at least %d arguments, got %d", len(args), len(callArgs))
		}
		newRuntime := ChunkRuntime(body, *rt)
		newRuntime.Scopes = currentScopes
		newRuntime.PushScope()
		if infArgs {
			for i, argName := range args[:len(args)-1] {
				newRuntime.SetVar(argName, callArgs[i])
			}
			argName := args[len(args)-1]
			finalArg, err := NewArray(callArgs[len(args)-1:])
			if err != nil {
				return nil, err
			}
			newRuntime.SetVar(argName, finalArg)
		} else {
			for i, argName := range args {
				newRuntime.SetVar(argName, callArgs[i])
			}
		}
		newRuntime.ExcStack = append(newRuntime.ExcStack, rt.CurrentExc)
		// Execute the function body
		err := newRuntime.Run()
		if err != nil {
			return nil, err
		}
		SyncRuntime(rt, newRuntime)
		if len(newRuntime.Functions) > 0 {
			currentFunc := newRuntime.Functions[len(newRuntime.Functions)-1]
			if currentFunc.status == "return" {
				return currentFunc.value, nil
			}
		}
		return NewNull(), nil
	}
}

func (rt *Runtime) GetValue(raw any) (Obj, error) {
	switch v := raw.(type) {
	case int:
		return NewNumber(float64(v))
	case float64:
		return NewNumber(v)
	case string:
		return NewString(v)
	case bool:
		return NewBool(v)
	case []any:
		arr := make([]Obj, len(v))
		for i, item := range v {
			itemVal, err := rt.GetValue(item)
			if err != nil {
				return nil, err
			}
			arr[i] = itemVal
		}
		return NewArray(arr)
	case nil:
		return NewNull(), nil
	case map[string]any:
		return rt.GetCmdValue(v)
	default:
		return nil, fmt.Errorf("RuntimeError: unsupported value type: %T", raw)
	}
}

func (rt *Runtime) GetCmdValue(cmd map[string]any) (Obj, error) {
	cmdType, ok := cmd["cmd"].(string)
	if !ok {
		return nil, fmt.Errorf("RuntimeError: invalid value object, missing 'cmd' key or 'cmd' is not a string")
	}
	switch cmdType {
	case "var":  // Get variable value
		name, ok := cmd["name"].(string)
		if !ok {
			return nil, fmt.Errorf("RuntimeError: invalid var command, missing 'name' key or 'name' is not a string")
		}
		value, exists := rt.GetVar(name)
		if !exists {
			return nil, fmt.Errorf("NameError: name '%s' is not defined", name)
		}
		return value, nil
	case "get_item":  // Get item from array or map
		obj, ok := cmd["object"]
		if !ok {
			return nil, fmt.Errorf("RuntimeError: invalid get_item command, missing 'object' key")
		}
		key, ok := cmd["key"]
		if !ok {
			return nil, fmt.Errorf("RuntimeError: invalid get_item command, missing 'key' key")
		}
		objVal, err := rt.GetValue(obj)
		if err != nil {
			return nil, err
		}
		keyVal, err := rt.GetValue(key)
		if err != nil {
			return nil, err
		}
		getItemMagic, ok := objVal.Magic["get_item"]
		if !ok {
			return nil, fmt.Errorf("TypeError: object of type '%s' does not support get_item operation", objVal.Type)
		}
		return getItemMagic(objVal, []Obj{keyVal})
	case "unary":  // Unary operation
		operator, ok := cmd["operator"].(string)
		if !ok {
			return nil, fmt.Errorf("RuntimeError: invalid unary command, missing 'operator' key or 'operator' is not a string")
		}
		operand, ok := cmd["operand"]
		if !ok {
			return nil, fmt.Errorf("RuntimeError: invalid unary command, missing 'operand' key")
		}
		operandVal, err := rt.GetValue(operand)
		if err != nil {
			return nil, err
		}
		switch operator {
		case "+":
			posMagic, ok := operandVal.Magic["pos"]
			if !ok {
				return nil, fmt.Errorf("TypeError: object of type '%s' does not support positive unary operation", operandVal.Type)
			}
			return posMagic(operandVal, nil)
		case "-":
			negMagic, ok := operandVal.Magic["neg"]
			if !ok {
				return nil, fmt.Errorf("TypeError: object of type '%s' does not support negation", operandVal.Type)
			}
			return negMagic(operandVal, nil)
		case "!":
			if operandVal.Type != "bool" {
				return nil, fmt.Errorf("TypeError: logical NOT operator requires a boolean operand, got '%s'", operandVal.Type)
			}
			res, ok := operandVal.Value.(bool)
			if !ok {
				return nil, fmt.Errorf("RuntimeError: invalid boolean value for logical NOT operator")
			}
			return NewBool(!res)
		default:
			return nil, fmt.Errorf("RuntimeError: unknown unary operator '%s'", operator)
		}
	case "operation":  // Binary operation
		operator, ok := cmd["operator"].(string)
		if !ok {
			return nil, fmt.Errorf("RuntimeError: invalid operation command, missing 'operator' key or 'operator' is not a string")
		}
		left, ok := cmd["left"]
		if !ok {
			return nil, fmt.Errorf("RuntimeError: invalid operation command, missing 'left' key")
		}
		right, ok := cmd["right"]
		if !ok {
			return nil, fmt.Errorf("RuntimeError: invalid operation command, missing 'right' key")
		}
		leftVal, err := rt.GetValue(left)
		if err != nil {
			return nil, err
		}
		rightVal, err := rt.GetValue(right)
		if err != nil {
			return nil, err
		}
		var magicName string
		switch operator {
		case "+":
			magicName = "op+"
		case "-":
			magicName = "op-"
		case "*":
			magicName = "op*"
		case "/":
			magicName = "op/"
		case "%":
			magicName = "op%"
		case "^":
			magicName = "op^"
		case "<":
			magicName = "op<"
		case "<=":
			magicName = "op<="
		case ">":
			magicName = "op>"
		case ">=":
			magicName = "op>="
		case "<<":
			magicName = "op<<"
		case ">>":
			magicName = "op>>"
		case "==":
			magicName = "op=="
		case "!=":
			magicName = "op!="
		case "&&":
			if leftVal.Type != "bool" || rightVal.Type != "bool" {
				return nil, fmt.Errorf("TypeError: logical AND operator requires boolean operands, got '%s' and '%s'", leftVal.Type, rightVal.Type)
			}
			leftBool, ok := leftVal.Value.(bool)
			if !ok {
				return nil, fmt.Errorf("RuntimeError: invalid boolean value for logical AND operator")
			}
			rightBool, ok := rightVal.Value.(bool)
			if !ok {
				return nil, fmt.Errorf("RuntimeError: invalid boolean value for logical AND operator")
			}
			return NewBool(leftBool && rightBool)
		case "||":
			if leftVal.Type != "bool" || rightVal.Type != "bool" {
				return nil, fmt.Errorf("TypeError: logical OR operator requires boolean operands, got '%s' and '%s'", leftVal.Type, rightVal.Type)
			}
			leftBool, ok := leftVal.Value.(bool)
			if !ok {
				return nil, fmt.Errorf("RuntimeError: invalid boolean value for logical OR operator")
			}
			rightBool, ok := rightVal.Value.(bool)
			if !ok {
				return nil, fmt.Errorf("RuntimeError: invalid boolean value for logical OR operator")
			}
			return NewBool(leftBool || rightBool)
		case "^^":
			if leftVal.Type != "bool" || rightVal.Type != "bool" {
				return nil, fmt.Errorf("TypeError: logical XOR operator requires boolean operands, got '%s' and '%s'", leftVal.Type, rightVal.Type)
			}
			leftBool, ok := leftVal.Value.(bool)
			if !ok {
				return nil, fmt.Errorf("RuntimeError: invalid boolean value for logical XOR operator")
			}
			rightBool, ok := rightVal.Value.(bool)
			if !ok {
				return nil, fmt.Errorf("RuntimeError: invalid boolean value for logical XOR operator")
			}
			return NewBool(leftBool != rightBool)
		default:
			return nil, fmt.Errorf("RuntimeError: unknown binary operator '%s'", operator)
		}
		opMagic, ok := leftVal.Magic[magicName]
		if !ok {
			return nil, fmt.Errorf("TypeError: object of type '%s' does not support operator '%s'", leftVal.Type, operator)
		}
		return opMagic(leftVal, []Obj{rightVal})
	case "ternary":  // Ternary operation
		cond, ok := cmd["cond"]
		if !ok {
			return nil, fmt.Errorf("RuntimeError: invalid ternary command, missing 'cond' key")
		}
		trueVal, ok := cmd["true_expr"]
		if !ok {
			return nil, fmt.Errorf("RuntimeError: invalid ternary command, missing 'true_expr' key")
		}
		falseVal, ok := cmd["false_expr"]
		if !ok {
			return nil, fmt.Errorf("RuntimeError: invalid ternary command, missing 'false_expr' key")
		}
		condVal, err := rt.GetValue(cond)
		if err != nil {
			return nil, err
		}
		if condVal.Type != "bool" {
			return nil, fmt.Errorf("TypeError: condition in ternary operator must be boolean, got '%s'", condVal.Type)
		}
		condBool, ok := condVal.Value.(bool)
		if !ok {
			return nil, fmt.Errorf("RuntimeError: invalid boolean value for ternary operator condition")
		}
		if condBool {
			return rt.GetValue(trueVal)
		} else {
			return rt.GetValue(falseVal)
		}
	case "map":  // Map literal
		keys, ok := cmd["keys"].([]any)
		if !ok {
			return nil, fmt.Errorf("RuntimeError: invalid map command, missing 'keys' key or 'keys' is not a list")
		}
		values, ok := cmd["values"].([]any)
		if !ok {
			return nil, fmt.Errorf("RuntimeError: invalid map command, missing 'values' key or 'values' is not a list")
		}
		if len(keys) != len(values) {
			return nil, fmt.Errorf("RuntimeError: number of keys and values in map command do not match")
		}
		keyObjs := make([]Obj, len(keys))
		valueObjs := make([]Obj, len(values))
		for i := 0; i < len(keys); i++ {
			keyObj, err := rt.GetValue(keys[i])
			if err != nil {
				return nil, err
			}
			keyObjs[i] = keyObj
			valueObj, err := rt.GetValue(values[i])
			if err != nil {
				return nil, err
			}
			valueObjs[i] = valueObj
		}
		return NewMap([2][]Obj{keyObjs, valueObjs})
	case "function_call":  // Function call
		callable, ok := cmd["function"]
		if !ok {
			return nil, fmt.Errorf("RuntimeError: invalid function_call command, missing 'function' key")
		}
		callableVal, err := rt.GetValue(callable)
		if err != nil {
			return nil, err
		}
		args, ok := cmd["args"].([]map[string]any)
		if !ok {
			argsList, ok := cmd["args"].([]any)
			if !ok {
				return nil, fmt.Errorf("RuntimeError: invalid function_call command, missing 'args' key or 'args' is not a list")
			}
			args, ok = ToSliceOfMapStringAny(argsList)
			if !ok {
				return nil, fmt.Errorf("RuntimeError: invalid function_call command, missing 'args' key or 'args' is not a list")
			}
		}
		argObjs := make([]Obj, len(args))
		for i, arg := range args {
			argUnpack, ok := arg["unpack"].(bool)
			if !ok {
				return nil, fmt.Errorf("RuntimeError: invalid function_call command, argument missing 'unpack' key or 'unpack' is not a boolean")
			}
			argVal, err := rt.GetValue(arg["value"])
			if err != nil {
				return nil, err
			}
			if argUnpack {
				// Parse to array and unpack
				parseArr, err := ToArray(argVal)
				if err != nil {
					return nil, err
				}
				parseArrVal, ok := parseArr.Value.([]Obj)
				if !ok {
					return nil, fmt.Errorf("RuntimeError: cannot unpack non-array value in function call argument")
				}
				argObjs = append(argObjs, parseArrVal...)
			} else {
				argObjs[i] = argVal
			}
		}
		callMagic, ok := callableVal.Magic["call"]
		if !ok {
			return nil, fmt.Errorf("TypeError: object of type '%s' is not callable", callableVal.Type)
		}
		return callMagic(callableVal, argObjs)
	case "func_def":  // Function definition (anonymous function)
		args, ok := cmd["params"].([]string)
		if !ok {
			return nil, fmt.Errorf("RuntimeError: invalid func_def command, missing 'params' key or 'params' is not a list of strings")
		}
		body, ok := cmd["body"].([]map[string]any)
		if !ok {
			bodyList, ok := cmd["body"].([]any)
			if !ok {
				return nil, fmt.Errorf("RuntimeError: invalid func_def command, missing 'body' key or 'body' is not a list of command objects")
			}
			body, ok = ToSliceOfMapStringAny(bodyList)
			if !ok {
				return nil, fmt.Errorf("RuntimeError: invalid func_def command, missing 'body' key or 'body' is not a list of command objects")
			}
		}
		infArgs, ok := cmd["inf_args"].(bool)
		if !ok {
			return nil, fmt.Errorf("RuntimeError: invalid func_def command, missing 'inf_args' key or 'inf_args' is not a boolean")
		}
		return NewFunction(rt.DefineFunction(args, body, infArgs))
	case "self_incr_decr":
		operator, ok := cmd["operator"].(string)
		if !ok {
			return nil, fmt.Errorf("RuntimeError: invalid self_incr_decr command, missing 'operator' key or 'operator' is not a string")
		}
		target, ok := cmd["target"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("RuntimeError: invalid self_incr_decr command, missing 'target' key")
		}
		operand, ok := cmd["operand"]
		if !ok {
			return nil, fmt.Errorf("RuntimeError: invalid self_incr_decr command, missing 'operand' key")
		}
		operandVal, err := rt.GetValue(operand)
		if err != nil {
			return nil, err
		}
		changeFunc := func() error {
			var opMagicName string
			switch operator {
			case "++":
				opMagicName = "op+"
			case "--":
				opMagicName = "op-"
			default:
				return fmt.Errorf("RuntimeError: unknown self increment/decrement operator '%s'", operator)
			}
			opMagic, ok := operandVal.Magic[opMagicName]
			if !ok {
				return fmt.Errorf("TypeError: object of type '%s' does not support operator '%s'", operandVal.Type, operator[:1])
			}
			oneVal, err := NewNumber(1)
			if err != nil {
				return err
			}
			newVal, err := opMagic(operandVal, []Obj{oneVal})
			if err != nil {
				return err
			}
			err = rt.Set(target, newVal)
			if err != nil {
				return err
			}
			return nil
		}
		position, ok := cmd["position"].(string)
		if !ok {
			return nil, fmt.Errorf("RuntimeError: invalid self_incr_decr command, missing 'position' key or 'position' is not a string")
		}
		switch position {
		case "prefix":
			err := changeFunc()
			if err != nil {
				return nil, err
			}
			return rt.GetValue(operand)
		case "postfix":
			currentVal, err := rt.GetValue(operand)
			if err != nil {
				return nil, err
			}
			err = changeFunc()
			if err != nil {
				return nil, err
			}
			return currentVal, nil
		default:
			return nil, fmt.Errorf("RuntimeError: unknown self increment/decrement position '%s'", position)
		}
	}
	return nil, fmt.Errorf("RuntimeError: unknown command type '%s'", cmdType)
}

func (rt *Runtime) GotoEntryPoint() {
	entryPoint := 0
	for i, cmd := range rt.Cmds {
		if cmdType, ok := cmd["cmd"].(string); ok && cmdType == "entry_point" {
			entryPoint = i
			break
		}
	}
	rt.CmdIndex = entryPoint
}
