package lib

import (
	"fmt"
	"io"
	"github.com/gtlang/gt/core"
	"time"

	"github.com/scorredoira/xlsx"
)

func init() {
	core.RegisterLib(XLSX, `


declare namespace xlsx {
    export function newFile(): XLSXFile
    export function openFile(path: string): XLSXFile
    export function openFile(file: io.File): XLSXFile
    export function openReaderAt(r: io.ReaderAt, size: number): XLSXFile 
    export function openBinary(file: io.File): XLSXFile
    export function newStyle(): Style

    export interface XLSXFile {
        sheets: XLSXSheet[]
        addSheet(name: string): XLSXSheet
        save(path?: string): void
        write(w: io.Writer): void
    }

    export interface XLSXSheet {
        rows: XLSXRow[]
        col(i: number): Col
        addRow(): XLSXRow
    }

    export interface Col {
        width: number
    }

    export interface XLSXRow {
        cells: XLSXCell[]
        height: number
        addCell(v?: any): XLSXCell
    }

    export interface XLSXCell {
        value: any
        numberFormat: string
        style: Style
        getDate(): time.Time
        merge(hCells: number, vCells: number): void
    }

    export interface Style {
        alignment: Alignment
        applyAlignment: boolean
        font: Font
        applyFont: boolean
    }

    export interface Alignment {
        horizontal: string
        vertical: string
    }

    export interface Font {
        bold: boolean
        size: number
    }
}

`)
}

var XLSX = []core.NativeFunction{
	core.NativeFunction{
		Name:      "xlsx.openReaderAt",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Object, core.Int); err != nil {
				return core.NullValue, err
			}

			r, ok := args[0].ToObjectOrNil().(io.ReaderAt)
			if !ok {
				return core.NullValue, fmt.Errorf("invalid argument type. Expected a io.ReaderAt, got %s", args[0].TypeName())
			}

			size := args[1].ToInt()

			reader, err := xlsx.OpenReaderAt(r, size)
			if err != nil {
				return core.NullValue, err
			}

			return core.NewObject(&xlsxFile{obj: reader}), nil
		},
	},
	core.NativeFunction{
		Name:      "xlsx.openBinary",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.Bytes); err != nil {
				return core.NullValue, err
			}

			b := args[0].ToBytes()

			reader, err := xlsx.OpenBinary(b)
			if err != nil {
				return core.NullValue, err
			}

			return core.NewObject(&xlsxFile{obj: reader}), nil
		},
	},
	core.NativeFunction{
		Name:      "xlsx.openFile",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			var r io.ReaderAt
			var size int64

			a := args[0]

			switch a.Type {
			case core.Object:
				f, ok := a.ToObject().(*file)
				if !ok {
					return core.NullValue, fmt.Errorf("invalid argument type. Expected a io.ReaderAt, got %s", a.TypeName())
				}
				r = f
				st, err := f.Stat()
				if err != nil {
					return core.NullValue, err
				}
				size = st.Size()

			case core.String:
				f, err := vm.FileSystem.Open(a.ToString())
				if err != nil {
					return core.NullValue, err
				}
				r = f
				st, err := f.Stat()
				if err != nil {
					return core.NullValue, err
				}
				size = st.Size()

			default:
				return core.NullValue, fmt.Errorf("invalid argument type. Expected a io.ReaderAt, got %s", a.TypeName())
			}

			reader, err := xlsx.OpenReaderAt(r, size)
			if err != nil {
				return core.NullValue, err
			}

			return core.NewObject(&xlsxFile{obj: reader, path: a.ToString()}), nil
		},
	},
	core.NativeFunction{
		Name:      "xlsx.newFile",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args); err != nil {
				return core.NullValue, err
			}

			file := xlsx.NewFile()
			return core.NewObject(&xlsxFile{obj: file}), nil
		},
	},
	core.NativeFunction{
		Name:      "xlsx.newStyle",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args); err != nil {
				return core.NullValue, err
			}

			s := xlsx.NewStyle()
			return core.NewObject(&xlsxStyle{obj: s}), nil
		},
	},
}

