package core

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"unicode/utf8"
)

type Value struct {
	Type   Type
	object interface{}
}

type Type int8

const (
	Null Type = iota
	Undefined
	Int
	Float
	Bool
	Bytes
	String
	Array
	Map
	Func
	NativeFunc
	Rune
	Object
)

func (t Type) String() string {
	switch t {
	case Null:
		return "null"
	case Int:
		return "int"
	case Rune:
		return "rune"
	case Float:
		return "float"
	case Bool:
		return "bool"
	case Bytes:
		return "bytes"
	case String:
		return "string"
	case Array:
		return "array"
	case Map:
		return "map"
	case Object:
		return "object"
	case Func:
		return "function"
	case NativeFunc:
		return "native function"
	case Undefined:
		return "undefined"
	default:
		panic("unknown type: " + strconv.Itoa(int(t)))
	}
}

func NewValue(v interface{}) Value {
	switch t := v.(type) {
	case Value:
		return t
	case nil:
		return NullValue
	case int:
		return NewInt(t)
	case int64:
		return NewInt64(t)
	case float64:
		return NewFloat(t)
	case rune:
		return NewRune(t)
	case bool:
		return NewBool(t)
	case []byte:
		return NewBytes(t)
	case string:
		return NewString(t)
	default:
		panic(fmt.Sprintf("Invalid type %T: %v", t, t))
		//return Object(t)
	}
}

var (
	UndefinedValue = Value{Type: Undefined}
	NullValue      = Value{Type: Null}
	TrueValue      = Value{Type: Bool, object: true}
	FalseValue     = Value{Type: Bool, object: false}
)

func NewInt(v int) Value {
	return Value{Type: Int, object: int64(v)}
}

func NewInt64(v int64) Value {
	return Value{Type: Int, object: int64(v)}
}

func NewRune(v rune) Value {
	return Value{Type: Rune, object: rune(v)}
}

func NewBool(v bool) Value {
	if v {
		return Value{Type: Bool, object: true}
	}
	return Value{Type: Bool, object: false}
}

func NewFloat(v float64) Value {
	return Value{Type: Float, object: v}
}

func NewBytes(v []byte) Value {
	return Value{Type: Bytes, object: v}
}

func NewString(v string) Value {
	return Value{Type: String, object: v}
}

func NewObject(v interface{}) Value {
	return Value{Type: Object, object: v}
}

type NewArrayObject struct {
	Array []Value
}

func NewArray(size int) Value {
	a := NewArrayObject{make([]Value, size)}
	return Value{Type: Array, object: &a}
}

func NewArrayValues(v []Value) Value {
	a := NewArrayObject{v}
	return Value{Type: Array, object: &a}
}

type MapValue struct {
	Map   map[string]Value
	Mutex *sync.RWMutex
}

func newMapValue(m map[string]Value) *MapValue {
	return &MapValue{
		Map:   m,
		Mutex: &sync.RWMutex{},
	}
}

func NewMap(size int) Value {
	o := newMapValue(make(map[string]Value, size))
	return Value{Type: Map, object: o}
}

func NewMapValues(m map[string]Value) Value {
	o := newMapValue(m)
	return Value{Type: Map, object: o}
}

func NewFunction(v int) Value {
	return Value{Type: Func, object: int64(v)}
}

func NewNativeFunction(v int) Value {
	return Value{Type: NativeFunc, object: int64(v)}
}

// Convert the object to a string
func (v Value) ToString() string {
	switch v.Type {
	case String:
		return v.object.(string)
	case Rune:
		return string(v.object.(rune))
	case Null:
		return "null"
	case Undefined:
		return "undefined"
	case Int:
		return strconv.FormatInt(v.object.(int64), 10)
	case Float:
		return fmt.Sprint(v.object)
	case Bool:
		if v.object.(bool) {
			return "true"
		}
		return "false"
	case Bytes:
		return string(v.object.([]byte))
	case Object:
		st, ok := v.object.(fmt.Stringer)
		if ok {
			return st.String()
		}
		return fmt.Sprintf("%T", v.object)
	default:
		return fmt.Sprintf("%T", v.object)
	}
}

func (v Value) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.Export(0))
}

func (v Value) TypeName() string {
	t := v.Type
	switch t {
	case Object:
		if o, ok := v.ToObject().(NamedType); ok {
			return o.Type()
		}

		if _, ok := v.object.([]interface{}); ok {
			return "array"
		}
	}
	return t.String()
}

func (v Value) ToInt() int64 {
	switch v.Type {
	case Int:
		return v.object.(int64)
	case Float:
		return int64(v.object.(float64))
	case Rune:
		return int64(v.ToRune())
	case Bool:
		if v.ToBool() {
			return 1
		}
		return 0
	default:
		panic(fmt.Sprintf("Invalid conversion to int: %v", v.TypeName()))
	}
}

func (v Value) ToFunction() int {
	switch v.Type {
	case Func:
		return int(v.object.(int64))
	default:
		panic(fmt.Sprintf("Invalid conversion: %v", v))
	}
}

