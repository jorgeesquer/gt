// Code generated by "stringer -type=SectionType"; DO NOT EDIT.

package binary

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[section_directives-0]
	_ = x[section_build-1]
	_ = x[section_functions-2]
	_ = x[section_dynamicCalls-3]
	_ = x[section_registers-4]
	_ = x[section_instructions-5]
	_ = x[section_constants-6]
	_ = x[section_positions-7]
	_ = x[section_files-8]
	_ = x[section_resources-9]
	_ = x[section_sources-10]
	_ = x[section_sourceLines-11]
	_ = x[section_string-12]
	_ = x[section_bytes-13]
	_ = x[section_kInt-14]
	_ = x[section_kFloat-15]
	_ = x[section_kBool-16]
	_ = x[section_kString-17]
	_ = x[section_kNull-18]
	_ = x[section_kUndefined-19]
	_ = x[section_kRune-20]
	_ = x[section_EOF-21]
}

const _SectionType_name = "section_directivessection_buildsection_functionssection_dynamicCallssection_registerssection_instructionssection_constantssection_positionssection_filessection_resourcessection_sourcessection_sourceLinessection_stringsection_bytessection_kIntsection_kFloatsection_kBoolsection_kStringsection_kNullsection_kUndefinedsection_kRunesection_EOF"

var _SectionType_index = [...]uint16{0, 18, 31, 48, 68, 85, 105, 122, 139, 152, 169, 184, 203, 217, 230, 242, 256, 269, 284, 297, 315, 328, 339}

func (i SectionType) String() string {
	if i < 0 || i >= SectionType(len(_SectionType_index)-1) {
		return "SectionType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _SectionType_name[_SectionType_index[i]:_SectionType_index[i+1]]
}
