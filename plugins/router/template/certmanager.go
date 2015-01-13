package templaterouter

import (
	"io/ioutil"
	"github.com/golang/glog"

	routeapi "github.com/openshift/origin/pkg/route/api"
)

type certManager struct {}

// write certificates for edge and reencrypt termination by appending the key, cert, and ca cert
// into a single <host>.pem file.  Also write <host>_pod.pem file if it is reencrypt termination
func (cm *certManager) writeCertificatesForConfig(config *ServiceAliasConfig) error {
	if len(config.Certificates) > 0 {
		if config.TLSTermination == routeapi.TLSTerminationEdge || config.TLSTermination == routeapi.TLSTerminationReencrypt{
			certObj, ok := config.Certificates[config.Host]

			if ok {
				cert := certObj.Contents
				key := certObj.PrivateKey
				dat := append(key, cert...)

				caCertObj, caOk := config.Certificates[config.Host + CaCertPostfix]

				if caOk {
					dat = append(dat, caCertObj.Contents...)
				}

				cm.writeCertificate(CertDir, config.Host, dat)
			}
		}

		if config.TLSTermination == routeapi.TLSTerminationReencrypt {
			destCertKey := config.Host + DestCertPostfix
			destCert, ok := config.Certificates[destCertKey]

			if ok {
				cm.writeCertificate(CaCertDir, destCertKey, destCert.Contents)
			}
		}
	}

	return nil
}

// writeCertificate writes to disk
func (cm *certManager) writeCertificate(directory string, id string, cert []byte) error {
	fileName := directory + id + ".pem"
	err := ioutil.WriteFile(fileName, cert, 0644)

	if err != nil {
		glog.Errorf("Error writing certificate file %v: %v", fileName, err)
		return err
	}

	return nil
}

func (cm *certManager) deleteCertificatesForConfig(config *ServiceAliasConfig) error {
	//TODO
	return nil
}