func (v Value) ToNativeFunction() int {
	switch v.Type {
	case NativeFunc:
		return int(v.object.(int64))
	default:
		panic(fmt.Sprintf("Invalid conversion: %v", v))
	}
}

func (v Value) ToFloat() float64 {
	switch v.Type {
	case Int:
		return float64(v.ToInt())
	case Float:
		return v.object.(float64)
	case Rune:
		return float64(v.ToRune())
	default:
		panic(fmt.Sprintf("Invalid conversion to float: %v", v.TypeName()))
	}
}

func (v Value) ToRune() rune {
	switch v.Type {
	case Rune:
		return v.object.(rune)
	case Int:
		return rune(v.object.(int64))
	default:
		panic(fmt.Sprintf("Invalid conversion: %v", v))
	}
}

func (v Value) ToBool() bool {
	switch v.Type {
	case Bool:
		return v.object.(bool)
	case Int:
		return v.ToInt() > 0
	default:
		panic(fmt.Sprintf("Invalid conversion: %v", v))
	}
}

func (v Value) ToBytes() []byte {
	switch v.Type {
	case String:
		return []byte(v.object.(string))
	case Bytes:
		return v.object.([]byte)
	case Array:
		a := v.ToArray()
		b := make([]byte, len(a))
		for i, v := range a {
			b[i] = byte(v.ToInt())
		}
		return b
	default:
		panic(fmt.Sprintf("Invalid conversion: %v", v))
	}
}

// String representation for the formatter.
func (v Value) String() string {
	switch v.Type {
	case Undefined:
		return "undefined"
	case Null:
		return "null"
	case String:
		s := v.object.(string)
		if len(s) > 200 {
			s = s[:200]
		}
		return s
	case Int:
		return strconv.FormatInt(v.ToInt(), 10)
	case Float:
		f := v.ToFloat()
		if f == 0.0 {
			return "0" // prevent negative zero
		}
		return strconv.FormatFloat(v.ToFloat(), 'f', 6, 64)
	case Bool:
		if v.ToBool() {
			return "true"
		}
		return "false"
	case Rune:
		return string(v.ToRune())
	case Func:
		return "[function]"
	case NativeFunc:
		return "[native function]"
	case Bytes:
		s := string(v.object.([]byte))
		if len(s) > 30 {
			s = s[:30]
		}
		return s
	case Map:
		return "[map]"
	case Object:
		if stg, ok := v.Export(0).(fmt.Stringer); ok {
			return stg.String()
		}
		if n, ok := v.object.(NamedType); ok {
			return n.Type()
		}
		return fmt.Sprintf("%T", v.object)
	default:
		return fmt.Sprintf("%v", v.Type)
	}
}

func (v Value) ToArray() []Value {
	switch v.Type {
	case Bytes:
		a := v.object.([]byte)
		b := make([]Value, len(a))
		for i, v := range a {
			b[i] = NewInt(int(v))
		}
		return b
	case Array:
		return v.object.(*NewArrayObject).Array
	default:
		panic(fmt.Sprintf("Invalid conversion: %v", v))
	}
}

func (v Value) ToArrayObject() *NewArrayObject {
	if v.Type != Array {
		panic(fmt.Sprintf("Invalid conversion: %v", v))
	}
	return v.object.(*NewArrayObject)
}

func (v Value) ToMap() *MapValue {
	if v.Type != Map {
		panic(fmt.Sprintf("Invalid conversion: %v", v))
	}
	return v.object.(*MapValue)
}

func (v Value) ToObject() interface{} {
	if v.Type != Object {
		panic(fmt.Sprintf("Invalid conversion: %v", v))
	}
	return v.object
}
func (v Value) ToObjectOrNil() interface{} {
	if v.Type != Object {
		return nil
	}
	return v.object
}

func (v Value) Size() int {
	switch v.Type {
	case String:
		return len(v.object.(string))
	default:
		// because of recursive dangers: an array or map that contains itself
		return 1
	}

	// switch v.Type {
	// case NullType, IntType, FloatType, BoolType, FunctionType, NativeFunctionType,
	// 	UndefinedType, RuneType:
	// 	return 1
	// case StringType:
	// 	return len(v.ToString())
	// case BytesType:
	// 	return len(v.ToBytes())
	// case ArrayType:
	// 	var i int
	// 	for _, v := range v.ToArray() {
	// 		i += v.Size()
	// 	}
	// 	return i
	// case MapType:
	// 	var i int
	// 	m := v.ToMap()
	// 	m.Mutex.RLock()
	// 	for _, v := range m.Map {
	// 		i += v.Size()
	// 	}
	// 	m.Mutex.RUnlock()
	// 	return i
	// case ObjectType:
	// 	if a, ok := v.Obj.(Allocator); ok {
	// 		return a.Size()
	// 	}
	// 	return 1
	// default:
	// 	panic("unknown type: " + v.Type.String())
	// }
}

const MAX_EXPORT_RECURSION = 200

