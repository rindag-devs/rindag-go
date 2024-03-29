package problem

import (
	"bytes"
	"io"

	"rindag/service/etc"
	"rindag/service/judge"

	"github.com/criyle/go-judge/pb"
	log "github.com/sirupsen/logrus"
)

// Generator is a generator to the problem.
type Generator struct {
	// binaryID is the ID of the generator binary.
	//
	// If the generator is not compiled, the binaryID will be nil.
	binaryID *string

	// GetSource is a function returns the source code ReadCloser of the checker.
	GetSource func() (io.ReadCloser, error)
}

// NewGenerator creates a generator.
func NewGenerator(getSource func() (io.ReadCloser, error)) *Generator {
	return &Generator{binaryID: new(string), GetSource: getSource}
}

// NewGeneratorFromProblem creates a generator from a problem.
func NewGeneratorFromProblem(problem *Problem, rev [20]byte, path string) *Generator {
	return NewGenerator(func() (io.ReadCloser, error) { return problem.File(rev, path) })
}

// NewGeneratorFromBytes creates a generator from the source code.
func NewGeneratorFromBytes(source []byte) *Generator {
	return NewGenerator(
		func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(source)), nil })
}

// NewGeneratorFromReadCloser creates a generator from the ReadCloser.
func NewGeneratorFromReadCloser(r io.ReadCloser) *Generator {
	return NewGenerator(func() (io.ReadCloser, error) { return r, nil })
}

// CompileTask returns the compile task of the generator.
func (g *Generator) CompileTask(cb judge.CallbackFunction) (*judge.Task, error) {
	conf := etc.Config
	source, err := g.GetSource()
	if err != nil {
		return nil, err
	}
	defer source.Close()
	bytes, err := io.ReadAll(source)
	if err != nil {
		return nil, err
	}
	return judge.DefaultTask().
		WithCmd(conf.Compile.Cmd...).
		WithCmd(conf.Generator.Compile.Args...).
		WithCmd("generator.cpp", "-o", "generator").
		WithTimeLimit(conf.Compile.TimeLimit).
		WithMemoryLimit(conf.Compile.MemoryLimit).
		WithStderrLimit(conf.Compile.StderrLimit).
		WithCopyIn("generator.cpp", bytes).
		WithCopyIn("testlib.h", TestlibSource).
		WithCopyOut("generator").
		WithCallback(func(r *pb.Response_Result, err error) bool {
			if finished := err == nil && r.Status == pb.Response_Result_Accepted; finished {
				ok := false
				if *g.binaryID, ok = r.FileIDs["generator"]; !ok {
					// Impossible to happen.
					log.Fatal("generator compile successful, but binary ID not found")
				}
			}
			return cb(r, err)
		}), nil
}

// GenerateTask returns a judge task to run this generator.
func (g *Generator) GenerateTask(args []string, cb judge.CallbackFunction) *judge.Task {
	conf := &etc.Config.Generator
	return judge.DefaultTask().
		WithCmd("generator").
		WithCmd(args...).
		WithTimeLimit(conf.Run.TimeLimit).
		WithMemoryLimit(conf.Run.MemoryLimit).
		WithStderrLimit(conf.Run.StderrLimit).
		WithCopyInCached("generator", g.binaryID).
		WithCallback(cb)
}
