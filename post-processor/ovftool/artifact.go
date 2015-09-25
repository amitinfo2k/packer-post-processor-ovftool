package ovftool

import (
	"fmt"
	"os"
)

// Artifact is the result of running the VMware builder, namely a set
// of files associated with the resulting machine.
type Artifact struct {
	ova_file string
	id string
}

func (a *Artifact) BuilderId() string {
	return "x0A.ovftool"
}

func (a *Artifact) Id() string {
	return a.id
}

func (a *Artifact) Files() []string {
	return []string { a.ova_file }
}

func (a *Artifact) String() string {
	return fmt.Sprintf("OVA file : %s", a.ova_file)
}

func (a *Artifact) State(name string) interface{} {
	return nil
}

func (a *Artifact) Destroy() error {
	return os.Remove( a.ova_file)
}
