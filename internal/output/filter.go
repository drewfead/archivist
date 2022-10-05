package output

import (
	"io"
)

type filteredOutput struct {
	output io.WriteCloser
	accept func(bytes []byte) bool
}

func (fo *filteredOutput) Write(b []byte) (int, error) {
	if fo.accept(b) {
		return fo.output.Write(b)
	}
	return 0, nil
}

func (fo *filteredOutput) Close() error {
	return fo.output.Close()
}

func Filtered(output io.WriteCloser, accept func(bytes []byte) bool) io.WriteCloser {
	return &filteredOutput{
		output: output,
		accept: accept,
	}
}
