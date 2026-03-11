package runtime

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

type (
	Obj      = *MeowppObject
	ObjMagic = func(Obj, []Obj) (Obj, error)
)

type MeowppObject struct {
	Type  string
	Value any
	Magic map[string]ObjMagic
}

var TYPES = []string{
	"number",
	"string",
	"bool",
	"null",
	"array",
	"map",
	"function",
	"subruntime",
}

var ( // Define magic methods for each type
	NumberMagic = map[string]ObjMagic{
		"op+": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for + operator, got %d", len(args))
			}
			if args[0].Type != "number" {
				return nil, fmt.Errorf("TypeError: expected number argument for + operator, got %s", args[0].Type)
			}
			return NewNumber(self.Value.(float64) + args[0].Value.(float64))
		},
		"op-": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for - operator, got %d", len(args))
			}
			if args[0].Type != "number" {
				return nil, fmt.Errorf("TypeError: expected number argument for - operator, got %s", args[0].Type)
			}
			return NewNumber(self.Value.(float64) - args[0].Value.(float64))
		},
		"op*": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for * operator, got %d", len(args))
			}
			if args[0].Type != "number" {
				return nil, fmt.Errorf("TypeError: expected number argument for * operator, got %s", args[0].Type)
			}
			return NewNumber(self.Value.(float64) * args[0].Value.(float64))
		},
		"op/": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for / operator, got %d", len(args))
			}
			if args[0].Type != "number" {
				return nil, fmt.Errorf("TypeError: expected number argument for / operator, got %s", args[0].Type)
			}
			if args[0].Value.(float64) == 0 {
				return nil, fmt.Errorf("ZeroDivisionError: division by zero")
			}
			return NewNumber(self.Value.(float64) / args[0].Value.(float64))
		},
		"op%": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for %% operator, got %d", len(args))
			}
			if args[0].Type != "number" {
				return nil, fmt.Errorf("TypeError: expected number argument for %% operator, got %s", args[0].Type)
			}
			if args[0].Value.(float64) == 0 {
				return nil, fmt.Errorf("ZeroDivisionError: division by zero")
			}
			divided := math.Floor(self.Value.(float64) / args[0].Value.(float64))
			return NewNumber(self.Value.(float64) - divided*args[0].Value.(float64))
		},
		"op^": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for ^ operator, got %d", len(args))
			}
			if args[0].Type != "number" {
				return nil, fmt.Errorf("TypeError: expected number argument for ^ operator, got %s", args[0].Type)
			}
			return NewNumber(math.Pow(self.Value.(float64), args[0].Value.(float64)))
		},
		"op<": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for < operator, got %d", len(args))
			}
			if args[0].Type != "number" {
				return nil, fmt.Errorf("TypeError: expected number argument for < operator, got %s", args[0].Type)
			}
			return NewBool(self.Value.(float64) < args[0].Value.(float64))
		},
		"op>": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for > operator, got %d", len(args))
			}
			if args[0].Type != "number" {
				return nil, fmt.Errorf("TypeError: expected number argument for > operator, got %s", args[0].Type)
			}
			return NewBool(self.Value.(float64) > args[0].Value.(float64))
		},
		"op<=": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for <= operator, got %d", len(args))
			}
			if args[0].Type != "number" {
				return nil, fmt.Errorf("TypeError: expected number argument for <= operator, got %s", args[0].Type)
			}
			return NewBool(self.Value.(float64) <= args[0].Value.(float64))
		},
		"op>=": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for >= operator, got %d", len(args))
			}
			if args[0].Type != "number" {
				return nil, fmt.Errorf("TypeError: expected number argument for >= operator, got %s", args[0].Type)
			}
			return NewBool(self.Value.(float64) >= args[0].Value.(float64))
		},
		"op==": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for == operator, got %d", len(args))
			}
			return NewBool(self.Value.(float64) == args[0].Value.(float64))
		},
		"op!=": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for != operator, got %d", len(args))
			}
			return NewBool(self.Value.(float64) != args[0].Value.(float64))
		},
		"pos": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 0 {
				return nil, fmt.Errorf("RuntimeError: expected 0 arguments for unary + operator, got %d", len(args))
			}
			return NewNumber(self.Value.(float64))
		},
		"neg": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 0 {
				return nil, fmt.Errorf("RuntimeError: expected 0 arguments for unary - operator, got %d", len(args))
			}
			return NewNumber(-self.Value.(float64))
		},
		"bool": func(self Obj, args []Obj) (Obj, error) {
			return NewBool(self.Value.(float64) != 0)
		},
		"string": func(self Obj, args []Obj) (Obj, error) {
			return NewString(fmt.Sprintf("%v", self.Value))
		},
		"repr": func(self Obj, args []Obj) (Obj, error) {
			return NewString(fmt.Sprintf("%v", self.Value))
		},
	}
	StringMagic = map[string]ObjMagic{
		"op+": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for + operator, got %d", len(args))
			}
			if args[0].Type != "string" {
				return nil, fmt.Errorf("TypeError: expected string argument for + operator, got %s", args[0].Type)
			}
			return NewString(self.Value.(string) + args[0].Value.(string))
		},
		"op*": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for * operator, got %d", len(args))
			}
			if args[0].Type != "number" {
				return nil, fmt.Errorf("TypeError: expected number argument for * operator, got %s", args[0].Type)
			}
			count := args[0].Value.(float64)
			if count < 0 {
				return nil, fmt.Errorf("ValueError: negative repeat count: %d", int(count))
			}
			if !IsInteger(args[0]) {
				return nil, fmt.Errorf("ValueError: non-integer repeat count: %d", int(count))
			}
			return NewString(strings.Repeat(self.Value.(string), int(count)))
		},
		"op%": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for %% operator, got %d", len(args))
			}
			if args[0].Type != "array" {
				return nil, fmt.Errorf("TypeError: expected array argument for %% operator, got %s", args[0].Type)
			}
			return Sprintf(self, args[0])
		},
		"op<": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for < operator, got %d", len(args))
			}
			if args[0].Type != "string" {
				return nil, fmt.Errorf("TypeError: expected string argument for < operator, got %s", args[0].Type)
			}
			return NewBool(self.Value.(string) < args[0].Value.(string))
		},
		"op>": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for > operator, got %d", len(args))
			}
			if args[0].Type != "string" {
				return nil, fmt.Errorf("TypeError: expected string argument for > operator, got %s", args[0].Type)
			}
			return NewBool(self.Value.(string) > args[0].Value.(string))
		},
		"op<=": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for <= operator, got %d", len(args))
			}
			if args[0].Type != "string" {
				return nil, fmt.Errorf("TypeError: expected string argument for <= operator, got %s", args[0].Type)
			}
			return NewBool(self.Value.(string) <= args[0].Value.(string))
		},
		"op>=": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for >= operator, got %d", len(args))
			}
			if args[0].Type != "string" {
				return nil, fmt.Errorf("TypeError: expected string argument for >= operator, got %s", args[0].Type)
			}
			return NewBool(self.Value.(string) >= args[0].Value.(string))
		},
		"op==": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for == operator, got %d", len(args))
			}
			return NewBool(self.Value.(string) == args[0].Value.(string))
		},
		"op!=": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for != operator, got %d", len(args))
			}
			return NewBool(self.Value.(string) != args[0].Value.(string))
		},
		"bool": func(self Obj, args []Obj) (Obj, error) {
			return NewBool(self.Value.(string) != "")
		},
		"number": func(self Obj, args []Obj) (Obj, error) {
			num, err := strconv.ParseFloat(self.Value.(string), 64)
			if err != nil {
				return nil, fmt.Errorf("ValueError: cannot convert string to number: %s", self.Value.(string))
			}
			return NewNumber(num)
		},
		"array": func(self Obj, args []Obj) (Obj, error) {
			runes := []rune(self.Value.(string))
			array := make([]Obj, len(runes))
			for i, r := range runes {
				charStr := string(r)
				charObj, err := NewString(charStr)
				if err != nil {
					return nil, fmt.Errorf("RuntimeError: failed to create string object for character: %v", err)
				}
				array[i] = charObj
			}
			return NewArray(array)
		},
		"repr": func(self Obj, args []Obj) (Obj, error) {
			return NewString(fmt.Sprintf("%q", self.Value.(string)))
		},
		"get_item": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for get_item, got %d", len(args))
			}
			if args[0].Type != "number" && args[0].Type != "string" {
				return nil, fmt.Errorf("TypeError: expected number argument for get_item, got %s", args[0].Type)
			}
			switch args[0].Type {
			case "number":
				index := int(args[0].Value.(float64))
				runes := []rune(self.Value.(string))
				if index < 0 || index >= len(runes) {
					return nil, fmt.Errorf("IndexError: string index out of range: %d", index)
				}
				charObj, err := NewString(string(runes[index]))
				if err != nil {
					return nil, fmt.Errorf("RuntimeError: failed to create string object for character: %v", err)
				}
				return charObj, nil
			case "string":
				name := args[0].Value.(string)
				switch name {
				case "length":
					return NewNumber(float64(len(self.Value.(string))))
				default:
					return nil, fmt.Errorf("AttributeError: string has no attribute '%s'", name)
				}
			}
			return nil, fmt.Errorf("TypeError: unsupported argument type for get_item: %s", args[0].Type)
		},
		"length": func(self Obj, args []Obj) (Obj, error) {
			return NewNumber(float64(len(self.Value.(string))))
		},
	}
	BoolMagic = map[string]ObjMagic{
		"op<": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for < operator, got %d", len(args))
			}
			if args[0].Type != "bool" {
				return nil, fmt.Errorf("TypeError: expected bool argument for < operator, got %s", args[0].Type)
			}
			return NewBool(!self.Value.(bool) && args[0].Value.(bool))
		},
		"op>": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for > operator, got %d", len(args))
			}
			if args[0].Type != "bool" {
				return nil, fmt.Errorf("TypeError: expected bool argument for > operator, got %s", args[0].Type)
			}
			return NewBool(self.Value.(bool) && !args[0].Value.(bool))
		},
		"op<=": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for <= operator, got %d", len(args))
			}
			if args[0].Type != "bool" {
				return nil, fmt.Errorf("TypeError: expected bool argument for <= operator, got %s", args[0].Type)
			}
			return NewBool(!self.Value.(bool) || args[0].Value.(bool))
		},
		"op>=": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for >= operator, got %d", len(args))
			}
			if args[0].Type != "bool" {
				return nil, fmt.Errorf("TypeError: expected bool argument for >= operator, got %s", args[0].Type)
			}
			return NewBool(self.Value.(bool) || !args[0].Value.(bool))
		},
		"op==": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for == operator, got %d", len(args))
			}
			return NewBool(self.Value.(bool) == args[0].Value.(bool))
		},
		"op!=": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for != operator, got %d", len(args))
			}
			return NewBool(self.Value.(bool) != args[0].Value.(bool))
		},
		"number": func(self Obj, args []Obj) (Obj, error) {
			if self.Value.(bool) {
				return NewNumber(1)
			} else {
				return NewNumber(0)
			}
		},
		"repr": func(self Obj, args []Obj) (Obj, error) {
			return NewString(fmt.Sprintf("%v", self.Value))
		},
	}
	NullMagic = map[string]ObjMagic{
		"bool": func(self Obj, args []Obj) (Obj, error) {
			return NewBool(false)
		},
		"repr": func(self Obj, args []Obj) (Obj, error) {
			return NewString("null")
		},
		"number": func(self Obj, args []Obj) (Obj, error) {
			return NewNumber(0)
		},
		"op==": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for == operator, got %d", len(args))
			}
			return NewBool(args[0].Type == "null")
		},
		"op!=": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for != operator, got %d", len(args))
			}
			return NewBool(args[0].Type != "null")
		},
	}
	ArrayMagic = map[string]ObjMagic{
		"repr": func(self Obj, args []Obj) (Obj, error) {
			elements := self.Value.([]Obj)
			strs := make([]string, len(elements))
			for i, elem := range elements {
				elemRepr, err := Repr(elem)
				if err != nil {
					return nil, fmt.Errorf("RuntimeError: failed to get repr of array element: %v", err)
				}
				strs[i] = elemRepr.Value.(string)
			}
			return NewString("[" + strings.Join(strs, ", ") + "]")
		},
		"get_item": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for get_item, got %d", len(args))
			}
			if args[0].Type != "number" && args[0].Type != "string" {
				return nil, fmt.Errorf("TypeError: expected number argument for get_item, got %s", args[0].Type)
			}
			switch args[0].Type {
			case "number":
				index := int(args[0].Value.(float64))
				if index < 0 || index >= len(self.Value.([]Obj)) {
					return nil, fmt.Errorf("IndexError: array index out of range: %d", index)
				}
				return self.Value.([]Obj)[index], nil
			case "string":
				key := args[0].Value.(string)
				switch key {
				case "length":
					return NewNumber(float64(len(self.Value.([]Obj))))
				default:
					return nil, fmt.Errorf("KeyError: array has no key '%s'", key)
				}
			default:
				return nil, fmt.Errorf("TypeError: expected number or string argument for get_item, got %s", args[0].Type)
			}
		},
		"set_item": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("RuntimeError: expected 2 arguments for set_item, got %d", len(args))
			}
			if args[0].Type != "number" {
				return nil, fmt.Errorf("TypeError: expected number argument for set_item, got %s", args[0].Type)
			}
			index := int(args[0].Value.(float64))
			if index < 0 || index >= len(self.Value.([]Obj)) {
				return nil, fmt.Errorf("IndexError: array index out of range: %d", index)
			}
			self.Value.([]Obj)[index] = args[1]
			return nil, nil
		},
		"op+": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for + operator, got %d", len(args))
			}
			if args[0].Type != "array" {
				return nil, fmt.Errorf("TypeError: expected array argument for + operator, got %s", args[0].Type)
			}
			newArray := append(self.Value.([]Obj), args[0].Value.([]Obj)...)
			return NewArray(newArray)
		},
		"op*": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for * operator, got %d", len(args))
			}
			if args[0].Type != "number" {
				return nil, fmt.Errorf("TypeError: expected number argument for * operator, got %s", args[0].Type)
			}
			count := args[0].Value.(float64)
			if count < 0 {
				return nil, fmt.Errorf("ValueError: negative repeat count: %d", int(count))
			}
			if !IsInteger(args[0]) {
				return nil, fmt.Errorf("ValueError: non-integer repeat count: %d", int(count))
			}
			newArray := make([]Obj, 0, len(self.Value.([]Obj))*int(count))
			for i := 0; i < int(count); i++ {
				newArray = append(newArray, self.Value.([]Obj)...)
			}
			return NewArray(newArray)
		},
		"bool": func(self Obj, args []Obj) (Obj, error) {
			return NewBool(len(self.Value.([]Obj)) > 0)
		},
		"op==": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for == operator, got %d", len(args))
			}
			if args[0].Type != "array" {
				return nil, fmt.Errorf("TypeError: expected array argument for == operator, got %s", args[0].Type)
			}
			arr1 := self.Value.([]Obj)
			arr2 := args[0].Value.([]Obj)
			if len(arr1) != len(arr2) {
				return NewBool(false)
			}
			for i := range arr1 {
				eq, err := Equal(arr1[i], arr2[i])
				if err != nil {
					return nil, err
				}
				if !eq.Value.(bool) {
					return NewBool(false)
				}
			}
			return NewBool(true)
		},
		"op!=": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for != operator, got %d", len(args))
			}
			if args[0].Type != "array" {
				return nil, fmt.Errorf("TypeError: expected array argument for != operator, got %s", args[0].Type)
			}
			eqResult, err := self.Magic["op=="](self, args)
			if err != nil {
				return nil, err
			}
			return NewBool(!eqResult.Value.(bool))
		},
		"length": func(self Obj, args []Obj) (Obj, error) {
			return NewNumber(float64(len(self.Value.([]Obj))))
		},
	}
	MapMagic = map[string]ObjMagic{
		"get_item": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for get_item, got %d", len(args))
			}
			keys := self.Value.(map[string]Obj)["keys"].Value.([]Obj)
			values := self.Value.(map[string]Obj)["values"].Value.([]Obj)
			for i, keyObj := range keys {
				eq, err := Equal(keyObj, args[0])
				if err != nil {
					return nil, err
				}
				if eq.Value.(bool) {
					return values[i], nil
				}
			}
			repred, err := Repr(self)
			if err != nil {
				return nil, fmt.Errorf("RuntimeError: failed to get repr of map: %v", err)
			}
			repredKey, err := Repr(args[0])
			if err != nil {
				return nil, fmt.Errorf("RuntimeError: failed to get repr of map key: %v", err)
			}
			return nil, fmt.Errorf("KeyError: key %s not found in map %s", repredKey.Value.(string), repred.Value.(string))
		},
		"set_item": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("RuntimeError: expected 2 arguments for set_item, got %d", len(args))
			}
			keys := self.Value.(map[string]Obj)["keys"].Value.([]Obj)
			values := self.Value.(map[string]Obj)["values"].Value.([]Obj)
			for i, keyObj := range keys {
				if eq, err := Equal(keyObj, args[0]); err == nil && eq.Value.(bool) {
					values[i] = args[1]
					return nil, nil
				}
			}
			keys = append(keys, args[0])
			values = append(values, args[1])
			keysObj, err := NewArray(keys)
			if err != nil {
				return nil, err
			}
			valuesObj, err := NewArray(values)
			if err != nil {
				return nil, err
			}
			self.Value.(map[string]Obj)["keys"] = keysObj
			self.Value.(map[string]Obj)["values"] = valuesObj
			return nil, nil
		},
		"array": func(self Obj, args []Obj) (Obj, error) {
			keys := self.Value.(map[string]Obj)["keys"].Value.([]Obj)
			return NewArray(keys)
		},
		"repr": func(self Obj, args []Obj) (Obj, error) {
			keys := self.Value.(map[string]Obj)["keys"].Value.([]Obj)
			values := self.Value.(map[string]Obj)["values"].Value.([]Obj)
			pairs := make([]string, len(keys))
			for i := range keys {
				keyRepr, err := Repr(keys[i])
				if err != nil {
					return nil, fmt.Errorf("RuntimeError: failed to get repr of map key: %v", err)
				}
				valueRepr, err := Repr(values[i])
				if err != nil {
					return nil, fmt.Errorf("RuntimeError: failed to get repr of map value: %v", err)
				}
				pairs[i] = fmt.Sprintf("%s: %s", keyRepr.Value.(string), valueRepr.Value.(string))
			}
			return NewString("{" + strings.Join(pairs, ", ") + "}")
		},
		"bool": func(self Obj, args []Obj) (Obj, error) {
			keys := self.Value.(map[string]Obj)["keys"].Value.([]Obj)
			return NewBool(len(keys) > 0)
		},
		"op==": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for == operator, got %d", len(args))
			}
			if args[0].Type != "map" {
				return nil, fmt.Errorf("TypeError: expected map argument for == operator, got %s", args[0].Type)
			}
			keys1 := self.Value.(map[string]Obj)["keys"].Value.([]Obj)
			values1 := self.Value.(map[string]Obj)["values"].Value.([]Obj)
			keys2 := args[0].Value.(map[string]Obj)["keys"].Value.([]Obj)
			values2 := args[0].Value.(map[string]Obj)["values"].Value.([]Obj)
			if len(keys1) != len(keys2) {
				return NewBool(false)
			}
			for i, key1 := range keys1 {
				key2 := keys2[i]
				eq, err := Equal(key1, key2)
				if err != nil {
					return nil, err
				}
				if !eq.Value.(bool) {
					return NewBool(false)
				}
				value1 := values1[i]
				value2 := values2[i]
				eq, err = Equal(value1, value2)
				if err != nil {
					return nil, err
				}
				if !eq.Value.(bool) {
					return NewBool(false)
				}
			}
			return NewBool(true)
		},
		"op!=": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for != operator, got %d", len(args))
			}
			if args[0].Type != "map" {
				return nil, fmt.Errorf("TypeError: expected map argument for != operator, got %s", args[0].Type)
			}
			eqResult, err := self.Magic["op=="](self, args)
			if err != nil {
				return nil, err
			}
			return NewBool(!eqResult.Value.(bool))
		},
	}
	SubRuntimeMagic = map[string]ObjMagic{
		"repr": func(self Obj, args []Obj) (Obj, error) {
			return NewString(fmt.Sprintf("<subruntime at %p>", self))
		},
		"set_item": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("RuntimeError: expected 2 arguments for set_item, got %d", len(args))
			}
			if args[0].Type != "string" {
				return nil, fmt.Errorf("TypeError: expected string argument for set_item, got %s", args[0].Type)
			}
			target := args[0].Value.(string)
			self.Value.(*Runtime).Vars[fmt.Sprintf("%s@0", target)] = args[1]
			return nil, nil
		},
		"get_item": func(self Obj, args []Obj) (Obj, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("RuntimeError: expected 1 argument for get_item, got %d", len(args))
			}
			if args[0].Type != "string" {
				return nil, fmt.Errorf("TypeError: expected string argument for get_item, got %s", args[0].Type)
			}
			target := args[0].Value.(string)
			value, ok := self.Value.(*Runtime).Vars[fmt.Sprintf("%s@0", target)]
			if !ok {
				return nil, fmt.Errorf("AttributeError: '%s' not found in subruntime", target)
			}
			return value, nil
		},
	}
)

