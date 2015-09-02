package admission

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	kadmission "k8s.io/kubernetes/pkg/admission"
	kapi "k8s.io/kubernetes/pkg/api"
	kerrors "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/auth/user"
	kclient "k8s.io/kubernetes/pkg/client"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/controller/serviceaccount"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util"
	"k8s.io/kubernetes/pkg/util/fielderrors"
	"k8s.io/kubernetes/pkg/watch"

	osclient "github.com/openshift/origin/pkg/client"
	allocator "github.com/openshift/origin/pkg/security"
	policyapi "github.com/openshift/origin/pkg/security/policy/api"
	policyprovider "github.com/openshift/origin/pkg/security/policy/provider"
	"github.com/openshift/origin/pkg/security/uid"

	"github.com/golang/glog"
)

func init() {
	kadmission.RegisterPlugin("PodSecurityPolicy", func(c kclient.Interface, config io.Reader) (kadmission.Interface, error) {
		osClient, ok := c.(osclient.Interface)
		if !ok {
			return nil, errors.New("client is not an Origin client")
		}
		return NewConstraint(c, osClient), nil
	})
}

type constraint struct {
	*kadmission.Handler
	client kclient.Interface
	store  cache.Store
}

var _ kadmission.Interface = &constraint{}

// NewConstraint creates a new SCC constraint admission plugin.
func NewConstraint(kubeClient kclient.Interface, osClient osclient.Interface) kadmission.Interface {
	store := cache.NewStore(cache.MetaNamespaceKeyFunc)
	reflector := cache.NewReflector(
		&cache.ListWatch{
			ListFunc: func() (runtime.Object, error) {
				return osClient.PodSecurityPolicies().List(labels.Everything(), fields.Everything())
			},
			WatchFunc: func(resourceVersion string) (watch.Interface, error) {
				return osClient.PodSecurityPolicies().Watch(labels.Everything(), fields.Everything(), resourceVersion)
			},
		},
		&policyapi.PodSecurityPolicy{},
		store,
		0,
	)
	reflector.Run()

	return &constraint{
		Handler: kadmission.NewHandler(kadmission.Create),
		client:  kubeClient,
		store:   store,
	}
}

