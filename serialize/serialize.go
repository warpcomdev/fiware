package serialize

import (
	"encoding/json"
)

type Serializer interface {
	KeyString(k, v string)                            // , "key": "val"
	String(v string, compact bool)                    // , "val"
	KeyInt(k string, v int)                           // , "key": val
	KeyFloat(k string, v float64)                     // , "key": val
	KeyBool(k string, v bool)                         // , "key": val
	KeyRaw(k string, v json.RawMessage, compact bool) // , "key": val
	BeginBlock(string)                                // { + omit first ","
	EndBlock()                                        // }
	BeginList(string)                                 // [ + omit first ","
	EndList()                                         // ]
	Error() error                                     // If it failed at any step
}

// Interfaz que permite serializar un objeto
type Serializable interface {
	Serialize(Serializer)
}
