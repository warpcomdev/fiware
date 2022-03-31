package serialize

import (
	"bufio"
	"bytes"
)

// Serializer that collects output into a buffer
type BufferedSerializer struct {
	JsonSerializer
	// We buffer the writer to prepend locals later
	Original Writer
	Buffer   bytes.Buffer
	Buffered *bufio.Writer
}

// Setup implements Serializer
func (j *BufferedSerializer) Setup(w Writer, params map[string]string) {
	j.Original = w
	j.Buffered = bufio.NewWriter(&(j.Buffer))
	j.JsonSerializer.Setup(j.Buffered, params)
}

// End implements Serializer
func (j *BufferedSerializer) End() {
	j.JsonSerializer.End()
	if j.Err != nil {
		return
	}
	j.Buffered.Flush()
	if j.Original != nil {
		if _, err := j.Original.Write(j.Buffer.Bytes()); err != nil {
			j.Err = err
		}
	}
}

func MarshalJSON(s Serializable) ([]byte, error) {
	b := BufferedSerializer{}
	b.Depth = -1
	b.Setup(nil, nil)
	b.Begin()
	b.Serialize(s)
	b.End()
	return b.Buffer.Bytes(), b.Err
}