type xlsxFile struct {
	path string
	obj  *xlsx.File
}

func (f *xlsxFile) Type() string {
	return "xlsx.Reader"
}

func (x *xlsxFile) GetProperty(key string, vm *core.VM) (core.Value, error) {
	switch key {
	case "sheets":
		xSheets := x.obj.Sheets
		sheets := make([]core.Value, len(xSheets))
		for i, c := range xSheets {
			sheets[i] = core.NewObject(&xlsxSheet{obj: c})
		}
		return core.NewArrayValues(sheets), nil
	}
	return core.UndefinedValue, nil
}

func (x *xlsxFile) GetMethod(name string) core.NativeMethod {
	switch name {
	case "addSheet":
		return x.addSheet
	case "save":
		return x.save
	case "write":
		return x.write
	}
	return nil
}

func (x *xlsxFile) addSheet(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	name := args[0].ToString()
	xSheet, err := x.obj.AddSheet(name)
	if err != nil {
		return core.NullValue, err
	}
	return core.NewObject(&xlsxSheet{obj: xSheet}), nil
}

func (x *xlsxFile) save(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateOptionalArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	var path string
	if len(args) == 1 {
		path = args[0].ToString()
	} else {
		path = x.path
	}

	if path == "" {
		return core.NullValue, fmt.Errorf("need a name to save the file")
	}

	f, err := vm.FileSystem.OpenForWrite(path)
	if err != nil {
		return core.NullValue, err
	}

	if err := x.obj.Write(f); err != nil {
		return core.NullValue, err
	}

	err = f.Close()
	return core.NullValue, err
}

func (x *xlsxFile) write(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateOptionalArgs(args, core.Object); err != nil {
		return core.NullValue, err
	}

	w, ok := args[0].ToObjectOrNil().(io.Writer)
	if !ok {
		return core.NullValue, fmt.Errorf("invalid argument type. Expected a io.Writer, got %s", args[0].TypeName())
	}

	err := x.obj.Write(w)

	return core.NullValue, err
}

type xlsxSheet struct {
	obj  *xlsx.Sheet
	rows []core.Value // important to cache this for large files
}

func (x *xlsxSheet) Type() string {
	return "xlsx.Sheet"
}

func (x *xlsxSheet) GetProperty(key string, vm *core.VM) (core.Value, error) {
	switch key {
	case "rows":
		if x.rows == nil {
			xRows := x.obj.Rows
			rows := make([]core.Value, len(xRows))
			for i, c := range xRows {
				rows[i] = core.NewObject(&xlsxRow{c})
			}
			x.rows = rows
		}
		return core.NewArrayValues(x.rows), nil
	}
	return core.UndefinedValue, nil
}

func (x *xlsxSheet) GetMethod(name string) core.NativeMethod {
	switch name {
	case "addRow":
		return x.addRow
	case "col":
		return x.col
	}
	return nil
}

func (x *xlsxSheet) col(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Int); err != nil {
		return core.NullValue, err
	}
	i := args[0].ToInt()
	col := x.obj.Col(int(i))
	return core.NewObject(&xlsxCol{col}), nil
}

func (x *xlsxSheet) addRow(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}
	xRow := x.obj.AddRow()
	return core.NewObject(&xlsxRow{xRow}), nil
}

type xlsxRow struct {
	obj *xlsx.Row
}

func (x *xlsxRow) Type() string {
	return "xlsx.Row"
}

func (x *xlsxRow) GetProperty(key string, vm *core.VM) (core.Value, error) {
	switch key {
	case "cells":
		xCells := x.obj.Cells
		cells := make([]core.Value, len(xCells))
		for i, c := range xCells {
			cells[i] = core.NewObject(&xlsxCell{c})
		}
		return core.NewArrayValues(cells), nil
	case "height":
		return core.NewFloat(x.obj.Height), nil
	}
	return core.UndefinedValue, nil
}

func (x *xlsxRow) SetProperty(name string, v core.Value, vm *core.VM) error {
	switch name {
	case "height":
		switch v.Type {
		case core.Float:
		case core.Int:
			x.obj.SetHeight(v.ToFloat())
			return nil
		default:
			return ErrInvalidType
		}
	}

	return ErrReadOnlyOrUndefined
}

