package packet

import "fmt"

type FromFilenameFunc func(string) (Packet, error)

var fromFilenames = make(map[string]FromFilenameFunc)

func FromFilename(impl, x string) (Packet, error) {
	fn, ok := fromFilenames[impl]
	if !ok {
		return nil, fmt.Errorf("unknown packet implementation: %s", impl)
	}
	return fn(x)
}

func RegisterImpl(name string, fn FromFilenameFunc) {
	if _, ok := fromFilenames[name]; ok {
		panic("Given impl already registered")
	}
	fromFilenames[name] = fn
}