// Admit determines if the pod should be admitted based on the requested security context
// and the available SCCs.
//
// 1.  Find SCCs for the user.
// 2.  Find SCCs for the SA.  If there is an error retrieving SA SCCs it is not fatal.
// 3.  Remove duplicates between the user/SA SCCs.
// 4.  Create the providers, includes setting pre-allocated values if necessary.
// 5.  Try to generate and validate an SCC with providers.  If we find one then admit the pod
//     with the validated SCC.  If we don't find any reject the pod and give all errors from the
//     failed attempts.
func (c *constraint) Admit(a kadmission.Attributes) error {
	if a.GetResource() != string(kapi.ResourcePods) {
		return nil
	}

	pod, ok := a.GetObject().(*kapi.Pod)
	// if we can't convert then we don't handle this object so just return
	if !ok {
		return nil
	}

	// get all constraints that are usable by the user
	glog.V(4).Infof("getting pod security policies for pod %s (generate: %s) in namespace %s with user info %v", pod.Name, pod.GenerateName, a.GetNamespace(), a.GetUserInfo())
	matchedConstraints, err := getMatchingPodSecurityPolicies(c.store, a.GetUserInfo())
	if err != nil {
		return kadmission.NewForbidden(a, err)
	}

	// get all constraints that are usable by the SA
	if len(pod.Spec.ServiceAccountName) > 0 {
		userInfo := serviceaccount.UserInfo(a.GetNamespace(), pod.Spec.ServiceAccountName, "")
		glog.V(4).Infof("getting pod security policies for pod %s (generate: %s) with service account info %v", pod.Name, pod.GenerateName, userInfo)
		saConstraints, err := getMatchingPodSecurityPolicies(c.store, userInfo)
		if err != nil {
			return kadmission.NewForbidden(a, err)
		}
		matchedConstraints = append(matchedConstraints, saConstraints...)
	}

	// remove duplicate constraints and sort
	matchedConstraints = deduplicateSecurityContextConstraints(matchedConstraints)
	sort.Sort(ByRestrictions(matchedConstraints))
	providers, errs := c.createProvidersFromConstraints(a.GetNamespace(), matchedConstraints)
	logProviders(pod, providers, errs)

	if len(providers) == 0 {
		return kadmission.NewForbidden(a, fmt.Errorf("no providers available to validated pod request"))
	}

	// all containers in a single pod must validate under a single provider or we will reject the request
	validationErrs := fielderrors.ValidationErrorList{}
	for _, provider := range providers {
		if errs := assignSecurityContext(provider, pod); len(errs) > 0 {
			validationErrs = append(validationErrs, errs.Prefix(fmt.Sprintf("provider %s: ", provider.GetPolicyName()))...)
			continue
		}

		// the entire pod validated, annotate and accept the pod
		glog.V(4).Infof("pod %s (generate: %s) validated against provider %s", pod.Name, pod.GenerateName, provider.GetPolicyName())
		if pod.ObjectMeta.Annotations == nil {
			pod.ObjectMeta.Annotations = map[string]string{}
		}
		pod.ObjectMeta.Annotations[allocator.ValidatedSCCAnnotation] = provider.GetPolicyName()
		return nil
	}

	// we didn't validate against any security context constraint provider, reject the pod and give the errors for each attempt
	glog.V(4).Infof("unable to validate pod %s (generate: %s) against any security context constraint: %v", pod.Name, pod.GenerateName, validationErrs)
	return kadmission.NewForbidden(a, fmt.Errorf("unable to validate against any security context constraint: %v", validationErrs))
}

// assignSecurityContext creates a security context for each container in the pod
// and validates that the sc falls within the scc constraints.  All containers must validate against
// the same scc or is not considered valid.
func assignSecurityContext(provider policyprovider.PodSecurityPolicyProvider, pod *kapi.Pod) fielderrors.ValidationErrorList {
	generatedSCs := make([]*kapi.SecurityContext, len(pod.Spec.Containers))

	errs := fielderrors.ValidationErrorList{}

	for i, c := range pod.Spec.Containers {
		sc, err := provider.CreateSecurityContext(pod, &c)
		if err != nil {
			errs = append(errs, fielderrors.NewFieldInvalid(fmt.Sprintf("spec.containers[%d].securityContext", i), "", err.Error()))
			continue
		}
		generatedSCs[i] = sc

		c.SecurityContext = sc
		errs = append(errs, provider.ValidateSecurityContext(pod, &c).Prefix(fmt.Sprintf("spec.containers[%d].securityContext", i))...)
	}

	if len(errs) > 0 {
		return errs
	}

	// if we've reached this code then we've generated and validated an SC for every container in the
	// pod so let's apply what we generated
	for i, sc := range generatedSCs {
		pod.Spec.Containers[i].SecurityContext = sc
	}
	return nil
}

