package importer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/warpcomdev/fiware/internal/serialize"
)

// loadCue reads a Cue file with the provided params as arguments
func loadCue(datafile string, params map[string]string, pathLib string) (string, error) {
	// Read cue file
	handle, err := os.Open(datafile)
	if err != nil {
		return "", err
	}
	defer handle.Close()
	databytes, err := ioutil.ReadAll(handle)
	if err != nil {
		return "", err
	}
	// Set params scope
	ctx := cuecontext.New()
	paramsJson, err := json.Marshal(params)
	if err != nil {
		return "", err
	}
	scope := ctx.CompileString(fmt.Sprintf("{\"params\": %s}", string(paramsJson)))
	// Compile cue
	value := ctx.CompileBytes(
		databytes,
		cue.Filename(datafile),
		cue.ImportPath(pathLib),
		cue.Scope(scope),
	)
	// Resolve cue
	resolved := value.Eval()
	if err := resolved.Err(); err != nil {
		return "", err
	}
	// Return resolved json!
	text, err := resolved.MarshalJSON()
	if err != nil {
		return "", err
	}
	return string(text), nil
}

type CueSerializer struct {
	serialize.BufferedSerializer
}

func (j *CueSerializer) End() {
	// Prepend matched variables
	if len(j.Matched) > 0 {
		for k, v := range j.Matched {
			if _, err := fmt.Fprintf(j.Original, "let %s = params.%s // %q;\n", k, k, v); err != nil {
				j.Err = err
				return
			}
		}
		if _, err := j.Original.WriteString("\n"); err != nil {
			j.Err = err
			return
		}
	}
	j.BufferedSerializer.End()
}
