package bootstrappolicy

import (
	kapi "k8s.io/kubernetes/pkg/api"

	sccapi "github.com/openshift/origin/pkg/security/policy/api"
)

const (
	// PodSecurityPolicyPrivileged is used as the name for the system default privileged policy.
	PodSecurityPolicyPrivileged = "privileged"
	// PodSecurityPolicyRestricted is used as the name for the system default restricted policy.
	PodSecurityPolicyRestricted = "restricted"
)

// GetBootstrapSecurityContextConstraints returns the slice of default SecurityContextConstraints
// for system bootstrapping.
func GetBootstrapPodSecurityPolicy(buildControllerUsername string) []sccapi.PodSecurityPolicy {
	constraints := []sccapi.PodSecurityPolicy{
		{
			ObjectMeta: kapi.ObjectMeta{
				Name: PodSecurityPolicyPrivileged,
			},
			AllowPrivilegedContainer: true,
			AllowHostDirVolumePlugin: true,
			AllowHostNetwork:         true,
			AllowHostPorts:           true,
			SELinuxContext: sccapi.SELinuxContextStrategyOptions{
				Type: sccapi.SELinuxStrategyRunAsAny,
			},
			RunAsUser: sccapi.RunAsUserStrategyOptions{
				Type: sccapi.RunAsUserStrategyRunAsAny,
			},
			Users:  []string{buildControllerUsername},
			Groups: []string{ClusterAdminGroup, NodesGroup},
		},
		{
			ObjectMeta: kapi.ObjectMeta{
				Name: PodSecurityPolicyRestricted,
			},
			SELinuxContext: sccapi.SELinuxContextStrategyOptions{
				// This strategy requires that annotations on the namespace which will be populated
				// by the admission controller.  If namespaces are not annotated creating the strategy
				// will fail.
				Type: sccapi.SELinuxStrategyMustRunAs,
			},
			RunAsUser: sccapi.RunAsUserStrategyOptions{
				// This strategy requires that annotations on the namespace which will be populated
				// by the admission controller.  If namespaces are not annotated creating the strategy
				// will fail.
				Type: sccapi.RunAsUserStrategyMustRunAsRange,
			},
			Groups: []string{AuthenticatedGroup},
		},
	}
	return constraints
}
