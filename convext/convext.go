// Package convext (convert extensions) are helpers for converting things.
package convext

import (
	"encoding/json"
	"sort"
	"strconv"
)

func Must(e error) {
	if e != nil {
		panic(e)
	}
}

// ToObject converts o to json, and then parses it into an object.
// Useful when you need to get the fields for an object, like for log fields.
// This is slow, so use it sparingly- if you need it to be fast,
// create a method on your type.
func ToObject(o interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}

func MustToObject(o interface{}) map[string]interface{} {
	m, err := ToObject(o)
	Must(err)
	return m
}

func MustToJson(o interface{}) string {
	b, err := json.MarshalIndent(o, "", "  ")
	Must(err)
	return string(b)
}

func MustParseBool(s string) bool {
	b, err := strconv.ParseBool(s)
	Must(err)
	return b
}

func MustParseFloat(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	Must(err)
	return f
}

func MustParseInt(s string) int {
	i, err := strconv.Atoi(s)
	Must(err)
	return i
}

func SortedObjectKeys(o map[string]interface{}) []string {
	result := make([]string, 0, len(o))
	for k := range o {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}
