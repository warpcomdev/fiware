package serialize

import (
	"encoding/json"
	"sort"
)

type Serializable interface {
	Serialize(Serializer)
}

type Serializer interface {
	KeyString(k, v string)                            // , "key": "val"
	String(v string, compact bool)                    // , "val"
	KeyInt(k string, v int)                           // , "key": val
	KeyFloat(k string, v float64)                     // , "key": val
	KeyBool(k string, v bool)                         // , "key": val
	KeyRaw(k string, v json.RawMessage, compact bool) // , "key": val
	Serialize(Serializable)
	BeginBlock(string) // { + omit first ","
	EndBlock()         // }
	BeginList(string)  // [ + omit first ","
	EndList()          // ]
	Error() error      // If it failed at any step
}

// Keys returns the sorted list of keys in a map
func Keys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return Sorted(keys)
}

// Sorted sorts a string slice
func Sorted(keys []string) []string {
	sort.Sort(sort.StringSlice(keys))
	return keys
}
