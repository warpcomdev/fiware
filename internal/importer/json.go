package importer

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/warpcomdev/fiware"
)

type Writer interface {
	io.Writer
	io.StringWriter
}

const minIndent = "  "

// Serializes a JSON object to a string.
type JsonSerializer struct {
	Writer        Writer
	ReverseParams map[string]string
	Matched       map[string]string
	Depth         int
	sep           string
	err           error
}

type bufferedSerializer struct {
	JsonSerializer
	// We buffer the writer to prepend locals later
	original Writer
	buffer   bytes.Buffer
	buffered *bufio.Writer
}

func (j *JsonSerializer) Setup(w Writer, params map[string]string) {
	j.Writer = w
	j.ReverseParams = make(map[string]string)
	j.Matched = make(map[string]string)
	for k, v := range params {
		j.ReverseParams[v] = k
	}
}

func (j *JsonSerializer) Begin() {
	j.sep = "\n"
	indent := j.indent()
	if _, err := fmt.Fprintf(j.Writer, "%s{", indent); err != nil {
		j.err = err
		return
	}
	j.Depth += 1
}

func (j *JsonSerializer) End() {
	if j.err != nil {
		return
	}
	j.Depth -= 1
	indent := j.indent()
	fmt.Fprintf(j.Writer, "\n%s}\n", indent)
}

func (j *JsonSerializer) indent() string {
	if j.err != nil {
		return ""
	}
	var result string
	for i := 0; i < j.Depth; i++ {
		result = result + minIndent
	}
	return result
}

func (j *JsonSerializer) Param(s string) {
	if j.err != nil {
		return
	}
	if j.ReverseParams != nil {
		if r, ok := j.ReverseParams[s]; ok {
			if _, err := j.Writer.WriteString(r); err == nil {
				j.Matched[r] = s
			} else {
				j.err = err
			}
			return
		}
	}
	if _, err := fmt.Fprintf(j.Writer, "%q", s); err != nil {
		j.err = err
	}
}

func (j *JsonSerializer) KeyString(k, v string) {
	if j.err != nil {
		return
	}
	if _, err := fmt.Fprintf(j.Writer, "%s%s%q: ", j.sep, j.indent(), k); err != nil {
		j.err = err
		return
	}
	j.Param(v)
	j.sep = ",\n"
}

func (j *JsonSerializer) String(v string) {
	if j.err != nil {
		return
	}
	if _, err := fmt.Fprintf(j.Writer, "%s%s", j.sep, j.indent()); err != nil {
		j.err = err
		return
	}
	j.Param(v)
	j.sep = ",\n"
}

func (j *JsonSerializer) KeyInt(k string, v int) {
	if j.err != nil {
		return
	}
	if _, err := fmt.Fprintf(j.Writer, "%s%s%q: %d", j.sep, j.indent(), k, v); err != nil {
		j.err = err
		return
	}
	j.sep = ",\n"
}

func (j *JsonSerializer) KeyFloat(k string, v float64) {
	if j.err != nil {
		return
	}
	if _, err := fmt.Fprintf(j.Writer, "%s%s%q: %f", j.sep, j.indent(), k, v); err != nil {
		j.err = err
		return
	}
	j.sep = ",\n"
}

func (j *JsonSerializer) KeyBool(k string, v bool) {
	if j.err != nil {
		return
	}
	if _, err := fmt.Fprintf(j.Writer, "%s%s%q: %v", j.sep, j.indent(), k, v); err != nil {
		j.err = err
		return
	}
	j.sep = ",\n"
}

func (j *JsonSerializer) KeyRaw(k string, v json.RawMessage, compact bool) {
	if j.err != nil {
		return
	}
	indent := j.indent()
	if _, err := fmt.Fprintf(j.Writer, "%s%s%q: ", j.sep, indent, k); err != nil {
		j.err = err
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
		j.err = err
		return
	}
	j.sep = ",\n"
}

func (j *JsonSerializer) Serialize(s fiware.Serializable) {
	s.Serialize(j)
}

func (j *JsonSerializer) BeginBlock(optionalTitle string) {
	if j.err != nil {
		return
	}
	indent := j.indent()
	if optionalTitle != "" {
		if _, err := fmt.Fprintf(j.Writer, "%s%s%q: {", j.sep, indent, optionalTitle); err != nil {
			j.err = err
			return
		}
	} else {
		if _, err := fmt.Fprintf(j.Writer, "%s%s{", j.sep, indent); err != nil {
			j.err = err
			return
		}
	}
	j.Depth += 1
	j.sep = "\n"
}

func (j *JsonSerializer) EndBlock() {
	if j.err != nil {
		return
	}
	j.Depth -= 1
	indent := j.indent()
	if _, err := fmt.Fprintf(j.Writer, "\n%s}", indent); err != nil {
		j.err = err
		return
	}
	j.sep = ",\n"
}

func (j *JsonSerializer) BeginList(optionalTitle string) {
	if j.err != nil {
		return
	}
	indent := j.indent()
	if optionalTitle != "" {
		if _, err := fmt.Fprintf(j.Writer, "%s%s%q: [", j.sep, indent, optionalTitle); err != nil {
			j.err = err
			return
		}
	} else {
		if _, err := fmt.Fprintf(j.Writer, "%s: ", j.sep); err != nil {
			j.err = err
			return
		}
	}
	j.Depth += 1
	j.sep = "\n"
}

func (j *JsonSerializer) EndList() {
	if j.err != nil {
		return
	}
	j.Depth -= 1
	if _, err := fmt.Fprintf(j.Writer, "\n%s]", j.indent()); err != nil {
		j.err = err
		return
	}
	j.sep = ",\n"
}

func (j *JsonSerializer) Error() error {
	return j.err
}

func (j *bufferedSerializer) Setup(w Writer, params map[string]string) {
	j.original = w
	j.buffered = bufio.NewWriter(&(j.buffer))
	j.JsonSerializer.Setup(j.buffered, params)
}

func (j *bufferedSerializer) End() {
	j.JsonSerializer.End()
	if j.err != nil {
		return
	}
	j.buffered.Flush()
	if _, err := j.original.Write(j.buffer.Bytes()); err != nil {
		j.err = err
	}
}
