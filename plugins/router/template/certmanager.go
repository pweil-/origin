package templaterouter

import (
	"io/ioutil"
	"github.com/golang/glog"
)

type certManager struct {}

func (cm *certManager) writeCertificatesForConfig(config *ServiceAliasConfig) error {
	if len(config.Certificates) > 0 {
		for id, cert := range config.Certificates {
			if err := cm.writeCertificate(id, &cert); err != nil {
				cm.deleteCertificatesForConfig(config)
				return err
			}
		}
	}

	return nil
}
func (cm *certManager) writeCertificate(id string, cert *Certificate) error {
	if len(cert.Contents) > 0 && len(cert.PrivateKey) > 0 {
		//write a single file (required by haproxy, optional for others)
		dat := append(cert.PrivateKey, cert.Contents...)

		fileName := CertDir + cm.createCertFileName(id)
		err := ioutil.WriteFile(fileName, dat, 0644)

		if err != nil {
			glog.Errorf("Error writing certificate file %v: %v", fileName, err)
			return err
		}

		//todo ca certs
	}

	return nil
}

func (cm *certManager) deleteCertificatesForConfig(config *ServiceAliasConfig) error {
	return nil
}

func (cm *certManager) deleteCertificate(cert *Certificate) error {
	return nil
}

func (cm *certManager) createCertFileName(id string) string {
	return id + ".pem"
}
