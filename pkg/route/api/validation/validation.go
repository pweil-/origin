package validation

import (
	errs "github.com/GoogleCloudPlatform/kubernetes/pkg/api/errors"
	routeapi "github.com/openshift/origin/pkg/route/api"
	"fmt"
)

// ValidateRoute tests if required fields in the route are set.
func ValidateRoute(route *routeapi.Route) errs.ValidationErrorList {
	result := errs.ValidationErrorList{}

	if len(route.Host) == 0 {
		result = append(result, errs.NewFieldRequired("host", ""))
	}
	if len(route.ServiceName) == 0 {
		result = append(result, errs.NewFieldRequired("serviceName", ""))
	}

	if errs := validateTLS(&route.TLS); len(errs) != 0{
		result = append(result, errs...)
	}

	return result
}

// ValidateTLS tests fields for different types of TLS combinations are set.  Called
// by ValidateRoute.
func validateTLS(tls *routeapi.TLSConfig) errs.ValidationErrorList {
	result := errs.ValidationErrorList{}

	//no termination, ignore other settings
	if tls.Termination == "" {
		return nil
	}

	//reencrypt must specify pod cert
	if tls.Termination == routeapi.TLSTerminationReencrypt {
		if len(tls.PodCACertificate) == 0 && len(tls.PodCACertificateFile) == 0 {
			result = append(result, errs.NewFieldInvalid("podCACertificateFile", "", "reencrypt termination must specify the podCACertificateFile or the podCACertificate field"))
			result = append(result, errs.NewFieldInvalid("podCACertificate", "", "reencrypt termination must specify the podCACertificateFile or the podCACertificate field"))
		}
	}

	//pod cert should not specify any cert
	if tls.Termination == routeapi.TLSTerminationPod {
		if err := podTerminationCertError("certificateFile", []byte(tls.CertificateFile)); err != nil {
			result = append(result, err)
		}

		if err := podTerminationCertError("certificate", tls.Certificate); err != nil {
			result = append(result, err)
		}

		if err :=  podTerminationCertError("keyFile", []byte(tls.KeyFile)); err != nil {
			result = append(result, err)
		}

		if err := podTerminationCertError("key", tls.Key); err != nil {
			result = append(result, err)
		}

		if err := podTerminationCertError("caCertificateFile", []byte(tls.CACertificateFile)); err != nil {
			result = append(result, err)
		}

		if err := podTerminationCertError("caCertificate", tls.CACertificate); err != nil {
			result = append(result, err)
		}

		if err := podTerminationCertError("podCACertificateFile", []byte(tls.PodCACertificateFile)); err != nil {
			result = append(result, err)
		}

		if err :=podTerminationCertError("podCACertificate", tls.PodCACertificate); err != nil {
			result = append(result, err)
		}
	}

	//edge cert should specify cert, key, and cacert
	if tls.Termination == routeapi.TLSTerminationEdge {
		result = append(result, edgeTerminationCertError("certificate", "certificateFile", tls.Certificate, tls.CertificateFile)...)
		result = append(result, edgeTerminationCertError("key", "keyFile", tls.Key, tls.KeyFile)...)
		result = append(result, edgeTerminationCertError("caCertificate", "caCertificateFile", tls.CACertificate, tls.CACertificateFile)...)


		if len(tls.PodCACertificate) > 0 {
			result = append(result, errs.NewFieldInvalid("podCACertificate", "", "edge termination does not support pod certificates"))
		}
		if len(tls.PodCACertificateFile) > 0 {
			result = append(result, errs.NewFieldInvalid("podCACertificateFile", "", "edge termination does not support pod certificates"))
		}
	}

	//TODO: unsupported field
	if len(tls.KeyPassPhrase) > 0 {
		result = append(result, errs.NewFieldNotSupported("keyPassPhrase is not yet supported", ""))
	}

	return result
}

//helper to return a standard error for the pod term validations
func podTerminationCertError(field string, value []byte) *errs.ValidationError {
	if len(value) > 0 {
		return errs.NewFieldInvalid(field, "", "specifying certificates with pod termination is not valid")
	}
	return nil
}

//helper to return a standard error for edge termination validations
func edgeTerminationCertError(contentsField string, fileField string, contents []byte, file string) []error {
	if len(contents) == 0 && len(file) == 0 {
		err := fmt.Sprintf("edge termination requires %s or %s", contentsField, fileField)
		return []error{errs.NewFieldInvalid(contentsField, "", err), errs.NewFieldInvalid(fileField, "", err)}
	}

	return nil
}
