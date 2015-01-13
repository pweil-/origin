package validation

import (
	"testing"
	"github.com/openshift/origin/pkg/route/api"
)

func TestValidateTLSNoTLSTermOk(t *testing.T){
	errs := validateTLS(&api.TLSConfig{
		Termination: "",
	})

	if len(errs) > 0 {
		t.Errorf("Unexpected non-empty error list: %#v", errs)
	}
}

func TestValidateTLSPodTermoOK(t *testing.T){
	errs := validateTLS(&api.TLSConfig{
		Termination: api.TLSTerminationPassthrough,
	})

	if len(errs) > 0 {
		t.Errorf("Unexpected non-empty error list: %#v", errs)
	}
}

func TestValidateTLSReencryptTermOKFile(t *testing.T){
	errs := validateTLS(&api.TLSConfig{
		Termination: api.TLSTerminationReencrypt,
		DestinationCACertificateFile: "abc",
	})

	if len(errs) > 0 {
		t.Errorf("Unexpected non-empty error list: %#v", errs)
	}
}

func TestValidateTLSReencryptTermOKCert(t *testing.T){
	errs := validateTLS(&api.TLSConfig{
		Termination: api.TLSTerminationReencrypt,
		DestinationCACertificate: []byte("abc"),
	})

	if len(errs) > 0 {
		t.Errorf("Unexpected non-empty error list: %#v", errs)
	}
}

func TestValidateTLSEdgeTermOKFiles(t *testing.T){
	errs := validateTLS(&api.TLSConfig{
		Termination: api.TLSTerminationEdge,
		CertificateFile: "abc",
		KeyFile: "abc",
		CACertificateFile: "abc",
	})

	if len(errs) > 0 {
		t.Errorf("Unexpected non-empty error list: %#v", errs)
	}
}

func TestValidateTLSEdgeTermOKCerts(t *testing.T){
	errs := validateTLS(&api.TLSConfig{
		Termination: api.TLSTerminationEdge,
		Certificate: []byte("abc"),
		Key: []byte("abc"),
		CACertificate: []byte("abc"),
	})

	if len(errs) > 0 {
		t.Errorf("Unexpected non-empty error list: %#v", errs)
	}
}

func TestValidateEdgeTermInvalid(t *testing.T){
	testCases := [] struct {
		name string
		cfg api.TLSConfig
	}{
		{"no cert", api.TLSConfig{
			Termination: api.TLSTerminationEdge,
			Key: []byte("abc"),
			CACertificate: []byte("abc"),
		}},
		{"no key", api.TLSConfig{
			Termination: api.TLSTerminationEdge,
			Certificate: []byte("abc"),
			CACertificate: []byte("abc"),
		}},
		{"no ca cert", api.TLSConfig{
			Termination: api.TLSTerminationEdge,
			Certificate: []byte("abc"),
			Key: []byte("abc"),
		}},
	}

	for _, tc := range testCases {
		errs := validateTLS(&tc.cfg)

		//one error for file contents, one error for file name since either can be specified
		if len(errs) != 2 {
			t.Errorf("Unexpected error list encountered for case %v: %#v.  Expected 2 errors, got %v", tc.name, errs, len(errs))
		}
	}
}

func TestValidatePodTermInvalid(t *testing.T){
	testCases := []struct {
		name string
		cfg api.TLSConfig
	}{
		{"cert file", api.TLSConfig{Termination: api.TLSTerminationPassthrough, CertificateFile: "test"}},
		{"cert", api.TLSConfig{Termination: api.TLSTerminationPassthrough, Certificate: []byte("test")}},
		{"key file", api.TLSConfig{Termination: api.TLSTerminationPassthrough, KeyFile: "test"}},
		{"key", api.TLSConfig{Termination: api.TLSTerminationPassthrough, Key: []byte("test")}},
		{"ca cert file", api.TLSConfig{Termination: api.TLSTerminationPassthrough, CACertificateFile: "test"}},
		{"ca cert", api.TLSConfig{Termination: api.TLSTerminationPassthrough, CACertificate: []byte("test")}},
		{"dest cert file", api.TLSConfig{Termination: api.TLSTerminationPassthrough, DestinationCACertificateFile: "test"}},
		{"dest cert", api.TLSConfig{Termination: api.TLSTerminationPassthrough, DestinationCACertificate: []byte("test")}},
	}

	for _, tc := range testCases {
		errs := validateTLS(&tc.cfg)

		if len(errs) != 1 {
			t.Errorf("Unexpected error list encountered for test case %v: %#v.  Expected 1 error, got %v", tc.name, errs, len(errs))
		}
	}
}

func TestValidateReencryptTermInvalid(t *testing.T){
	errs := validateTLS(&api.TLSConfig{
		Termination: api.TLSTerminationReencrypt,
	})

	//one error for file contents, one error for file name since either can be specified
	if len(errs) != 2 {
		t.Errorf("Unexpected error list encountered: %#v.  Expected 2 errors, got %v", errs, len(errs))
	}
}

