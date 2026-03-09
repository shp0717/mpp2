package runtime

import (
	"fmt"
)

func (rt *Runtime) Run() error {
	for rt.CmdIndex < len(rt.Cmds) {
		err := rt.RunCmd()
		if err != nil {
			rt.RaiseError(err)
			return err
		}
		rt.CmdIndex++
	}
	return nil
}

func (rt *Runtime) RunOneCmd(cmd map[string]any) error {
	newRt := ChunkRuntime(
		[]map[string]any{cmd},
		*rt,
	)
	err := newRt.Run()
	if err != nil {
		return err
	}
	SyncRuntime(rt, newRt)
	return nil
}

func (rt *Runtime) RunCmd() error {
	cmd := rt.Cmds[rt.CmdIndex]
	cmdExc, ok := cmd["exc"].(string)
	if !ok {
		return fmt.Errorf("RuntimeError: invalid command exception: \n%v", cmd["exc"])
	}
	rt.CurrentExc = cmdExc
	if len(rt.Loops) > 0 {
		currentLoop := rt.Loops[len(rt.Loops)-1]
		if currentLoop.status == "break" || currentLoop.status == "continue" {
			return nil
		}
	}
	if len(rt.Functions) > 0 {
		currentFunc := rt.Functions[len(rt.Functions)-1]
		if currentFunc.status == "return" {
			return nil
		}
	}
	switch c := cmd["cmd"].(string); c {
	case "set":
		target, ok := cmd["target"].(map[string]any)
		if !ok {
			return fmt.Errorf("RuntimeError: invalid set command target")
		}
		return rt.SetStatement(target, cmd["value"])
	case "delete":
		target, ok := cmd["target"].(map[string]any)
		if !ok {
			return fmt.Errorf("RuntimeError: invalid delete command target")
		}
		return rt.Delete(target)
	case "if_chunk":
		return rt.RunIfChunk(cmd)
	case "for_loop":
		return rt.RunForLoop(cmd)
	case "for_each":
		return rt.RunForEach(cmd)
	case "while_loop":
		return rt.RunWhileLoop(cmd)
	case "do_while_loop":
		return rt.RunDoWhileLoop(cmd)
	case "return":
		value, err := rt.GetValue(cmd["value"])
		if err != nil {
			return err
		}
		if len(rt.Functions) == 0 {
			return fmt.Errorf("SyntaxError: 'return' outside of function")
		}
		currentFunc := rt.Functions[len(rt.Functions)-1]
		currentFunc.status = "return"
		currentFunc.value = value
		return nil
	case "break":
		if len(rt.Loops) == 0 {
			return fmt.Errorf("SyntaxError: 'break' outside of loop")
		}
		currentLoop := rt.Loops[len(rt.Loops)-1]
		currentLoop.status = "break"
		return nil
	case "continue":
		if len(rt.Loops) == 0 {
			return fmt.Errorf("SyntaxError: 'continue' outside of loop")
		}
		currentLoop := rt.Loops[len(rt.Loops)-1]
		currentLoop.status = "continue"
		return nil
	case "label_def":
		name, ok := cmd["name"].(string)
		if !ok {
			return fmt.Errorf("RuntimeError: invalid label name")
		}
		rt.Labels[name] = rt.CmdIndex
		return nil
	case "goto":
		name, ok := cmd["name"].(string)
		if !ok {
			return fmt.Errorf("RuntimeError: invalid label name in goto")
		}
		labelIndex, ok := rt.Labels[name]
		if ok {
			rt.CmdIndex = labelIndex
			return nil
		} else {
			// try to find label in the remaining commands
			for i := rt.CmdIndex + 1; i < len(rt.Cmds); i++ {
				cmd := rt.Cmds[i]
				if cmd["cmd"] == "label_def" {
					labelName, ok := cmd["name"].(string)
					rt.Labels[labelName] = i
					if ok && labelName == name {
						rt.CmdIndex = i
						return nil
					}
				}
			}
			return fmt.Errorf("RuntimeError: label '%s' not found", name)
		}
	case "import":
		moduleName, ok := cmd["name"].(string)
		if !ok {
			return fmt.Errorf("RuntimeError: invalid module name in import")
		}
		content, ok := cmd["content"].([]map[string]any)
		if !ok {
			contentList, ok := cmd["content"].([]any)
			if !ok {
				return fmt.Errorf("RuntimeError: invalid module content in import")
			}
			content, ok = ToSliceOfMapStringAny(contentList)
			if !ok {
				return fmt.Errorf("RuntimeError: invalid module content in import")
			}
		}
		moduleObj, err := NewSubRuntime(
			content,
		)
		if err != nil {
			return err
		}
		rt.SetVar(moduleName, moduleObj)
		return nil
	case "builtin_import":
		moduleName, ok := cmd["name"].(string)
		if !ok {
			return fmt.Errorf("RuntimeError: invalid module name in builtin_import")
		}
		moduleObj, ok := BUILTIN_MODULES[moduleName]
		if !ok {
			return fmt.Errorf("ModuleNotFoundError: no module named '%s'", moduleName)
		}
		rt.SetVar(moduleName, moduleObj)
		return nil
	case "entry_point":
		return nil
	case "expr":
		_, err := rt.GetValue(cmd["value"])
		return err
	default:
		return fmt.Errorf("RuntimeError: unknown command '%s'", c)
	}
}

