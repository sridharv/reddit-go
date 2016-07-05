package reddit_go

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/serenize/snaker"
	"github.com/sridharv/fail"
)

// This is not really a test but more a code generation tool and abuses the test infrastructure.

var convert = flag.String("convert", "", "File containing API to generate")

// To use this, copy and paste the contents of a table from https://github.com/reddit/reddit/wiki/JSON
// into a file. Then run go test -v -run TestConvert --convert=<path-to-file>
// This will generate the internal JSON structure for that type.
// You can then copy+paste the structure into a go file after verifying that things look fine to you.
func TestConvert(t *testing.T) {
	if *convert == "" {
		return
	}
	conversions := map[string]string{
		"object":      "json.RawMessage",
		"list<thing>": "[]Thing",
		"boolean":     "bool",
		"long":        "int64",
	}

	defer fail.Using(t.Fatal)
	data, err := ioutil.ReadFile(*convert)
	fail.IfErr(err)

	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		l := bytes.TrimSpace(line)
		if len(l) == 0 {
			continue
		}
		tokens := bytes.Split(l, []byte("\t"))
		name := snaker.SnakeToCamel(string(bytes.Title(tokens[1])))
		if name == "Id" {
			name = "ID"
		}
		t := string(bytes.ToLower(tokens[0]))
		converted, ok := conversions[t]
		if !ok {
			converted = t
		}
		fmt.Printf("\t%s %s `json:\"%s\"`\n", name, converted, string(tokens[1]))
	}
}
