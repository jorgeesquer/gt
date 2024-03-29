package binary

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/gtlang/gt/core"
)

var ErrInvalidHeader = errors.New("invalid header")

func Load(b []byte) (*core.Program, error) {
	r := bytes.NewReader(b)
	return Read(r)
}

func Read(r io.Reader) (*core.Program, error) {
	p := &core.Program{}

	iKey, err := readInt32(r)
	if err != nil {
		return nil, err
	}
	key := byte(iKey)

	s, err := readString(r, key)
	if err != nil {
		return nil, err
	}

	if s != header {
		return nil, ErrInvalidHeader
	}

	if p.Directives, err = readDirectives(r, key); err != nil {
		return nil, err
	}

	if err := readFunctions(r, key, p); err != nil {
		return nil, err
	}

	if p.Constants, err = readConstants(r, key); err != nil {
		return nil, err
	}

	if p.Files, err = readFiles(r, key); err != nil {
		return nil, err
	}

	if p.Resources, err = readResources(r, key); err != nil {
		return nil, err
	}

	if err = readEOF(r); err != nil {
		return nil, err
	}

	return p, nil
}

func unxor(b []byte, key byte) {
	for i, j := range b {
		b[i] = j ^ key
	}
}

func readBool(r io.Reader) (bool, error) {
	var i int8
	err := binary.Read(r, binary.BigEndian, &i)
	if err != nil {
		return false, err
	}

	switch i {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, fmt.Errorf("invalid bool value: %d", i)
	}
}

func readInt32(r io.Reader) (int, error) {
	var i int32
	err := binary.Read(r, binary.BigEndian, &i)
	return int(i), err
}

func readInt64(r io.Reader) (int64, error) {
	var i int64
	err := binary.Read(r, binary.BigEndian, &i)
	return i, err
}

func readFloat64(r io.Reader) (float64, error) {
	var i float64
	err := binary.Read(r, binary.BigEndian, &i)
	return i, err
}

func readString(r io.Reader, key byte) (string, error) {
	s, err := readSection(r)
	if err != nil {
		return "", err
	}
	t, v := s.values()
	if t != section_string {
		return "", fmt.Errorf("invalid section, expected %v, got %v", section_string, t)
	}

	p := make([]byte, v)
	if err := binary.Read(r, binary.BigEndian, &p); err != nil {
		return "", err
	}

	unxor(p, key)
	return string(p), err
}

func readBytes(r io.Reader) ([]byte, error) {
	s, err := readSection(r)
	if err != nil {
		return nil, err
	}
	t, v := s.values()
	if t != section_bytes {
		return nil, fmt.Errorf("invalid section, expected %v, got %v", section_bytes, t)
	}

	p := make([]byte, v)
	if err := binary.Read(r, binary.BigEndian, &p); err != nil {
		return nil, err
	}

	return p, err
}

func readSection(r io.Reader) (section, error) {
	i, err := readInt64(r)
	if err != nil {
		return 0, fmt.Errorf("error reading section: %v", err)
	}
	return section(i), nil
}

func readDirectives(r io.Reader, key byte) (map[string]string, error) {
	s, err := readSection(r)
	if err != nil {
		return nil, err
	}
	t, v := s.values()
	if t != section_directives {
		return nil, fmt.Errorf("invalid section, expected %v, got %v", section_directives, t)
	}

	directives := make(map[string]string)

	for i, l := 0, int(v); i < l; i++ {
		k, err := readString(r, key)
		if err != nil {
			return nil, err
		}
		value, err := readString(r, key)
		if err != nil {
			return nil, err
		}
		directives[k] = value
	}

	return directives, nil
}

func readFunctions(r io.Reader, key byte, p *core.Program) error {
	s, err := readSection(r)
	if err != nil {
		return err
	}
	t, v := s.values()
	if t != section_functions {
		return fmt.Errorf("invalid section, expected %v, got %v", section_functions, t)
	}

	for i, l := 0, int(v); i < l; i++ {
		var err error
		f := &core.Function{}

		if f.Index, err = readInt32(r); err != nil {
			return err
		}
		if f.Name, err = readString(r, key); err != nil {
			return err
		}
		if f.Variadic, err = readBool(r); err != nil {
			return err
		}
		if f.Exported, err = readBool(r); err != nil {
			return err
		}
		if f.IsClass, err = readBool(r); err != nil {
			return err
		}
		if f.IsGlobal, err = readBool(r); err != nil {
			return err
		}
		if f.Arguments, err = readInt32(r); err != nil {
			return err
		}
		if f.MaxRegIndex, err = readInt32(r); err != nil {
			return err
		}
		if f.Registers, err = readRegisters(r, key); err != nil {
			return err
		}
		if f.Closures, err = readRegisters(r, key); err != nil {
			return err
		}
		if f.Instructions, err = readInstructions(r, key); err != nil {
			return err
		}
		if f.Positions, err = readPositions(r); err != nil {
			return err
		}

		p.Functions = append(p.Functions, f)
	}

	return nil
}

func readPositions(r io.Reader) ([]core.Position, error) {
	s, err := readSection(r)
	if err != nil {
		return nil, err
	}

	t, v := s.values()
	if t != section_positions {
		return nil, fmt.Errorf("invalid section, expected %v, got %v", section_positions, t)
	}

	var positions []core.Position

	for i, l := 0, int(v); i < l; i++ {
		file, err := readInt32(r)
		if err != nil {
			return nil, err
		}
		line, err := readInt32(r)
		if err != nil {
			return nil, err
		}
		positions = append(positions, core.Position{File: file, Line: line})
	}

	return positions, nil
}