func (rt *Runtime) SetStatement(target map[string]any, value any) error {
	valueObj, err := rt.GetValue(value)
	if err != nil {
		return err
	}
	err = rt.Set(target, valueObj)
	if err != nil {
		return err
	}
	return nil
}

func (rt *Runtime) Set(target map[string]any, value Obj) error {
	targetKind, ok := target["type"].(string)
	if !ok {
		return fmt.Errorf("RuntimeError: invalid set statement target")
	}
	switch targetKind {
	case "var":
		varName, ok := target["name"].(string)
		if !ok {
			return fmt.Errorf("RuntimeError: invalid variable name")
		}
		rt.SetVar(varName, value)
		return nil
	case "item":
		setTarget := target["target"]
		key := target["key"]
		targetObj, err := rt.GetValue(setTarget)
		if err != nil {
			return err
		}
		keyObj, err := rt.GetValue(key)
		if err != nil {
			return err
		}
		setItemMagic, ok := targetObj.Magic["set_item"]
		if !ok {
			return fmt.Errorf("RuntimeError: object of type '%s' does not support item assignment", targetObj.Type)
		}
		_, err = setItemMagic(targetObj, []Obj{keyObj, value})
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("RuntimeError: unknown target command '%v'", targetKind)
	}
}

func (rt *Runtime) Delete(target map[string]any) error {
	targetKind, ok := target["type"].(string)
	if !ok {
		return fmt.Errorf("RuntimeError: invalid delete statement target")
	}
	switch targetKind {
	case "var":
		varName, ok := target["name"].(string)
		if !ok {
			return fmt.Errorf("RuntimeError: invalid variable name in delete statement")
		}
		rt.DeleteVar(varName)
		return nil
	case "item":
		deleteTarget := target["target"]
		key := target["key"]
		targetObj, err := rt.GetValue(deleteTarget)
		if err != nil {
			return err
		}
		keyObj, err := rt.GetValue(key)
		if err != nil {
			return err
		}
		deleteItemMagic, ok := targetObj.Magic["delete_item"]
		if !ok {
			return fmt.Errorf("RuntimeError: object of type '%s' does not support item deletion", targetObj.Type)
		}
		_, err = deleteItemMagic(targetObj, []Obj{keyObj})
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("RuntimeError: unknown target command '%v'", targetKind)
	}
}

func (rt *Runtime) RunIfChunk(cmd map[string]any) error {
	conds, ok := cmd["conds"].([]map[string]any)
	if !ok {
		condsList, ok := cmd["conds"].([]any)
		if !ok {
			return fmt.Errorf("RuntimeError: invalid if_chunk conditions")
		}
		conds, ok = ToSliceOfMapStringAny(condsList)
		if !ok {
			return fmt.Errorf("RuntimeError: invalid if_chunk conditions")
		}
	}
	for _, cond := range conds {
		cmdExc, ok := cond["exc"].(string)
		if !ok {
			return fmt.Errorf("RuntimeError: invalid if_chunk condition exception")
		}
		rt.CurrentExc = cmdExc
		condValue, err := rt.GetValue(cond["cond"])
		if err != nil {
			return err
		}
		if condValue.Type != "bool" {
			return fmt.Errorf("RuntimeError: if condition does not evaluate to a boolean")
		}
		condBool, ok := condValue.Value.(bool)
		if !ok {
			return fmt.Errorf("RuntimeError: if condition does not evaluate to a boolean")
		}
		if condBool {
			cmds, ok := cond["body"].([]map[string]any)
			if !ok {
				cmdsList, ok := cond["body"].([]any)
				if !ok {
					return fmt.Errorf("RuntimeError: invalid if_chunk commands")
				}
				cmds, ok = ToSliceOfMapStringAny(cmdsList)
				if !ok {
					return fmt.Errorf("RuntimeError: invalid if_chunk commands")
				}
			}
			newRt := ChunkRuntime(cmds, *rt)
			err := newRt.Run()
			if err != nil {
				return err
			}
			SyncRuntime(rt, newRt)
			return nil
		}
	}
	return nil
}

