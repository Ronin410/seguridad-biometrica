package handlers

import "github.com/lib/pq"

// pq's GenericArray only implements Scan for element types with an explicit
// sql.Scanner, so a smallint[] column can't be scanned directly into a plain
// []int — it must go through pq.Int64Array first. toIntSlice converts the
// result back to []int for the rest of the code.
func toIntSlice(values pq.Int64Array) []int {
	result := make([]int, len(values))
	for i, v := range values {
		result[i] = int(v)
	}
	return result
}
