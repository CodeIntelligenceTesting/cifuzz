package stringutil

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

func ToJsonString(v interface{}) (string, error) {
	var bytes []byte
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", errors.WithStack(err)
	}
	return string(bytes), nil
}

func PrettyString(v interface{}) string {
	jsonString, err := ToJsonString(v)
	if err != nil {
		return fmt.Sprintf("%+v", v)
	}
	return jsonString
}

// JoinNonEmpty does the same as strings.Join but omits empty elements
func JoinNonEmpty(elems []string, sep string) string {
	return strings.Join(NonEmpty(elems), sep)
}

// NonEmpty returns a slice with all empty strings removed
func NonEmpty(elems []string) []string {
	var res []string
	for _, e := range elems {
		if e != "" {
			res = append(res, e)
		}
	}
	return res
}

func JoinSlices(sep string, slices ...[]string) []string {
	switch len(slices) {
	case 0:
		return nil
	case 1:
		return slices[0]
	}

	res := slices[0]
	for _, s := range slices[1:] {
		res = append(append(res, sep), s...)
	}
	return res
}

func QuotedStrings(elems []string) []string {
	var quotedElems []string
	for _, arg := range elems {
		quotedElems = append(quotedElems, fmt.Sprintf("%q", arg))
	}
	return quotedElems
}

func Contains(slice []string, element string) bool {
	for _, e := range slice {
		if e == element {
			return true
		}
	}
	return false
}

func ContainsStringWithPrefix(slice []string, prefix string) bool {
	for _, e := range slice {
		if strings.HasPrefix(e, prefix) {
			return true
		}
	}
	return false
}

func Equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func SubtractSlices(a, b []string) (diff []string) {
	// Based on https://stackoverflow.com/a/45428032
	// Original author: https://stackoverflow.com/users/604260/peterwilliams97
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}
