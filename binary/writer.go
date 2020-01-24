package binary

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"

	"github.com/gtlang/gt/core"
)

func Write(w io.Writer, p *core.Program) error {
	key := byte(5 + rand.Intn(255-5))

	if err := binary.Write(w, binary.BigEndian, int32(key)); err != nil {
		return err
	}

	if err := writeString(w, header, key); err != nil {
		return err
	}

	if err := writeDirectives(w, p.Directives, key); err != nil {
		return err
	}

	if err := writeFunctions(w, p.Functions, key); err != nil {
		return err
	}

	if err := writeConstants(w, p.Constants, key); err != nil {
		return err
	}

	if err := writeFiles(w, p.Files, key); err != nil {
		return err
	}

	if err := writeResources(w, p.Resources, key); err != nil {
		return err
	}

	if err := writeSection(w, section_EOF, 0); err != nil {
		return err
	}

	return nil
}

func writeResources(w io.Writer, resources map[string][]byte, key byte) error {
	if err := writeSection(w, section_resources, len(resources)); err != nil {
		return err
	}

	for k, v := range resources {
		if err := writeString(w, k, key); err != nil {
			return err
		}
		if err := writeBytes(w, v); err != nil {
			return err
		}
	}

	return nil
}

func writeFiles(w io.Writer, files []string, key byte) error {
	if err := writeSection(w, section_files, len(files)); err != nil {
		return err
	}
	for _, f := range files {
		if err := writeString(w, f, key); err != nil {
			return err
		}
	}
	return nil
}

func writeConstants(w io.Writer, constants []core.Value, key byte) error {
	if err := writeSection(w, section_constants, len(constants)); err != nil {
		return err
	}

	for _, k := range constants {
		switch k.Type {
		case core.Int:
			if err := writeSection(w, section_kInt, 0); err != nil {
				return err
			}
			if err := binary.Write(w, binary.BigEndian, k.ToInt()); err != nil {
				return err
			}

		case core.Float:
			if err := writeSection(w, section_kFloat, 0); err != nil {
				return err
			}
			if err := binary.Write(w, binary.BigEndian, k.ToFloat()); err != nil {
				return err
			}

		case core.Bool:
			if err := writeSection(w, section_kBool, 0); err != nil {
				return err
			}
			if err := binary.Write(w, binary.BigEndian, k.ToBool()); err != nil {
				return err
			}

		case core.String:
			b := []byte(k.ToString())
			xor(b, key)
			if err := writeSection(w, section_kString, len(b)); err != nil {
				return err
			}
			if err := binary.Write(w, binary.BigEndian, b); err != nil {
				return err
			}

		case core.Null:
			if err := writeSection(w, section_kNull, 0); err != nil {
				return err
			}

		case core.Undefined:
			if err := writeSection(w, section_kUndefined, 0); err != nil {
				return err
			}

		case core.Rune:
			if err := writeSection(w, section_kRune, 0); err != nil {
				return err
			}
			if err := binary.Write(w, binary.BigEndian, int64(k.ToRune())); err != nil {
				return err
			}

		default:
			return fmt.Errorf("invalid constant type: %v", k.Type)
		}
	}
	return nil
}

func writeFunctions(w io.Writer, funcs []*core.Function, key byte) error {
	if err := writeSection(w, section_functions, len(funcs)); err != nil {
		return err
	}

	for _, f := range funcs {
		if err := binary.Write(w, binary.BigEndian, int32(f.Index)); err != nil {
			return err
		}
		if err := writeString(w, f.Name, key); err != nil {
			return err
		}
		if err := writeBool(w, f.Variadic); err != nil {
			return err
		}
		if err := writeBool(w, f.Exported); err != nil {
			return err
		}
		if err := writeBool(w, f.IsClass); err != nil {
			return err
		}
		if err := writeBool(w, f.IsGlobal); err != nil {
			return err
		}
		if err := writeInt32(w, f.Arguments); err != nil {
			return err
		}
		if err := writeInt32(w, f.MaxRegIndex); err != nil {
			return err
		}
		if err := writeRegisters(w, f.Registers, key); err != nil {
			return err
		}
		if err := writeRegisters(w, f.Closures, key); err != nil {
			return err
		}
		if err := writeInstructions(w, f.Instructions, key); err != nil {
			return err
		}
		if err := writePositions(w, f.Positions); err != nil {
			return err
		}
	}

	return nil
}

