package fs

import (
	"fmt"
	"io/fs"
	"os"
)

// Tests are classified by the way they're defined:
// - k6-standalone is a test that can be run with one k6 binary
// - k6-operator is a test that requires a yaml file deployment

type TestType string

const (
	k6Standalone = TestType("k6-standalone")
	k6Operator   = TestType("k6-operator")
)

type TestDef struct {
	Location string
	Type     TestType
	folder   string // duplicate?

	Opts *K6Options
}

func (td TestDef) IsYaml() bool {
	return td.Type == k6Operator
}

func (td TestDef) ReadTest() ([]byte, error) {
	fsys := os.DirFS(td.folder)
	data, err := fs.ReadFile(fsys, td.Location)
	return data, err
}

func (td TestDef) Cmd() string {
	return fmt.Sprintf("k6 %s %s %s", td.Opts.Command, td.Location, td.Opts.Arguments)
}

type K6Options struct {
	Version   string
	Command   string
	Arguments string
	Envvars   string
}

var defaultK6Opts = &K6Options{
	Version: "0.45.1",
	Command: "run",
}