func (v Value) Export(recursionLevel int) interface{} {
	if recursionLevel > MAX_EXPORT_RECURSION {
		fmt.Println("[Export Error: max recursion exceeded]", recursionLevel)
		return "[Export Error: max recursion exceeded]"
	}
	recursionLevel++

	switch v.Type {
	case Null:
		return nil
	case Undefined:
		return nil
	case Int:
		return v.ToInt()
	case Rune:
		return string(v.ToRune())
	case Float:
		return v.ToFloat()
	case Bool:
		return v.ToBool()
	case String:
		return v.ToString()
	case Bytes:
		return v.ToBytes()
	case Array:
		o := v.ToArray()
		m := make([]interface{}, len(o))
		for i, v := range o {
			m[i] = v.Export(recursionLevel)
		}
		return m
	case Map:
		om := v.ToMap()
		om.Mutex.RLock()
		o := om.Map
		m := make(map[string]interface{}, len(o))
		for k, v := range o {
			m[k] = v.Export(recursionLevel)
		}
		om.Mutex.RUnlock()
		return m
	case Object:
		if o, ok := v.object.(Exporter); ok {
			return o.Export(recursionLevel)
		}
		return v.object
	case Func:
		return fmt.Sprintf("[function(%d)]", v.ToFunction())
	case NativeFunc:
		return fmt.Sprintf("[native function(%d)]", v.ToFunction())
	default:
		panic("unknown type: " + v.Type.String())
	}
}

func (v Value) IsNil() bool {
	switch v.Type {
	case Null:
		return true
	case Undefined:
		return true
	case Object:
		return v.object == nil
	default:
		return false
	}
}

func (v Value) IsNilOrEmpty() bool {
	switch v.Type {
	case Null:
		return true
	case Undefined:
		return true
	case String:
		return v.object == ""
	case Object:
		return v.object == nil
	default:
		return false
	}
}

func (v Value) StrictEquals(other Value) bool {
	t1 := v.Type
	t2 := other.Type

	if t1 != t2 {
		return false
	}

	switch t1 {
	case Int:
		return v.ToInt() == other.ToInt()
	case Float:
		return v.ToFloat() == other.ToFloat()
	case Bool:
		return v.ToBool() == other.ToBool()
	case Rune:
		return v.ToRune() == other.ToRune()
	case Func:
		return v.ToFunction() == other.ToFunction()
	case NativeFunc:
		return v.ToNativeFunction() == other.ToNativeFunction()
	case String:
		return v.ToString() == other.ToString()
	case Object:
		return v.object == other.object
	case Null, Undefined:
		return true
	default:
		return false
	}
}

func (v Value) Equals(other Value) bool {
	t1 := v.Type
	t2 := other.Type

	if t1 != t2 {
		switch t1 {

		case Object:
			switch t2 {
			case Null:
				return v.object == nil
			default:
				return false
			}

		case Null:
			switch t2 {
			case Object:
				return other.object == nil
			case Undefined:
				return true
			default:
				return false
			}

		case Undefined:
			switch t2 {
			case Object:
				return other.object == nil
			case Null:
				return true
			default:
				return false
			}

		case Int:
			switch t2 {
			case Float, Rune:
			case Bool:
				a := v.ToInt()
				b := other.ToBool()
				return b && a == 1 || !b && a == 0
			default:
				return false
			}

		case Bool:
			switch t2 {
			case Bool:
				return v.ToBool() == other.ToBool()
			case Int:
				a := v.ToBool()
				b := other.ToInt()
				return a && b == 1 || !a && b == 0
			default:
				return false
			}

		case Float:
			// allow to continue if is int
			if t2 != Int {
				return false
			}

		case String:
			if t2 == Rune {
				s1 := v.ToString()
				if utf8.RuneCountInString(s1) == 1 {
					return s1 == other.ToString()
				}
				return false
			}

		case Rune:
			if t2 == String {
				s2 := other.ToString()
				if utf8.RuneCountInString(s2) == 1 {
					return s2 == v.ToString()
				}
				return false
			}

		default:
			return false
		}
	}

	switch t1 {
	case Int, Float:
		return v.ToFloat() == other.ToFloat()
	case Bool:
		return v.ToBool() == other.ToBool()
	case Rune:
		return v.ToRune() == other.ToRune()
	case String:
		return v.ToString() == other.ToString()
	case Object:
		return v.object == other.object
	case Null, Undefined:
		return true
	default:
		return false
	}
}

type Callable interface {
	GetMethod(name string) NativeMethod
}

type PropertyGetter interface {
	GetProperty(string, *VM) (Value, error)
}

type PropertySetter interface {
	SetProperty(string, Value, *VM) error
}

type IndexerGetter interface {
	GetIndex(int) (Value, error)
}

type KeyGetter interface {
	GetKey(string) (Value, error)
}

type IndexerSetter interface {
	SetIndex(int, Value) error
}

type IterableByIndex interface {
	IndexerGetter
	Len() int
}

type IterableByKey interface {
	KeyGetter
	Keys() []string
}

type Enumerable interface {
	Values() ([]Value, error)
}

type Allocator interface {
	Size() int
}

type Exporter interface {
	Export(recursionLevel int) interface{}
}

type NamedType interface {
	Type() string
}

type Finalizable interface {
	Close() error
}
