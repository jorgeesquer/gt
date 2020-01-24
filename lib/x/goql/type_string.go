// Code generated by "stringer -type=Type"; DO NOT EDIT.

package goql

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[NOTSET-0]
	_ = x[ERROR-1]
	_ = x[EOF-2]
	_ = x[COMMENT-3]
	_ = x[CREATE-4]
	_ = x[SHOW-5]
	_ = x[DROP-6]
	_ = x[ALTER-7]
	_ = x[TABLE-8]
	_ = x[DATABASE-9]
	_ = x[NOT-10]
	_ = x[EXISTS-11]
	_ = x[CONSTRAINT-12]
	_ = x[INTEGER-13]
	_ = x[DECIMAL-14]
	_ = x[CHAR-15]
	_ = x[VARCHAR-16]
	_ = x[TEXT-17]
	_ = x[MEDIUMTEXT-18]
	_ = x[BOOL-19]
	_ = x[BLOB-20]
	_ = x[DATETIME-21]
	_ = x[DEFAULT-22]
	_ = x[SELECT-23]
	_ = x[DISTINCT-24]
	_ = x[INSERT-25]
	_ = x[INTO-26]
	_ = x[VALUES-27]
	_ = x[UPDATE-28]
	_ = x[SET-29]
	_ = x[DELETE-30]
	_ = x[FROM-31]
	_ = x[WHERE-32]
	_ = x[GROUP-33]
	_ = x[HAVING-34]
	_ = x[JOIN-35]
	_ = x[LEFT-36]
	_ = x[RIGHT-37]
	_ = x[INNER-38]
	_ = x[OUTER-39]
	_ = x[CROSS-40]
	_ = x[ON-41]
	_ = x[AS-42]
	_ = x[IN-43]
	_ = x[NOTIN-44]
	_ = x[BETWEEN-45]
	_ = x[LIKE-46]
	_ = x[IS-47]
	_ = x[ISNOT-48]
	_ = x[NOTLIKE-49]
	_ = x[ORDER-50]
	_ = x[BY-51]
	_ = x[ASC-52]
	_ = x[DESC-53]
	_ = x[RANDOM-54]
	_ = x[LIMIT-55]
	_ = x[UNION-56]
	_ = x[AND-57]
	_ = x[OR-58]
	_ = x[NULL-59]
	_ = x[TRUE-60]
	_ = x[FALSE-61]
	_ = x[FOR-62]
	_ = x[IDENT-63]
	_ = x[INT-64]
	_ = x[FLOAT-65]
	_ = x[STRING-66]
	_ = x[ADD-67]
	_ = x[SUB-68]
	_ = x[MUL-69]
	_ = x[DIV-70]
	_ = x[MOD-71]
	_ = x[LSF-72]
	_ = x[RSF-73]
	_ = x[ANB-74]
	_ = x[EQL-75]
	_ = x[LSS-76]
	_ = x[GTR-77]
	_ = x[NT-78]
	_ = x[NEQ-79]
	_ = x[LEQ-80]
	_ = x[GEQ-81]
	_ = x[LPAREN-82]
	_ = x[LBRACK-83]
	_ = x[LBRACE-84]
	_ = x[COMMA-85]
	_ = x[PERIOD-86]
	_ = x[RPAREN-87]
	_ = x[COLON-88]
	_ = x[SEMICOLON-89]
	_ = x[QUESTION-90]
}

const _Type_name = "NOTSETERROREOFCOMMENTCREATESHOWDROPALTERTABLEDATABASENOTEXISTSCONSTRAINTINTEGERDECIMALCHARVARCHARTEXTMEDIUMTEXTBOOLBLOBDATETIMEDEFAULTSELECTDISTINCTINSERTINTOVALUESUPDATESETDELETEFROMWHEREGROUPHAVINGJOINLEFTRIGHTINNEROUTERCROSSONASINNOTINBETWEENLIKEISISNOTNOTLIKEORDERBYASCDESCRANDOMLIMITUNIONANDORNULLTRUEFALSEFORIDENTINTFLOATSTRINGADDSUBMULDIVMODLSFRSFANBEQLLSSGTRNTNEQLEQGEQLPARENLBRACKLBRACECOMMAPERIODRPARENCOLONSEMICOLONQUESTION"

var _Type_index = [...]uint16{0, 6, 11, 14, 21, 27, 31, 35, 40, 45, 53, 56, 62, 72, 79, 86, 90, 97, 101, 111, 115, 119, 127, 134, 140, 148, 154, 158, 164, 170, 173, 179, 183, 188, 193, 199, 203, 207, 212, 217, 222, 227, 229, 231, 233, 238, 245, 249, 251, 256, 263, 268, 270, 273, 277, 283, 288, 293, 296, 298, 302, 306, 311, 314, 319, 322, 327, 333, 336, 339, 342, 345, 348, 351, 354, 357, 360, 363, 366, 368, 371, 374, 377, 383, 389, 395, 400, 406, 412, 417, 426, 434}

func (i Type) String() string {
	if i >= Type(len(_Type_index)-1) {
		return "Type(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Type_name[_Type_index[i]:_Type_index[i+1]]
}
