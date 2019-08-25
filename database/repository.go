package database

// Repository describes a Repo as found on an upstream server
type Repository struct {
	Name string
	Arch string
}

func (r Repository) String() string {
	return r.Arch + "/" + r.Name
}
