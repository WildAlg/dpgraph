package kcorecdp

import "strconv"

func formatF(v float64) string { return strconv.FormatFloat(v, 'f', 4, 64) }