func (x *xlsxRow) GetMethod(name string) core.NativeMethod {
	switch name {
	case "addCell":
		return x.addCell
	}
	return nil
}

func (x *xlsxRow) addCell(args []core.Value, vm *core.VM) (core.Value, error) {
	l := len(args)

	if l > 1 {
		return core.NullValue, fmt.Errorf("expected 0 or 1 arguments, got %d", l)
	}

	xCell := x.obj.AddCell()
	cell := &xlsxCell{xCell}

	if l == 1 {
		if err := cell.setValue(args[0], vm); err != nil {
			return core.NullValue, err
		}
	}

	return core.NewObject(cell), nil
}

type xlsxCell struct {
	obj *xlsx.Cell
}

func (x *xlsxCell) Type() string {
	return "xlsx.Cell"
}

func (x *xlsxCell) GetMethod(name string) core.NativeMethod {
	switch name {
	case "getDate":
		return x.getDate
	case "merge":
		return x.merge
	}
	return nil
}

func (x *xlsxCell) merge(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Int, core.Int); err != nil {
		return core.NullValue, err
	}
	x.obj.Merge(int(args[0].ToInt()), int(args[1].ToInt()))
	return core.NullValue, nil
}

func (x *xlsxCell) getDate(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}
	cell := x.obj
	t, err := cell.GetTime(false)
	if err != nil {
		if cell.Value == "" {
			return core.NullValue, nil
		}
		return core.NullValue, err
	}

	// many times the value es like: 2019-09-28 08:17:59.999996814 +0000 UTC
	t = t.Round(time.Millisecond)

	loc := GetContext(vm).GetLocation()

	t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, loc)

	return core.NewObject(TimeObj(t)), nil
}

func (x *xlsxCell) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {

	case "numberFormat":
		return core.NewString(x.obj.GetNumberFormat()), nil

	case "value":
		cell := x.obj
		if cell.Value == "" {
			return core.NullValue, nil
		}
		switch cell.Type() {
		case xlsx.CellTypeNumeric:
			f, err := cell.Float()
			if err != nil {
				return core.NullValue, err
			}
			i := int64(f)
			if f == float64(i) {
				return core.NewInt64(i), nil
			}
			return core.NewFloat(f), nil

		case xlsx.CellTypeBool:
			return core.NewBool(cell.Bool()), nil

		case xlsx.CellTypeDate:
			t, err := cell.GetTime(false)
			if err != nil {
				return core.NullValue, err
			}
			return core.NewObject(TimeObj(t)), nil

		default:
			return core.NewString(cell.String()), nil
		}

	case "style":
		return core.NewObject(&xlsxStyle{x.obj.GetStyle()}), nil
	}
	return core.UndefinedValue, nil
}

func (x *xlsxCell) SetProperty(name string, v core.Value, vm *core.VM) error {
	switch name {
	case "error":
		return x.setValue(v, vm)
	case "style":
		s, ok := v.ToObjectOrNil().(*xlsxStyle)
		if !ok {
			return ErrInvalidType
		}
		x.obj.SetStyle(s.obj)
		return nil
	}

	return ErrReadOnlyOrUndefined
}

func (x *xlsxCell) setValue(v core.Value, vm *core.VM) error {
	switch v.Type {
	case core.Int:
		x.obj.SetInt64(v.ToInt())
	case core.Float:
		x.obj.SetFloat(v.ToFloat())
	case core.Bool:
		x.obj.SetInt64(v.ToInt())
	case core.String, core.Rune, core.Bytes:
		x.obj.SetString(v.ToString())
	case core.Object:
		switch t := v.ToObject().(type) {
		case TimeObj:
			loc := GetContext(vm).GetLocation()
			x.obj.SetDateWithOptions(time.Time(t), xlsx.DateTimeOptions{
				Location:        loc,
				ExcelTimeFormat: xlsx.DefaultDateTimeFormat,
			})
		}
	default:
		return ErrInvalidType
	}

	return nil
}

