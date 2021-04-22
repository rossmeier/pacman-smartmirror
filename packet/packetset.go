package packet

// Set is a set of packets indexed by filename
type Set map[string]Packet

// Insert inserts a new Packet into the set overwriting existing ones
// with the same filename
func (s Set) Insert(p Packet) {
	s[p.Filename()] = p
}

// ByFilename returns the searched packet by filename or nil if none
// matches the given filename
func (s Set) ByFilename(filename string) Packet {
	p, ok := s[filename]
	if !ok {
		return nil
	}

	return p
}

// ByName searches the set for all packet versions with the given
// packet name and returns a list of them
func (s Set) ByName(name string) []Packet {
	ps := make([]Packet, 0)
	for _, p := range s {
		if p.Name() == name {
			ps = append(ps, p)
		}
	}

	return ps
}

// FindOtherVersions searches the set for all packets that match the
// given one but are newer or older
func (s Set) FindOtherVersions(p Packet) []Packet {
	ps := make([]Packet, 0)
	for _, myP := range s {
		if myP.Name() == p.Name() {
			ps = append(ps, myP)
		}
	}

	return ps
}

// Delete removes a packet from the set by filename
func (s Set) Delete(filename string) {
	delete(s, filename)
}
