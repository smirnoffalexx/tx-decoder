package main

import "strings"

func removeHexPrefix(str string) string {
	return strings.Replace(str, "0x", "", -1)
}

func Contains[T comparable](s []T, e T) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}
	return false
}
