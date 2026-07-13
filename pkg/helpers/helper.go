package helpers


// ----------------------------
// Math Helpers
// ----------------------------

func MinInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func MaxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// ----------------------------
// String Helpers
// ----------------------------

// func IsEmpty(value string) bool {
// 	return value == ""
// }