func (rt *Runtime) RunForLoop(cmd map[string]any) error {
	// C-style for loop: for init; cond; post { cmds }
	initCmds, ok := cmd["init"].([]map[string]any)
	if !ok {
		initCmdsList, ok := cmd["init"].([]any)
		if !ok {
			return fmt.Errorf("RuntimeError: invalid for_loop init command")
		}
		initCmds, ok = ToSliceOfMapStringAny(initCmdsList)
		if !ok {
			return fmt.Errorf("RuntimeError: invalid for_loop init command")
		}
	}
	condCmd, ok := cmd["cond"]
	if !ok {
		return fmt.Errorf("RuntimeError: invalid for_loop condition command")
	}
	postCmds, ok := cmd["incr"].([]map[string]any)
	if !ok {
		postCmdsList, ok := cmd["incr"].([]any)
		if !ok {
			return fmt.Errorf("RuntimeError: invalid for_loop post command")
		}
		postCmds, ok = ToSliceOfMapStringAny(postCmdsList)
		if !ok {
			return fmt.Errorf("RuntimeError: invalid for_loop post command")
		}
	}
	cmds, ok := cmd["body"].([]map[string]any)
	if !ok {
		cmdsList, ok := cmd["body"].([]any)
		if !ok {
			return fmt.Errorf("RuntimeError: invalid for_loop commands")
		}
		cmds, ok = ToSliceOfMapStringAny(cmdsList)
		if !ok {
			return fmt.Errorf("RuntimeError: invalid for_loop commands")
		}
	}
	exc, ok := cmd["exc"].(string)
	if !ok {
		return fmt.Errorf("RuntimeError: invalid for_loop exception")
	}
	// Run init command
	for _, initCmd := range initCmds {
		err := rt.RunOneCmd(initCmd)
		if err != nil {
			return err
		}
	}
	loop := Loop{
		status: "running",
	}
	rt.PushLoop(&loop)
	for {
		// Check condition
		rt.CurrentExc = exc
		condValue, err := rt.GetValue(condCmd)
		if err != nil {
			return err
		}
		if condValue.Type != "bool" {
			return fmt.Errorf("RuntimeError: for loop condition does not evaluate to a boolean")
		}
		condBool, ok := condValue.Value.(bool)
		if !ok {
			return fmt.Errorf("RuntimeError: for loop condition does not evaluate to a boolean")
		}
		if !condBool {
			break
		}
		if loop.status == "break" {
			break
		}
		if loop.status == "continue" {
			loop.status = "running"
			for _, postCmd := range postCmds {
				err := rt.RunOneCmd(postCmd)
				if err != nil {
					return err
				}
			}
			continue
		}
		// Run loop body
		newRt := ChunkRuntime(cmds, *rt)
		err = newRt.Run()
		if err != nil {
			return err
		}
		SyncRuntime(rt, newRt)
		// Run post command
		for _, postCmd := range postCmds {
			err := rt.RunOneCmd(postCmd)
			if err != nil {
				return err
			}
		}
	}
	rt.RemoveLoop(&loop)
	return nil
}

func (rt *Runtime) RunForEach(cmd map[string]any) error {
	exc, ok := cmd["exc"].(string)
	if !ok {
		return fmt.Errorf("RuntimeError: invalid for_each exception")
	}
	iterableCmd, ok := cmd["iterable"]
	if !ok {
		return fmt.Errorf("RuntimeError: invalid for_each iterable command")
	}
	rt.CurrentExc = exc
	iterableValue, err := rt.GetValue(iterableCmd)
	if err != nil {
		return err
	}
	iterArray, err := ToArray(iterableValue)
	if err != nil {
		return err
	}
	iterSlice, ok := iterArray.Value.([]Obj)
	if !ok {
		return fmt.Errorf("RuntimeError: for_each target is not iterable")
	}
	target, ok := cmd["target"].(map[string]any)
	if !ok {
		return fmt.Errorf("RuntimeError: invalid for_each target")
	}
	bodyCmds, ok := cmd["body"].([]map[string]any)
	if !ok {
		bodyCmdsList, ok := cmd["body"].([]any)
		if !ok {
			return fmt.Errorf("RuntimeError: invalid for_each body commands")
		}
		bodyCmds, ok = ToSliceOfMapStringAny(bodyCmdsList)
		if !ok {
			return fmt.Errorf("RuntimeError: invalid for_each body commands")
		}
	}
	loop := Loop{
		status: "running",
	}
	rt.PushLoop(&loop)
	for _, item := range iterSlice {
		if loop.status == "break" {
			break
		}
		if loop.status == "continue" {
			loop.status = "running"
			continue
		}
		// Set loop variable
		rt.Set(target, item)
		// Run loop body
		newRt := ChunkRuntime(bodyCmds, *rt)
		err = newRt.Run()
		if err != nil {
			return err
		}
		SyncRuntime(rt, newRt)
	}
	rt.RemoveLoop(&loop)
	return nil
}

