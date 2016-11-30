package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"

	log "github.com/Sirupsen/logrus"
	"k8s.io/kubernetes/pkg/util/logs"

	cmdutil "github.com/openshift/origin/pkg/cmd/util"
	"github.com/openshift/origin/pkg/cmd/util/serviceability"

	// install all APIs
	_ "github.com/openshift/origin/pkg/api/install"
	"github.com/openshift/origin/pkg/image/policy"
	"github.com/spf13/cobra"
	_ "k8s.io/kubernetes/pkg/api/install"
	_ "k8s.io/kubernetes/pkg/apis/extensions/install"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()
	defer serviceability.BehaviorOnPanic(os.Getenv("OPENSHIFT_ON_PANIC"))()
	defer serviceability.Profile(os.Getenv("OPENSHIFT_PROFILE")).Stop()
	startProfiler()

	runtime.GOMAXPROCS(runtime.NumCPU())

	cmd := &cobra.Command{
		Use:   "imagepolicy",
		Short: "Image policy server",
		Run: func(cmd *cobra.Command, args []string) {
			server := policy.NewImagePolicyServer()
			log.Fatal(server.Run())
		},
	}

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func startProfiler() {
	if cmdutil.Env("OPENSHIFT_PROFILE", "") == "web" {
		go func() {
			runtime.SetBlockProfileRate(1)
			profilePort := cmdutil.Env("OPENSHIFT_PROFILE_PORT", "6060")
			profileHost := cmdutil.Env("OPENSHIFT_PROFILE_HOST", "127.0.0.1")
			log.Infof(fmt.Sprintf("Starting profiling endpoint at http://%s:%s/debug/pprof/", profileHost, profilePort))
			log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%s", profileHost, profilePort), nil))
		}()
	}
}
