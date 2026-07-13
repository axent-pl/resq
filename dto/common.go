package dto

import "strconv"

func UintPointerToStringPointer(val *uint) *string {
	if val == nil {
		return nil
	}
	value := strconv.FormatUint(uint64(*val), 10)
	return &value
}