type xlsxCol struct {
	obj *xlsx.Col
}

func (x *xlsxCol) Type() string {
	return "xlsx.Col"
}

func (x *xlsxCol) GetProperty(key string, vm *core.VM) (core.Value, error) {
	switch key {
	case "width":
		return core.NewFloat(x.obj.Width), nil
	}
	return core.UndefinedValue, nil
}

func (x *xlsxCol) SetProperty(name string, v core.Value, vm *core.VM) error {
	switch name {
	case "width":
		switch v.Type {
		case core.Float:
		case core.Int:
			x.obj.Width = v.ToFloat()
			return nil
		default:
			return ErrInvalidType
		}
	}

	return ErrReadOnlyOrUndefined
}

type xlsxStyle struct {
	obj *xlsx.Style
}

func (x *xlsxStyle) Type() string {
	return "xlsx.Style"
}

func (x *xlsxStyle) GetProperty(key string, vm *core.VM) (core.Value, error) {
	switch key {
	case "alignment":
		return core.NewObject(&xlsxAlignment{&x.obj.Alignment}), nil
	case "applyAlignment":
		return core.NewBool(x.obj.ApplyAlignment), nil
	case "applyFont":
		return core.NewBool(x.obj.ApplyFont), nil
	case "font":
		return core.NewObject(&xlsxFont{&x.obj.Font}), nil
	}
	return core.UndefinedValue, nil
}

func (x *xlsxStyle) SetProperty(name string, v core.Value, vm *core.VM) error {
	switch name {
	case "alignment":
		a, ok := v.ToObjectOrNil().(*xlsxAlignment)
		if !ok {
			return ErrInvalidType
		}
		x.obj.Alignment = *a.obj
		return nil
	case "applyAlignment":
		if v.Type != core.Bool {
			return ErrInvalidType
		}
		x.obj.ApplyAlignment = v.ToBool()
		return nil
	case "font":
		a, ok := v.ToObjectOrNil().(*xlsxFont)
		if !ok {
			return ErrInvalidType
		}
		x.obj.Font = *a.obj
		return nil
	case "applyFont":
		if v.Type != core.Bool {
			return ErrInvalidType
		}
		x.obj.ApplyFont = v.ToBool()
		return nil
	}

	return ErrReadOnlyOrUndefined
}

type xlsxAlignment struct {
	obj *xlsx.Alignment
}

func (x *xlsxAlignment) Type() string {
	return "xlsx.Alignment"
}

func (x *xlsxAlignment) GetProperty(key string, vm *core.VM) (core.Value, error) {
	switch key {
	case "horizontal":
		return core.NewString(x.obj.Horizontal), nil
	case "vertical":
		return core.NewString(x.obj.Vertical), nil
	}
	return core.UndefinedValue, nil
}

func (x *xlsxAlignment) SetProperty(name string, v core.Value, vm *core.VM) error {
	switch name {
	case "horizontal":
		if v.Type != core.String {
			return ErrInvalidType
		}
		x.obj.Horizontal = v.ToString()
		return nil
	case "vertical":
		if v.Type != core.String {
			return ErrInvalidType
		}
		x.obj.Vertical = v.ToString()
		return nil
	}

	return ErrReadOnlyOrUndefined
}

type xlsxFont struct {
	obj *xlsx.Font
}

func (x *xlsxFont) Type() string {
	return "xlsx.Font"
}

func (x *xlsxFont) GetProperty(key string, vm *core.VM) (core.Value, error) {
	switch key {
	case "bold":
		return core.NewBool(x.obj.Bold), nil
	case "size":
		return core.NewInt(x.obj.Size), nil
	}
	return core.UndefinedValue, nil
}

func (x *xlsxFont) SetProperty(name string, v core.Value, vm *core.VM) error {
	switch name {
	case "bold":
		if v.Type != core.Bool {
			return ErrInvalidType
		}
		x.obj.Bold = v.ToBool()
		return nil
	case "size":
		if v.Type != core.Int {
			return ErrInvalidType
		}
		x.obj.Size = int(v.ToInt())
		return nil
	}

	return ErrReadOnlyOrUndefined
}