var ( // Define default methods for each type

)

var (
	numberMagicRef map[string]ObjMagic
	stringMagicRef map[string]ObjMagic
	boolMagicRef   map[string]ObjMagic
	nullMagicRef   map[string]ObjMagic
	arrayMagicRef  map[string]ObjMagic
	mapMagicRef    map[string]ObjMagic
	subRuntimeMagicRef map[string]ObjMagic
)

var EMPTY_MAGIC = map[string]ObjMagic{}

func InitMagicRefs() {
	numberMagicRef = NumberMagic
	stringMagicRef = StringMagic
	boolMagicRef = BoolMagic
	nullMagicRef = NullMagic
	arrayMagicRef = ArrayMagic
	mapMagicRef = MapMagic
	subRuntimeMagicRef = SubRuntimeMagic
}

func DBGPRT(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}

func NewMeowppObject(value any, magics map[string]ObjMagic) (Obj, error) {
	switch v := value.(type) {
	case int, float64:
		res, err := NewNumber(v)
		if err != nil {
			return nil, err
		}
		res.Magic = magics
		return res, nil
	case string:
		res, err := NewString(v)
		if err != nil {
			return nil, err
		}
		res.Magic = magics
		return res, nil
	case bool:
		res, err := NewBool(v)
		if err != nil {
			return nil, err
		}
		res.Magic = magics
		return res, nil
	case nil:
		res := NewNull()
		res.Magic = magics
		return res, nil
	case []any:
		res, err := NewArray(v)
		if err != nil {
			return nil, err
		}
		res.Magic = magics
		return res, nil
	case map[string]any:
		res, err := NewMap(v)
		if err != nil {
			return nil, err
		}
		res.Magic = magics
		return res, nil
	case func(args []MeowppObject) (MeowppObject, error):
		res, err := NewFunction(v)
		if err != nil {
			return nil, err
		}
		res.Magic = magics
		return res, nil
	case *Runtime:
		res, err := NewSubRuntime(v)
		if err != nil {
			return nil, err
		}
		res.Magic = magics
		return res, nil
	default:
		return nil, fmt.Errorf("RuntimeError: unsupported type: %T", value)
	}
}

