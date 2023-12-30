package serialize

import "encoding/json"

// Some bools in the IoTA APIs have different behaviour if they are
// undefined versus false. For instance, "timestamp === false" might
// not be the same as "timestamp === undefined", there is a global config
// parameter for that.
//
// For this reason, all those bools that are marked as `omitempty`
// cannot be just omitted when set to 'false'. They must only be omitted
// if they are actually not defined.
//
// I could achieve this using `*bool` instead of `bool` as the type for
// these settings, but then I'd have to support pointers in the
// code generation tool, and see how to handle them in cue too.
//
// So in the end I preferred a custom type with custom marshaller,
// unmarshaller and a "SerializeHook" method that tells the code
// generation tool to generate the correct code.
type OptionalBool struct {
	Defined bool
	Value   bool
}

// UnmarshalJson implements the json.Unmarshaler interface.
func (o *OptionalBool) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &o.Value); err != nil {
		return err
	}
	o.Defined = true
	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (o *OptionalBool) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.Value)
}

// IsEmpty implements the code generator isEmpty interface
func (o OptionalBool) IsEmpty() bool {
	return !o.Defined
}

// SerializeHook Implements the code generator isHook interface
func (o OptionalBool) SerializeHook(property string, serializer Serializer) {
	serializer.KeyBool(property, o.Value)
}
