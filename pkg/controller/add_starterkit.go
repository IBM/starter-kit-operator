package controller

import (
	"github.com/ibm/starter-kit-operator/pkg/controller/starterkit"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, starterkit.Add)
}