func writeInstructions(w io.Writer, ins []*core.Instruction, key byte) error {
	if err := writeSection(w, section_instructions, len(ins)); err != nil {
		return err
	}

	for _, i := range ins {
		if err := binary.Write(w, binary.BigEndian, byte(i.Opcode)); err != nil {
			return err
		}
		if err := writeAddress(w, i.A, key); err != nil {
			return err
		}
		if err := writeAddress(w, i.B, key); err != nil {
			return err
		}
		if err := writeAddress(w, i.C, key); err != nil {
			return err
		}
	}

	return nil
}

func writeAddress(w io.Writer, a *core.Address, key byte) error {
	if err := binary.Write(w, binary.BigEndian, byte(a.Kind)); err != nil {
		return err
	}

	if a.Kind == core.AddrNativeFunc {
		f := core.NativeFuncFromIndex(int(a.Value))
		if err := writeString(w, f.Name, key); err != nil {
			return err
		}
	} else {
		if err := binary.Write(w, binary.BigEndian, a.Value); err != nil {
			return err
		}
	}

	return nil
}

func writePositions(w io.Writer, positions []core.Position) error {
	if err := writeSection(w, section_positions, len(positions)); err != nil {
		return err
	}
	for _, pos := range positions {
		if err := binary.Write(w, binary.BigEndian, int32(pos.File)); err != nil {
			return err
		}
		if err := binary.Write(w, binary.BigEndian, int32(pos.Line)); err != nil {
			return err
		}
	}
	return nil
}

func writeRegisters(w io.Writer, regs []*core.Register, key byte) error {
	if err := writeSection(w, section_registers, len(regs)); err != nil {
		return err
	}
	for _, r := range regs {
		if err := writeString(w, r.Name, key); err != nil {
			return err
		}
		if err := binary.Write(w, binary.BigEndian, int32(r.Index)); err != nil {
			return err
		}
		if err := binary.Write(w, binary.BigEndian, int32(r.StartPC)); err != nil {
			return err
		}
		if err := binary.Write(w, binary.BigEndian, int32(r.EndPC)); err != nil {
			return err
		}
		if err := writeBool(w, r.Exported); err != nil {
			return err
		}
	}
	return nil
}

func writeDirectives(w io.Writer, directives map[string]string, key byte) error {
	if err := writeSection(w, section_directives, len(directives)); err != nil {
		return err
	}
	for k, v := range directives {
		if err := writeString(w, k, key); err != nil {
			return err
		}
		if err := writeString(w, v, key); err != nil {
			return err
		}
	}

	return nil
}

func writeString(w io.Writer, v string, key byte) error {
	b := []byte(v)

	xor(b, key)

	if err := writeSection(w, section_string, len(b)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, b); err != nil {
		return err
	}
	return nil
}

func writeBytes(w io.Writer, v []byte) error {
	if err := writeSection(w, section_bytes, len(v)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, v); err != nil {
		return err
	}
	return nil
}

func writeSection(w io.Writer, sType SectionType, v int) error {
	s := newSection(sType, v)
	return binary.Write(w, binary.BigEndian, int64(s))
}

func writeBool(w io.Writer, b bool) error {
	var v int8
	if b {
		v = 1
	}
	return binary.Write(w, binary.BigEndian, v)
}

func writeInt32(w io.Writer, i int) error {
	return binary.Write(w, binary.BigEndian, int32(i))
}

func xor(b []byte, key byte) {
	for i, j := range b {
		b[i] = j ^ key
	}
}