func NewNumber(value any) (Obj, error) {
	switch v := value.(type) {
	case int:
		return &MeowppObject{
			Type:  "number",
			Value: float64(v),
			Magic: numberMagicRef,
		}, nil
	case float64:
		return &MeowppObject{
			Type:  "number",
			Value: v,
			Magic: numberMagicRef,
		}, nil
	default:
		return nil, fmt.Errorf("RuntimeError: invalid type for number: %T", value)
	}
}

func NewString(value any) (Obj, error) {
	switch v := value.(type) {
	case string:
		return &MeowppObject{
			Type:  "string",
			Value: v,
			Magic: stringMagicRef,
		}, nil
	default:
		return nil, fmt.Errorf("RuntimeError: invalid type for string: %T", value)
	}
}

func NewBool(value any) (Obj, error) {
	switch v := value.(type) {
	case bool:
		return &MeowppObject{
			Type:  "bool",
			Value: v,
			Magic: boolMagicRef,
		}, nil
	default:
		return nil, fmt.Errorf("RuntimeError: invalid type for bool: %T", value)
	}
}

func NewNull() Obj {
	return &MeowppObject{
		Type:  "null",
		Value: nil,
		Magic: nullMagicRef,
	}
}

func NewArray(value any) (Obj, error) {
	switch v := value.(type) {
	case []Obj:
		return &MeowppObject{
			Type: "array",
			Value: v,
			Magic: arrayMagicRef,
		}, nil
	default:
		return nil, fmt.Errorf("RuntimeError: invalid type for array: %T", value)
	}
}

