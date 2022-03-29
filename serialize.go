package fiware

import "encoding/json"

type Serializable interface {
	Serialize(Serializer)
}

type Serializer interface {
	KeyString(k, v string)                            // , "key": "val"
	String(v string)                                  // , "val"
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
