package utils

import "strings"

func NormalizeMac(mac string) string {
    return strings.ReplaceAll(strings.ToLower(mac), ":", "")
}
