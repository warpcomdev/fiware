package serialize

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
)

type Writer interface {
	io.Writer
	io.StringWriter
}

const MinIndent = "  "

// Serializes a JSON object to a string.
type JsonSerializer struct {
	Writer        Writer            // Where to write
	SupportParams bool              // True to support parameter matching
	ReverseParams map[string]string // match strings and turn into parameters
	Matched       map[string]string // Which params were matched
	Depth         int               // indentation depth, if -1 then do not indent
	Err           error
	sep           string
}

// Setup must be called before starting serializing
func (j *JsonSerializer) Setup(w Writer, params map[string]string) {
	j.Writer = w
	j.ReverseParams = make(map[string]string)
	j.Matched = make(map[string]string)
	for k, v := range params {
		j.ReverseParams[v] = k
	}
}

// Begin serializing a new object
func (j *JsonSerializer) Begin() {
	j.sep = "\n"
	indent := j.indent()
	if _, err := fmt.Fprintf(j.Writer, "%s{", indent); err != nil {
		j.Err = err
		return
	}
	j.Depth += 1
}

// End object serialization
func (j *JsonSerializer) End() {
	if j.Err != nil {
		return
	}
	j.Depth -= 1
	indent := j.indent()
	fmt.Fprintf(j.Writer, "\n%s}\n", indent)
}

func (j *JsonSerializer) indent() string {
	if j.Err != nil || j.Depth < 0 {
		return ""
	}
	var result string
	for i := 0; i < j.Depth; i++ {
		result = result + MinIndent
	}
	return result
}

// Param called on every string, to check if it must be replaced
func (j *JsonSerializer) Param(s string) {
	if j.Err != nil {
		return
	}
	if j.SupportParams && j.ReverseParams != nil {
		if r, ok := j.ReverseParams[s]; ok {
			if _, err := j.Writer.WriteString(r); err == nil {
				j.Matched[r] = s
			} else {
				j.Err = err
			}
			return
		}
	}
	if _, err := fmt.Fprintf(j.Writer, "%q", s); err != nil {
		j.Err = err
	}
}

// KeyString dumps a key and value pair, value is string
func (j *JsonSerializer) KeyString(k, v string) {
	if j.Err != nil {
		return
	}
	if _, err := fmt.Fprintf(j.Writer, "%s%s%q: ", j.sep, j.indent(), k); err != nil {
		j.Err = err
		return
	}
	j.Param(v)
	j.sep = ",\n"
}

// KeyString dumps a string
func (j *JsonSerializer) String(v string, compact bool) {
	if j.Err != nil {
		return
	}
	var (
		indent string
		sep    string
	)
	if compact && strings.HasPrefix(j.sep, ",") {
		sep = ","
		indent = " "
	} else {
		sep = j.sep
		indent = j.indent()
	}
	if _, err := fmt.Fprintf(j.Writer, "%s%s", sep, indent); err != nil {
		j.Err = err
		return
	}
	j.Param(v)
	j.sep = ",\n"
}

// KeyInt dumps a key and value pair, value is int
func (j *JsonSerializer) KeyInt(k string, v int) {
	if j.Err != nil {
		return
	}
	if _, err := fmt.Fprintf(j.Writer, "%s%s%q: %d", j.sep, j.indent(), k, v); err != nil {
		j.Err = err
		return
	}
	j.sep = ",\n"
}

// KeyFloat dumps a key and value pair, value is float
func (j *JsonSerializer) KeyFloat(k string, v float64) {
	if j.Err != nil {
		return
	}
	if _, err := fmt.Fprintf(j.Writer, "%s%s%q: %f", j.sep, j.indent(), k, v); err != nil {
		j.Err = err
		return
	}
	j.sep = ",\n"
}

// KeyBool dumps a key and value pair, value is bool
func (j *JsonSerializer) KeyBool(k string, v bool) {
	if j.Err != nil {
		return
	}
	if _, err := fmt.Fprintf(j.Writer, "%s%s%q: %v", j.sep, j.indent(), k, v); err != nil {
		j.Err = err
		return
	}
	j.sep = ",\n"
}

// KeyRaw dumps a key and value pair, value is json.RawMessage
func (j *JsonSerializer) KeyRaw(k string, v json.RawMessage, compact bool) {
	if j.Err != nil {
		return
	}
	indent := j.indent()
	if _, err := fmt.Fprintf(j.Writer, "%s%s%q: ", j.sep, indent, k); err != nil {
		j.Err = err
		return
	}
	if !compact {
		var buf bytes.Buffer
		if err := json.Indent(&buf, v, indent, "  "); err != nil {
			log.Printf("failed to indent %s: %v", k, err)
		} else {
			v = json.RawMessage(buf.Bytes())
		}
	}
	if _, err := j.Writer.Write(v); err != nil {
		j.Err = err
		return
	}
	j.sep = ",\n"
}

// Serialize recursos into a Serializable object
func (j *JsonSerializer) Serialize(s Serializable) {
	s.Serialize(j)
}

// BeginBlock opens a block with an optional key
func (j *JsonSerializer) BeginBlock(optionalTitle string) {
	if j.Err != nil {
		return
	}
	indent := j.indent()
	if optionalTitle != "" {
		if _, err := fmt.Fprintf(j.Writer, "%s%s%q: {", j.sep, indent, optionalTitle); err != nil {
			j.Err = err
			return
		}
	} else {
		if _, err := fmt.Fprintf(j.Writer, "%s%s{", j.sep, indent); err != nil {
			j.Err = err
			return
		}
	}
	j.Depth += 1
	j.sep = "\n"
}

// Endblock closes a block
func (j *JsonSerializer) EndBlock() {
	if j.Err != nil {
		return
	}
	j.Depth -= 1
	indent := j.indent()
	if _, err := fmt.Fprintf(j.Writer, "\n%s}", indent); err != nil {
		j.Err = err
		return
	}
	j.sep = ",\n"
}

// BeginList opens a list with an optional key
func (j *JsonSerializer) BeginList(optionalTitle string) {
	if j.Err != nil {
		return
	}
	indent := j.indent()
	if optionalTitle != "" {
		if _, err := fmt.Fprintf(j.Writer, "%s%s%q: [", j.sep, indent, optionalTitle); err != nil {
			j.Err = err
			return
		}
	} else {
		if _, err := fmt.Fprintf(j.Writer, "%s: ", j.sep); err != nil {
			j.Err = err
			return
		}
	}
	j.Depth += 1
	j.sep = "\n"
}

// EndList closes a list
func (j *JsonSerializer) EndList() {
	if j.Err != nil {
		return
	}
	j.Depth -= 1
	if _, err := fmt.Fprintf(j.Writer, "\n%s]", j.indent()); err != nil {
		j.Err = err
		return
	}
	j.sep = ",\n"
}

// Error accumulates errors while encoding to check at the end
func (j *JsonSerializer) Error() error {
	return j.Err
}
