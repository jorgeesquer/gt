// Code generated by "stringer -type=ColType"; DO NOT EDIT.

package dbx

import "fmt"

const _ColType_name = "StringIntDecimalBoolTimeDateDateTimeBlobUnknown"

var _ColType_index = [...]uint8{0, 6, 9, 16, 20, 24, 28, 36, 40, 47}

func (i ColType) String() string {
	if i < 0 || i >= ColType(len(_ColType_index)-1) {
		return fmt.Sprintf("ColType(%d)", i)
	}
	return _ColType_name[_ColType_index[i]:_ColType_index[i+1]]
}
