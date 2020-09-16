package controller

import (
	appsv1 "github.com/openshift/api/apps/v1"
	buildv1 "github.com/openshift/api/build/v1"
	configv1 "github.com/openshift/api/config/v1"
	consolev1 "github.com/openshift/api/console/v1"
	imagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager) error {
	if err := appsv1.AddToScheme(m.GetScheme()); err != nil {
		return err
	}

	if err := buildv1.AddToScheme(m.GetScheme()); err != nil {
		return err
	}

	if err := configv1.AddToScheme(m.GetScheme()); err != nil {
		return err
	}

	if err := imagev1.AddToScheme(m.GetScheme()); err != nil {
		return err
	}

	if err := routev1.AddToScheme(m.GetScheme()); err != nil {
		return err
	}

	if err := consolev1.AddToScheme(m.GetScheme()); err != nil {
		return err
	}

	for _, f := range AddToManagerFuncs {
		if err := f(m); err != nil {
			return err
		}
	}
	return nil
}