// createProvidersFromConstraints creates providers from the constraints supplied, including
// looking up pre-allocated values if necessary using the pod's namespace.
func (c *constraint) createProvidersFromConstraints(ns string, sccs []*policyapi.PodSecurityPolicy) ([]policyprovider.PodSecurityPolicyProvider, []error) {
	var (
		// namespace is declared here for reuse but we will not fetch it unless required by the matched constraints
		namespace *kapi.Namespace
		// collected providers
		providers []policyprovider.PodSecurityPolicyProvider
		// collected errors to return
		errs []error
	)

	// set pre-allocated values on constraints
	for _, constraint := range sccs {
		var err error
		resolveUIDRange := requiresPreAllocatedUIDRange(constraint)
		resolveSELinuxLevel := requiresPreAllocatedSELinuxLevel(constraint)

		if resolveUIDRange || resolveSELinuxLevel {
			var min, max *int64
			var level string

			// Ensure we have the namespace
			if namespace, err = c.getNamespace(ns, namespace); err != nil {
				errs = append(errs, fmt.Errorf("error fetching namespace %s required to preallocate values for %s: %v", ns, constraint.Name, err))
				continue
			}

			// Resolve the values from the namespace
			if resolveUIDRange {
				if min, max, err = getPreallocatedUIDRange(namespace); err != nil {
					errs = append(errs, fmt.Errorf("unable to find pre-allocated uid annotation for namespace %s while trying to configure SCC %s: %v", namespace.Name, constraint.Name, err))
					continue
				}
			}
			if resolveSELinuxLevel {
				if level, err = getPreallocatedLevel(namespace); err != nil {
					errs = append(errs, fmt.Errorf("unable to find pre-allocated mcs annotation for namespace %s while trying to configure SCC %s: %v", namespace.Name, constraint.Name, err))
					continue
				}
			}

			// Make a copy of the constraint so we don't mutate the store's cache
			var constraintCopy policyapi.PodSecurityPolicy = *constraint
			constraint = &constraintCopy
			if resolveSELinuxLevel && constraint.SELinuxContext.SELinuxOptions != nil {
				// Make a copy of the SELinuxOptions so we don't mutate the store's cache
				var seLinuxOptionsCopy kapi.SELinuxOptions = *constraint.SELinuxContext.SELinuxOptions
				constraint.SELinuxContext.SELinuxOptions = &seLinuxOptionsCopy
			}

			// Set the resolved values
			if resolveUIDRange {
				constraint.RunAsUser.UIDRangeMin = min
				constraint.RunAsUser.UIDRangeMax = max
			}
			if resolveSELinuxLevel {
				if constraint.SELinuxContext.SELinuxOptions == nil {
					constraint.SELinuxContext.SELinuxOptions = &kapi.SELinuxOptions{}
				}
				constraint.SELinuxContext.SELinuxOptions.Level = level
			}
		}

		// Create the provider
		provider, err := policyprovider.NewSimpleProvider(constraint)
		if err != nil {
			errs = append(errs, fmt.Errorf("error creating provider for SCC %s in namespace %s: %v", constraint.Name, ns, err))
			continue
		}
		providers = append(providers, provider)
	}
	return providers, errs
}

// getNamespace retrieves a namespace only if ns is nil.
func (c *constraint) getNamespace(name string, ns *kapi.Namespace) (*kapi.Namespace, error) {
	if ns != nil && name == ns.Name {
		return ns, nil
	}
	return c.client.Namespaces().Get(name)
}

// getMatchingPodSecurityPolicies returns constraints from the store that match the group,
// uid, or user of the service account.
func getMatchingPodSecurityPolicies(store cache.Store, userInfo user.Info) ([]*policyapi.PodSecurityPolicy, error) {
	matchedConstraints := make([]*policyapi.PodSecurityPolicy, 0)

	for _, c := range store.List() {
		constraint, ok := c.(*policyapi.PodSecurityPolicy)
		if !ok {
			return nil, kerrors.NewInternalError(fmt.Errorf("error converting object from store to a security context constraint: %v", c))
		}
		if PolicyAppliesTo(constraint, userInfo) {
			matchedConstraints = append(matchedConstraints, constraint)
		}
	}

	return matchedConstraints, nil
}

// PolicyAppliesTo inspects the constraint's users and groups against the userInfo to determine
// if it is usable by the userInfo.
func PolicyAppliesTo(constraint *policyapi.PodSecurityPolicy, userInfo user.Info) bool {
	for _, user := range constraint.Users {
		if userInfo.GetName() == user {
			return true
		}
	}
	for _, userGroup := range userInfo.GetGroups() {
		if policySupportsGroup(userGroup, constraint.Groups) {
			return true
		}
	}
	return false
}

