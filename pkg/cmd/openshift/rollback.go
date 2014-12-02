package openshift

import (
	"flag"
	"github.com/golang/glog"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"
	"github.com/spf13/cobra"
)

type config struct {
	ClientConfig         *clientcmd.Config
	DeploymentConfigName string
	RollbackConfigName   string
	Namespace            string
}

func NewCommandRollback() *cobra.Command {
	flag.Set("v", "4")

	cfg := &config{
		ClientConfig: clientcmd.NewConfig(),
	}

	cmd := &cobra.Command{
		Use: "rollback",
		Run: func(c *cobra.Command, args []string) {
			if err := rollback(cfg, args); err != nil {
				glog.Fatal(err)
			}
		},
	}

	flag := cmd.Flags()
	cfg.ClientConfig.Bind(flag)
	flag.StringVar(&cfg.DeploymentConfigName, "from", "", "name of the deployment config being rolled back")
	flag.StringVar(&cfg.RollbackConfigName, "to", "", "name of the deployment config to rollback to")
	flag.StringVar(&cfg.Namespace, "ns", "", "The deployment namespace")

	return cmd
}

func rollback(cfg *config, args []string) error {
	glog.Info("------------------ Executing a rollback ----------------------")
	glog.Infof("\t namespace                    : %s", cfg.Namespace)
	glog.Infof("\t deployment config name (from): %s", cfg.DeploymentConfigName)
	glog.Infof("\t rollback config name  (to)   : %s", cfg.RollbackConfigName)

	//do some api posts here that we'd aggregate into its own api call

	glog.Info("-------------------  Rollback Complete  ----------------------")
	return nil
}
