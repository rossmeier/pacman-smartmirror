package packet

// Packet represents a unix package.
// Requirements for implementations:
//  - Filename() has to be unique inside a set or repo
//  - Name() and Version() together have to be unique inside a set or repo
type Packet interface {
	Name() string
	Version() string
	Filename() string
}
