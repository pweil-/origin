package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"

	"github.com/golang/glog"
	"github.com/spf13/cobra"

	kcmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/util/logs"

	cmdutil "github.com/openshift/origin/pkg/cmd/util"
	"github.com/openshift/origin/pkg/cmd/util/serviceability"

	// install all APIs
	_ "github.com/openshift/origin/pkg/api/install"
	"github.com/openshift/origin/pkg/image/policy"
	_ "k8s.io/kubernetes/pkg/api/install"
	_ "k8s.io/kubernetes/pkg/apis/extensions/install"
)

type Config struct {
	cert          string
	key           string
	caCert        string
	listenAddress string

	certBytes   []byte
	keyBytes    []byte
	caCertBytes []byte
}

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()
	defer serviceability.BehaviorOnPanic(os.Getenv("OPENSHIFT_ON_PANIC"))()
	defer serviceability.Profile(os.Getenv("OPENSHIFT_PROFILE")).Stop()
	startProfiler()

	runtime.GOMAXPROCS(runtime.NumCPU())

	cfg := Config{
		listenAddress: "0.0.0.0:443",
	}

	cmd := &cobra.Command{
		Use:   "imagepolicy",
		Short: "Image policy server",
		Run: func(cmd *cobra.Command, args []string) {
			if err := cfg.Complete(); err != nil {
				kcmdutil.CheckErr(kcmdutil.UsageError(cmd, err.Error()))
			}
			server := policy.NewImagePolicyServer(cfg.certBytes, cfg.keyBytes, cfg.caCertBytes, cfg.listenAddress)
			kcmdutil.CheckErr(server.Run())
		},
	}

	cmd.Flags().StringVar(&cfg.cert, "cert", cfg.cert, "Certificate for serving tls")
	cmd.Flags().StringVar(&cfg.key, "key", cfg.key, "Key for serving tls")
	cmd.Flags().StringVar(&cfg.caCert, "cacert", cfg.caCert, "CA certificate for serving tls")
	cmd.Flags().StringVar(&cfg.listenAddress, "listen", cfg.listenAddress, "CA certificate for serving tls")

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func (cfg *Config) Complete() error {
	var parseErr error

	if len(cfg.cert) == 0 {
		return fmt.Errorf("certificate is required")
	} else {
		if cfg.certBytes, parseErr = ioutil.ReadFile(cfg.cert); parseErr != nil {
			return parseErr
		}
	}

	if len(cfg.key) == 0 {
		return fmt.Errorf("key is required")
	} else {
		if cfg.keyBytes, parseErr = ioutil.ReadFile(cfg.key); parseErr != nil {
			return parseErr
		}
	}

	if len(cfg.caCert) > 0 {
		if cfg.caCertBytes, parseErr = ioutil.ReadFile(cfg.caCert); parseErr != nil {
			return parseErr
		}
	}

	return nil
}

func startProfiler() {
	if cmdutil.Env("OPENSHIFT_PROFILE", "") == "web" {
		go func() {
			runtime.SetBlockProfileRate(1)
			profilePort := cmdutil.Env("OPENSHIFT_PROFILE_PORT", "6060")
			profileHost := cmdutil.Env("OPENSHIFT_PROFILE_HOST", "127.0.0.1")
			glog.Infof(fmt.Sprintf("Starting profiling endpoint at http://%s:%s/debug/pprof/", profileHost, profilePort))
			glog.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%s", profileHost, profilePort), nil))
		}()
	}
}