func NewMap(value any) (Obj, error) {
	switch v := value.(type) {
	case [2][]Obj:
		keys := v[0]
		values := v[1]
		if len(keys) != len(values) {
			return nil, fmt.Errorf("RuntimeError: keys and values length mismatch: %d vs %d", len(keys), len(values))
		}
		keysArray, err := NewArray(keys)
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: invalid keys for map: %v", err)
		}
		valuesArray, err := NewArray(values)
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: invalid values for map: %v", err)
		}
		return &MeowppObject{
			Type:  "map",
			Value: map[string]Obj{
				"keys":   keysArray,
				"values": valuesArray,
			},
			Magic: mapMagicRef,
		}, nil
	case map[string]Obj:
		keys := make([]Obj, 0, len(v))
		values := make([]Obj, 0, len(v))
		for k, val := range v {
			keyObj, err := NewString(k)
			if err != nil {
				return nil, fmt.Errorf("RuntimeError: invalid key for map: %v", err)
			}
			keys = append(keys, keyObj)
			values = append(values, val)
		}
		keysArray, err := NewArray(keys)
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: invalid keys for map: %v", err)
		}
		valuesArray, err := NewArray(values)
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: invalid values for map: %v", err)
		}
		return &MeowppObject{
			Type:  "map",
			Value: map[string]Obj{
				"keys":   keysArray,
				"values": valuesArray,
			},
			Magic: mapMagicRef,
		}, nil
	default:
		return nil, fmt.Errorf("RuntimeError: invalid type for map: %T", value)
	}
}