func (rt *Runtime) RunWhileLoop(cmd map[string]any) error {
	condCmd, ok := cmd["cond"]
	if !ok {
		return fmt.Errorf("RuntimeError: invalid while_loop condition command")
	}
	cmds, ok := cmd["body"].([]map[string]any)
	if !ok {
		cmdsList, ok := cmd["body"].([]any)
		if !ok {
			return fmt.Errorf("RuntimeError: invalid while_loop commands")
		}
		cmds, ok = ToSliceOfMapStringAny(cmdsList)
		if !ok {
			return fmt.Errorf("RuntimeError: invalid while_loop commands")
		}
	}
	exc, ok := cmd["exc"].(string)
	if !ok {
		return fmt.Errorf("RuntimeError: invalid while_loop exception")
	}
	loop := Loop{
		status: "running",
	}
	rt.PushLoop(&loop)
	for {
		// Check condition
		rt.CurrentExc = exc
		condValue, err := rt.GetValue(condCmd)
		if err != nil {
			return err
		}
		if condValue.Type != "bool" {
			return fmt.Errorf("RuntimeError: while loop condition does not evaluate to a boolean")
		}
		condBool, ok := condValue.Value.(bool)
		if !ok {
			return fmt.Errorf("RuntimeError: while loop condition does not evaluate to a boolean")
		}
		if !condBool {
			break
		}
		if loop.status == "break" {
			break
		}
		if loop.status == "continue" {
			loop.status = "running"
			continue
		}
		// Run loop body
		newRt := ChunkRuntime(cmds, *rt)
		err = newRt.Run()
		if err != nil {
			return err
		}
		SyncRuntime(rt, newRt)
	}
	rt.RemoveLoop(&loop)
	return nil
}

func (rt *Runtime) RunDoWhileLoop(cmd map[string]any) error {
	condCmd, ok := cmd["cond"]
	if !ok {
		return fmt.Errorf("RuntimeError: invalid do_while_loop condition command")
	}
	cmds, ok := cmd["body"].([]map[string]any)
	if !ok {
		cmdsList, ok := cmd["body"].([]any)
		if !ok {
			return fmt.Errorf("RuntimeError: invalid do_while_loop commands")
		}
		cmds, ok = ToSliceOfMapStringAny(cmdsList)
		if !ok {
			return fmt.Errorf("RuntimeError: invalid do_while_loop commands")
		}
	}
	exc, ok := cmd["exc"].(string)
	if !ok {
		return fmt.Errorf("RuntimeError: invalid do_while_loop exception")
	}
	loop := Loop{
		status: "running",
	}
	rt.PushLoop(&loop)
	for {
		if loop.status == "break" {
			break
		}
		if loop.status == "continue" {
			loop.status = "running"
			rt.CurrentExc = exc
			condValue, err := rt.GetValue(condCmd)
			if err != nil {
				return err
			}
			if condValue.Type != "bool" {
				return fmt.Errorf("RuntimeError: do-while loop condition does not evaluate to a boolean")
			}
			condBool, ok := condValue.Value.(bool)
			if !ok {
				return fmt.Errorf("RuntimeError: do-while loop condition does not evaluate to a boolean")
			}
			if !condBool {
				break
			}
			continue
		}
		// Run loop body
		newRt := ChunkRuntime(cmds, *rt)
		err := newRt.Run()
		if err != nil {
			return err
		}
		SyncRuntime(rt, newRt)
		// Check condition
		rt.CurrentExc = exc
		condValue, err := rt.GetValue(condCmd)
		if err != nil {
			return err
		}
		if condValue.Type != "bool" {
			return fmt.Errorf("RuntimeError: do-while loop condition does not evaluate to a boolean")
		}
		condBool, ok := condValue.Value.(bool)
		if !ok {
			return fmt.Errorf("RuntimeError: do-while loop condition does not evaluate to a boolean")
		}
		if !condBool {
			break
		}
	}
	rt.RemoveLoop(&loop)
	return nil
}
