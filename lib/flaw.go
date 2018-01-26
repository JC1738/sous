package sous

import (
	"fmt"

	"github.com/pkg/errors"
)

type (
	// A Flaw captures the digression from a validation rule
	Flaw interface {
		AddContext(string, interface{})
		Repair() error
	}

	// Flawed covers types that can be validated and have flaws
	// Be kind to them, because aren't we all flawed somehow?
	Flawed interface {
		// Validate returns a list of flaws that enumerate problems with the Flawed
		Validate() []Flaw
	}

	// GenericFlaw is a generic Flaw.
	GenericFlaw struct {
		Desc       string
		RepairFunc func() error
	}
)

// RepairAll attempts to repair all the flaws in a slice, and returns errors
// and flaws when any of the flaws return errors
func RepairAll(in []Flaw) ([]Flaw, []error) {
	var fs []Flaw
	var es []error

	for _, f := range in {
		if e := f.Repair(); e != nil {
			es = append(es, e)
			fs = append(fs, f)
		}
	}
	return fs, es
}

// NewFlaw returns a new generic flaw with the given description and repair function
func NewFlaw(desc string, repair func() error) GenericFlaw {
	return GenericFlaw{
		Desc:       desc,
		RepairFunc: repair,
	}
}

// FatalFlaw constructs a Flaw that cannot be fixed.
func FatalFlaw(frmt string, vals ...interface{}) GenericFlaw {
	desc := fmt.Sprintf(frmt, vals...)
	return GenericFlaw{
		Desc: fmt.Sprintf(frmt, vals...),
		RepairFunc: func() error {
			return errors.Errorf("%s: cannot be repaired.", desc)
		},
	}
}

// Repair implements Flaw.Repair.
func (gf GenericFlaw) Repair() error {
	return gf.RepairFunc()
}

func (gf GenericFlaw) String() string {
	return gf.Desc
}

// AddContext discards the context - if you need the context, you should build
// a specialized Flaw
func (gf GenericFlaw) AddContext(name string, thing interface{}) {
}

func (gf GenericFlaw) Error() error {
	return errors.Errorf(gf.String())
}
