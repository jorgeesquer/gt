package lib

import (
	"github.com/gtlang/gt/core"

	"github.com/beevik/etree"
)

func init() {
	core.RegisterLib(XML, `

declare namespace xml {
    export function newDocument(): XMLDocument

    export function readString(s: string): XMLDocument

    export interface XMLDocument {
        createElement(name: string): XMLElement
        selectElement(name: string): XMLElement
        toString(): string
    }

    export interface XMLElement {
        tag: string
        selectElements(name: string): XMLElement[]
        selectElement(name: string): XMLElement
        createElement(name: string): XMLElement
        createAttribute(name: string, value: string): XMLElement
        getAttribute(name: string): string
        setValue(value: string | number | boolean): void
        getValue(): string
    }
}


`)
}

var XML = []core.NativeFunction{
	core.NativeFunction{
		Name: "xml.newDocument",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewObject(newXMLDoc()), nil
		},
	},
	core.NativeFunction{
		Name:      "xml.readString",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}

			xml := etree.NewDocument()
			if err := xml.ReadFromString(args[0].ToString()); err != nil {
				return core.NullValue, err
			}
			return core.NewObject(&xmlDoc{xml: xml}), nil
		},
	},
}

func newXMLDoc() *xmlDoc {
	xml := etree.NewDocument()
	xml.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)
	return &xmlDoc{xml: xml}
}

type xmlDoc struct {
	xml *etree.Document
}

func (t *xmlDoc) Type() string {
	return "XMLDocument"
}

func (t *xmlDoc) Size() int {
	return 1
}

func (t *xmlDoc) GetMethod(name string) core.NativeMethod {
	switch name {
	case "createElement":
		return t.createElement
	case "toString":
		return t.toString
	case "selectElement":
		return t.selectElement
	}
	return nil
}

func (t *xmlDoc) selectElement(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	e := t.xml.SelectElement(args[0].ToString())
	if e == nil {
		return core.NullValue, nil
	}
	return core.NewObject(&xmlElement{e}), nil
}

func (t *xmlDoc) toString(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgRange(args, 0, 0); err != nil {
		return core.NullValue, err
	}

	t.xml.Indent(2)

	s, err := t.xml.WriteToString()
	if err != nil {
		return core.NullValue, err
	}

	return core.NewString(s), nil
}

func (t *xmlDoc) createElement(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	e := t.xml.CreateElement(args[0].ToString())

	return core.NewObject(&xmlElement{e}), nil
}

type xmlElement struct {
	element *etree.Element
}

func (t *xmlElement) Type() string {
	return "XMLElement"
}

func (t *xmlElement) Size() int {
	return 1
}

func (t *xmlElement) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "tag":
		return core.NewString(t.element.Tag), nil
	}
	return core.UndefinedValue, nil
}

func (t *xmlElement) GetMethod(name string) core.NativeMethod {
	switch name {
	case "createAttribute":
		return t.createAttribute

	case "createElement":
		return t.createElement

	case "setValue":
		return t.setValue

	case "getAttribute":
		return t.getAttribute

	case "getValue":
		return t.getValue

	case "selectElement":
		return t.selectElement

	case "selectElements":
		return t.selectElements
	}
	return nil
}

func (t *xmlElement) selectElement(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	e := t.element.SelectElement(args[0].ToString())
	if e == nil {
		return core.NullValue, nil
	}
	return core.NewObject(&xmlElement{e}), nil
}

func (t *xmlElement) selectElements(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	elements := t.element.SelectElements(args[0].ToString())

	items := make([]core.Value, len(elements))

	for i, v := range elements {
		items[i] = core.NewObject(&xmlElement{v})
	}

	return core.NewArrayValues(items), nil
}

func (t *xmlElement) getValue(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	v := t.element.Text()
	return core.NewString(v), nil
}

func (t *xmlElement) getAttribute(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	a := args[0]
	v := t.element.SelectAttrValue(a.ToString(), "")
	return core.NewString(v), nil
}

func (t *xmlElement) setValue(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgRange(args, 1, 1); err != nil {
		return core.NullValue, err
	}

	a := args[0]

	switch a.Type {
	case core.Int, core.Float, core.String, core.Bool:
	default:
		return core.NullValue, ErrInvalidType
	}

	t.element.SetText(a.ToString())
	return core.NewObject(t), nil
}

func (t *xmlElement) createElement(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	e := t.element.CreateElement(args[0].ToString())

	return core.NewObject(&xmlElement{e}), nil
}

func (t *xmlElement) createAttribute(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String, core.String); err != nil {
		return core.NullValue, err
	}

	t.element.CreateAttr(args[0].ToString(), args[1].ToString())
	return core.NewObject(t), nil
}