// policySupportsGroup checks that group is in constraintGroups.
func policySupportsGroup(group string, constraintGroups []string) bool {
	for _, g := range constraintGroups {
		if g == group {
			return true
		}
	}
	return false
}

// getPreallocatedUIDRange retrieves the annotated value from the service account, splits it to make
// the min/max and formats the data into the necessary types for the strategy options.
func getPreallocatedUIDRange(ns *kapi.Namespace) (*int64, *int64, error) {
	annotationVal, ok := ns.Annotations[allocator.UIDRangeAnnotation]
	if !ok {
		return nil, nil, fmt.Errorf("unable to find annotation %s", allocator.UIDRangeAnnotation)
	}
	if len(annotationVal) == 0 {
		return nil, nil, fmt.Errorf("found annotation %s but it was empty", allocator.UIDRangeAnnotation)
	}
	uidBlock, err := uid.ParseBlock(annotationVal)
	if err != nil {
		return nil, nil, err
	}

	var min int64 = int64(uidBlock.Start)
	var max int64 = int64(uidBlock.End)
	glog.V(4).Infof("got preallocated values for min: %d, max: %d for uid range in namespace %s", min, max, ns.Name)
	return &min, &max, nil
}

// getPreallocatedLevel gets the annotated value from the service account.
func getPreallocatedLevel(ns *kapi.Namespace) (string, error) {
	level, ok := ns.Annotations[allocator.MCSAnnotation]
	if !ok {
		return "", fmt.Errorf("unable to find annotation %s", allocator.MCSAnnotation)
	}
	if len(level) == 0 {
		return "", fmt.Errorf("found annotation %s but it was empty", allocator.MCSAnnotation)
	}
	glog.V(4).Infof("got preallocated value for level: %s for selinux options in namespace %s", level, ns.Name)
	return level, nil
}

// requiresPreAllocatedUIDRange returns true if the strategy is must run in range and the min or max
// is not set.
func requiresPreAllocatedUIDRange(constraint *policyapi.PodSecurityPolicy) bool {
	if constraint.RunAsUser.Type != policyapi.RunAsUserStrategyMustRunAsRange {
		return false
	}
	return constraint.RunAsUser.UIDRangeMin == nil && constraint.RunAsUser.UIDRangeMax == nil
}

// requiresPreAllocatedSELinuxLevel returns true if the strategy is must run as and the level is not set.
func requiresPreAllocatedSELinuxLevel(constraint *policyapi.PodSecurityPolicy) bool {
	if constraint.SELinuxContext.Type != policyapi.SELinuxStrategyMustRunAs {
		return false
	}
	if constraint.SELinuxContext.SELinuxOptions == nil {
		return true
	}
	return constraint.SELinuxContext.SELinuxOptions.Level == ""
}

// deduplicateSecurityContextConstraints ensures we have a unique slice of constraints.
func deduplicateSecurityContextConstraints(sccs []*policyapi.PodSecurityPolicy) []*policyapi.PodSecurityPolicy {
	deDuped := []*policyapi.PodSecurityPolicy{}
	added := util.NewStringSet()

	for _, s := range sccs {
		if !added.Has(s.Name) {
			deDuped = append(deDuped, s)
			added.Insert(s.Name)
		}
	}
	return deDuped
}

// logProviders logs what providers were found for the pod as well as any errors that were encountered
// while creating providers.
func logProviders(pod *kapi.Pod, providers []policyprovider.PodSecurityPolicyProvider, providerCreationErrs []error) {
	names := make([]string, len(providers))
	for i, p := range providers {
		names[i] = p.GetPolicyName()
	}
	glog.V(4).Infof("validating pod %s (generate: %s) against providers %s", pod.Name, pod.GenerateName, strings.Join(names, ","))

	for _, err := range providerCreationErrs {
		glog.V(4).Infof("provider creation error: %v", err)
	}
}
