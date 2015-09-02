package securitycontextconstraints

import (
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/runtime"

	"github.com/openshift/origin/pkg/client"
	"github.com/openshift/origin/pkg/security/policy/api"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/watch"
)

// REST provides a proxy storage to PodSecurityPolicy for dealing with requests that come
// in as an SCC.
type REST struct {
	client client.PodSecurityPolicyInterface
}

// NewREST creates a new REST.
func NewREST(client client.PodSecurityPolicyInterface) *REST {
	return &REST{
		client: client,
	}
}

// policyToConstraint converts a PodSecurityPolicy to a SecurityContextConstraints.
func policyToConstraint(psp *api.PodSecurityPolicy) *api.SecurityContextConstraints {
	return &api.SecurityContextConstraints{
		*psp,
	}
}

// New returns a new SecurityContextConstraints.
func (s *REST) New() runtime.Object {
	return &api.SecurityContextConstraints{}
}

// NewList returns a new SecurityContextConstraintsList.
func (s *REST) NewList() runtime.Object {
	return &api.SecurityContextConstraintsList{}
}

// Get gets PodSecurityPolicies in the form of SecurityContextConstraints.
func (s *REST) Get(ctx kapi.Context, name string) (runtime.Object, error) {
	psp, err := s.client.Get(name)
	if err != nil {
		return nil, err
	}
	return policyToConstraint(psp), nil
}

// List lists PodSecurityPolicies in the form of SecurityContextConstraints.
func (s *REST) List(ctx kapi.Context, label labels.Selector, field fields.Selector) (runtime.Object, error) {
	pspList, err := s.client.List(label, field)
	if err != nil {
		return nil, err
	}
	sccs := []api.SecurityContextConstraints{}
	for _, psp := range pspList.Items {
		sccs = append(sccs, *policyToConstraint(&psp))
	}
	sccList := &api.SecurityContextConstraintsList{
		Items: sccs,
	}
	return sccList, nil
}

// Create creates PodSecurityPolicies in the form of SecurityContextConstraints.
func (s *REST) Create(ctx kapi.Context, obj runtime.Object) (runtime.Object, error) {
	scc := obj.(*api.SecurityContextConstraints)
	psp, err := s.client.Create(&scc.PodSecurityPolicy)
	if err != nil {
		return nil, err
	}
	return policyToConstraint(psp), nil
}

// Update updates PodSecurityPolicies in the form of SecurityContextConstraints.
func (s *REST) Update(ctx kapi.Context, obj runtime.Object) (runtime.Object, bool, error) {
	scc := obj.(*api.SecurityContextConstraints)
	psp, err := s.client.Create(&scc.PodSecurityPolicy)
	if err != nil {
		return nil, false, err
	}
	return policyToConstraint(psp), true, nil
}

// Delete deletes PodSecurityPolicies in the form of SecurityContextConstraints.
func (s *REST) Delete(ctx kapi.Context, name string, options *kapi.DeleteOptions) (runtime.Object, error) {
	err := s.client.Delete(name)
	return nil, err
}

// Watch watches PodSecurityPolicies.  It DOES NOT convert them to SecurityContextConstraints.
func (s *REST) Watch(ctx kapi.Context, label labels.Selector, field fields.Selector, resourceVersion string) (watch.Interface, error) {
	return s.client.Watch(label, field, resourceVersion)
}