func readRegisters(r io.Reader, key byte) ([]*core.Register, error) {
	s, err := readSection(r)
	if err != nil {
		return nil, err
	}
	t, v := s.values()
	if t != section_registers {
		return nil, fmt.Errorf("invalid section, expected %v, got %v", section_registers, t)
	}

	var regs []*core.Register

	for i, l := 0, int(v); i < l; i++ {
		reg := &core.Register{}

		if reg.Name, err = readString(r, key); err != nil {
			return nil, err
		}
		if reg.Index, err = readInt32(r); err != nil {
			return nil, err
		}
		if reg.StartPC, err = readInt32(r); err != nil {
			return nil, err
		}
		if reg.EndPC, err = readInt32(r); err != nil {
			return nil, err
		}
		if reg.Exported, err = readBool(r); err != nil {
			return nil, err
		}

		regs = append(regs, reg)
	}
	return regs, nil
}

func readInstructions(r io.Reader, key byte) ([]*core.Instruction, error) {
	s, err := readSection(r)
	if err != nil {
		return nil, err
	}
	t, v := s.values()
	if t != section_instructions {
		return nil, fmt.Errorf("invalid section, expected %v, got %v", section_instructions, t)
	}

	l := int(v)
	instrs := make([]*core.Instruction, l)

	for i, l := 0, int(v); i < l; i++ {
		instr := &core.Instruction{}

		instrs[i] = instr

		if err := binary.Read(r, binary.BigEndian, &instr.Opcode); err != nil {
			return nil, err
		}

		addr, err := readAddress(r, key)
		if err != nil {
			return nil, err
		}
		instr.A = addr

		addr, err = readAddress(r, key)
		if err != nil {
			return nil, err
		}
		instr.B = addr

		addr, err = readAddress(r, key)
		if err != nil {
			return nil, err
		}
		instr.C = addr
	}

	return instrs, nil
}

func readAddress(r io.Reader, key byte) (*core.Address, error) {
	a := &core.Address{}

	if err := binary.Read(r, binary.BigEndian, &a.Kind); err != nil {
		return nil, err
	}

	if a.Kind == core.AddrNativeFunc {
		v, err := readString(r, key)
		if err != nil {
			return nil, err
		}
		f, ok := core.NativeFuncFromName(v)
		if !ok {
			return nil, fmt.Errorf("invalid native function %s", v)
		}
		a.Value = int32(f.Index)
	} else {
		if err := binary.Read(r, binary.BigEndian, &a.Value); err != nil {
			return nil, fmt.Errorf("invalid address: %v", err)
		}
	}

	// the instance must be the same
	// TODO: compare everywhere Kind only
	if a.Kind == core.AddrVoid {
		return core.Void, nil
	}

	return a, nil
}

func readConstants(r io.Reader, key byte) ([]core.Value, error) {
	s, err := readSection(r)
	if err != nil {
		return nil, err
	}
	t, v := s.values()
	if t != section_constants {
		return nil, fmt.Errorf("invalid section, expected %v, got %v", section_constants, t)
	}

	var constants []core.Value

	for i, l := 0, int(v); i < l; i++ {
		s, err := readSection(r)
		if err != nil {
			return nil, err
		}
		t, v := s.values()
		switch t {

		case section_kInt:
			k, err := readInt64(r)
			if err != nil {
				return nil, err
			}
			constants = append(constants, core.NewInt64(k))

		case section_kFloat:
			k, err := readFloat64(r)
			if err != nil {
				return nil, err
			}
			constants = append(constants, core.NewFloat(k))

		case section_kBool:
			k, err := readBool(r)
			if err != nil {
				return nil, err
			}
			constants = append(constants, core.NewBool(k))

		case section_kString:
			p := make([]byte, v)
			if _, err := r.Read(p); err != nil {
				return nil, err
			}
			unxor(p, key)
			constants = append(constants, core.NewString(string(p)))

		case section_kNull:
			constants = append(constants, core.NullValue)

		case section_kUndefined:
			constants = append(constants, core.UndefinedValue)

		case section_kRune:
			i, err := readInt64(r)
			if err != nil {
				return nil, err
			}
			constants = append(constants, core.NewRune(rune(i)))

		default:
			panic(fmt.Sprintf("Invalid constant type: %v", t))

		}
	}
	return constants, nil
}

func readFiles(r io.Reader, key byte) ([]string, error) {
	s, err := readSection(r)
	if err != nil {
		return nil, err
	}
	t, v := s.values()
	if t != section_files {
		return nil, fmt.Errorf("invalid section, expected %v, got %v", section_files, t)
	}

	var files []string

	for i, l := 0, int(v); i < l; i++ {
		s, err := readString(r, key)
		if err != nil {
			return nil, err
		}
		files = append(files, s)
	}

	return files, nil
}

func readResources(r io.Reader, key byte) (map[string][]byte, error) {
	s, err := readSection(r)
	if err != nil {
		return nil, err
	}
	t, v := s.values()
	if t != section_resources {
		return nil, fmt.Errorf("invalid section, expected %v, got %v", section_resources, t)
	}

	resources := make(map[string][]byte)

	for i, l := 0, int(v); i < l; i++ {
		k, err := readString(r, key)
		if err != nil {
			return nil, err
		}
		v, err := readBytes(r)
		if err != nil {
			return nil, err
		}
		resources[k] = v
	}

	return resources, nil
}

func readEOF(r io.Reader) error {
	s, err := readSection(r)
	if err != nil {
		return err
	}
	t, _ := s.values()
	if t != section_EOF {
		return fmt.Errorf("invalid section, expected %v, got %v", section_EOF, t)
	}
	return nil
}