func NewFunction(value any) (Obj, error) {
	switch v := value.(type) {
	case func(args []Obj) (Obj, error):
		return &MeowppObject{
			Type:  "function",
			Value: nil,
			Magic: map[string]ObjMagic{
				"call": func(self Obj, args []Obj) (Obj, error) {
					return v(args)
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("RuntimeError: invalid type for function: %T", value)
	}
}

func NewSubRuntime(value any) (Obj, error) {
	switch v := value.(type) {
	case *Runtime:
		return &MeowppObject{
			Type:  "subruntime",
			Value: v,
			Magic: subRuntimeMagicRef,
		}, nil
	case Runtime:
		return &MeowppObject{
			Type:  "subruntime",
			Value: &v,
			Magic: subRuntimeMagicRef,
		}, nil
	default:
		return nil, fmt.Errorf("RuntimeError: invalid type for subruntime: %T", value)
	}
}

func Method(self Obj, mthd func(self Obj, args []Obj) (Obj, error)) Obj {
	newFunc := func(args []Obj) (Obj, error) {
		return mthd(self, args)
	}
	res, err := NewFunction(newFunc)
	if err != nil {
		return nil
	}
	return res
}

func Sprintf(format Obj, args Obj) (Obj, error) {
	// C-style string formatting: %s for string, %d for integer, %f for float
	if format.Type != "string" {
		return nil, fmt.Errorf("TypeError: expected string format for sprintf, got %s", format.Type)
	}
	formatStr := format.Value.(string)
	argsArray, err := ToArray(args)
	if err != nil {
		return nil, fmt.Errorf("TypeError: expected array of arguments for sprintf, got %s", args.Type)
	}
	argsSlice := argsArray.Value.([]Obj)
	result := ""
	argIndex := 0
	for i := 0; i < len(formatStr); i++ {
		if formatStr[i] == '%' && i+1 < len(formatStr) {
			if argIndex >= len(argsSlice) {
				return nil, fmt.Errorf("RuntimeError: not enough arguments for sprintf format string")
			}
			switch formatStr[i+1] {
			case '%':
				result += "%"
			case 's':
				strArg, err := ToString(argsSlice[argIndex])
				if err != nil {
					return nil, fmt.Errorf("TypeError: cannot convert argument %d to string for %%s format", argIndex)
				}
				result += strArg.Value.(string)
			case 'd':
				numArg, err := ToNumber(argsSlice[argIndex])
				if err != nil {
					return nil, fmt.Errorf("TypeError: cannot convert argument %d to number for %%d format", argIndex)
				}
				result += strconv.Itoa(int(numArg.Value.(float64)))
			case 'f':
				numArg, err := ToNumber(argsSlice[argIndex])
				if err != nil {
					return nil, fmt.Errorf("TypeError: cannot convert argument %d to number for %%f format", argIndex)
				}
				result += fmt.Sprintf("%f", numArg.Value.(float64))
			default:
				return nil, fmt.Errorf("RuntimeError: unsupported format specifier: %%%c", formatStr[i+1])
			}
			argIndex++
			i++ // Skip the format specifier
		} else {
			result += string(formatStr[i])
		}
	}
	if argIndex < len(argsSlice) {
		return nil, fmt.Errorf("RuntimeError: too many arguments for sprintf format string")
	}
	return NewString(result)
}

func Sscanf(format Obj, input Obj) (Obj, error) {
	// C-style string parsing: %s for string, %d for integer, %f for float
	if format.Type != "string" {
		return nil, fmt.Errorf("TypeError: expected string format for sscanf, got %s", format.Type)
	}
	if input.Type != "string" {
		return nil, fmt.Errorf("TypeError: expected string input for sscanf, got %s", input.Type)
	}
	formatStr := format.Value.(string)
	for i := 0; i < len(formatStr); i++ {
		if formatStr[i] == '%' && i+1 < len(formatStr) {
			switch formatStr[i+1] {
			case '%', 's', 'd', 'f':
				i++ // Skip the format specifier
			default:
				return nil, fmt.Errorf("RuntimeError: unsupported format specifier: %%%c", formatStr[i+1])
			}
		}
	}
	var args []any
	_, err := fmt.Sscanf(input.Value.(string), formatStr, &args)
	if err != nil {
		return nil, fmt.Errorf("RuntimeError: failed to scan input: %v", err)
	}
	resultArray := make([]Obj, len(args))
	for i, arg := range args {
		obj, err := NewMeowppObject(arg, EMPTY_MAGIC)
		if err != nil {
			return nil, fmt.Errorf("RuntimeError: failed to convert scanned argument to Meowpp object: %v", err)
		}
		resultArray[i] = obj
	}
	return NewArray(resultArray)
}

func ToNumber(obj Obj) (Obj, error) {
	if obj.Type == "number" {
		return obj, nil
	} else if magic, ok := obj.Magic["number"]; ok {
		return magic(obj, nil)
	} else {
		return nil, fmt.Errorf("TypeError: cannot convert type %s to number", obj.Type)
	}
}

func ToString(obj Obj) (Obj, error) { // Must support string conversion for all types, using magic method if available
	if obj.Type == "string" {
		copy := *obj
		return &copy, nil
	} else if magic, ok := obj.Magic["string"]; ok {
		res, err := magic(obj, nil)
		if err == nil {
			return res, nil
		}
	}
	return Repr(obj)
}

func ToBool(obj Obj) (Obj, error) {
	if obj.Type == "bool" {
		copy := *obj
		return &copy, nil
	} else if magic, ok := obj.Magic["bool"]; ok {
		res, err := magic(obj, nil)
		if err == nil {
			return res, nil
		}
	}
	return NewBool(true)
}

func ToArray(obj Obj) (Obj, error) {
	if obj.Type == "array" {
		copy := *obj
		return &copy, nil
	} else if magic, ok := obj.Magic["array"]; ok {
		res, err := magic(obj, nil)
		if err == nil {
			return res, nil
		}
	}
	return nil, fmt.Errorf("TypeError: cannot convert type %s to array", obj.Type)
}

func ToMap(obj Obj) (Obj, error) {
	if obj.Type == "map" {
		copy := *obj
		return &copy, nil
	} else if magic, ok := obj.Magic["map"]; ok {
		res, err := magic(obj, nil)
		if err == nil {
			return res, nil
		}
	}
	return nil, fmt.Errorf("TypeError: cannot convert type %s to map", obj.Type)
}

func Repr(obj Obj) (Obj, error) {
	if magic, ok := obj.Magic["repr"]; ok {
		res, err := magic(obj, nil)
		if err == nil {
			return res, nil
		}
	}
	return NewString(fmt.Sprintf("<%s object at %p>", obj.Type, obj))
}

func Length(obj Obj) (Obj, error) {
	if magic, ok := obj.Magic["length"]; ok {
		res, err := magic(obj, nil)
		if err == nil {
			return res, nil
		}
	}
	return nil, fmt.Errorf("TypeError: object of type %s has no length", obj.Type)
}

func CopyTo(dest Obj, src Obj) {
	// dest.Type = src.Type
	// dest.Value = src.Value
	// dest.Magic = src.Magic
	*dest = *src
}

func SafeCopyTo(dest Obj, src Obj) error {
	if dest.Type == src.Type {
		*dest = *src
		return nil
	}
	return fmt.Errorf("TypeError: cannot copy from type %s to type %s", src.Type, dest.Type)
}

func Copy(obj Obj) Obj {
	newObj := *obj
	return &newObj
}

func DeepCopy(obj Obj) Obj {
	return &MeowppObject{
		Type:  obj.Type,
		Value: obj.Value,
		Magic: obj.Magic,
	}
}

func Equal(a Obj, b Obj) (Obj, error) {
	magic, ok := a.Magic["op=="]
	if ok {
		res, err := magic(a, []Obj{b})
		if err == nil {
			return res, nil
		}
	}
	magic, ok = b.Magic["op=="]
	if ok {
		res, err := magic(b, []Obj{a})
		if err == nil {
			return res, nil
		}
	}
	return NewBool(a == b)
}

func NotEqual(a Obj, b Obj) (Obj, error) {
	magic, ok := a.Magic["op!="]
	if ok {
		res, err := magic(a, []Obj{b})
		if err == nil {
			return res, nil
		}
	}
	magic, ok = b.Magic["op!="]
	if ok {
		res, err := magic(b, []Obj{a})
		if err == nil {
			return res, nil
		}
	}
	eqRes, err := Equal(a, b)
	if err != nil {
		return nil, err
	}
	return NewBool(!eqRes.Value.(bool))
}

func IsInteger(obj Obj) bool {
	if obj.Type == "number" {
		num := obj.Value.(float64)
		return num == math.Trunc(num)
	}
	return false
}
