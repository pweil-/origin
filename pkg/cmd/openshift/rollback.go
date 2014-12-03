package openshift

import (
	"flag"
	"github.com/golang/glog"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"
	"github.com/spf13/cobra"

	deployapi "github.com/openshift/origin/pkg/deploy/api"
	"encoding/json"
	"io/ioutil"
)

type config struct {
	ClientConfig         *clientcmd.Config
	RollbackConfigName   string
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
	flag.StringVar(&cfg.RollbackConfigName, "f", "", "path/name of the rollback config file")

	return cmd
}

func rollback(cfg *config, args []string) error {
	glog.Info("------------------ Executing a rollback ----------------------")
	_, osClient, err := cfg.ClientConfig.Clients()

	if err != nil {
		return err
	}

	//read the file - will be handled by the api in the future
	glog.V(4).Infof("Reading rollback config")
	rollbackConfig, err := ReadRollbackFile(cfg.RollbackConfigName)
	if err != nil {
		return err
	}

	//get the current config - existing api call
	glog.V(4).Infof("Finding current deploy config named %s", rollbackConfig.ObjectMeta.Name)
	currentConfig, err := osClient.DeploymentConfigs("").Get(rollbackConfig.ObjectMeta.Name)
	if err != nil {
		return err
	}


	//get the old config - existing api call
	glog.V(4).Infof("Finding rollback deployment with name %s", rollbackConfig.Rollback.To)
	oldConfig, err := osClient.Deployments("").Get(rollbackConfig.Rollback.To)
	if err != nil {
		return err
	}

	//replace current template with old - new rollback functionality
	currentConfig.Template.ControllerTemplate = oldConfig.ControllerTemplate

	//submit - existing api call
	osClient.DeploymentConfigs("").Update(currentConfig)

	glog.Info("-------------------  Rollback Complete  ----------------------")
	return nil
}

func ReadRollbackFile(fileName string) (*deployapi.DeploymentConfig, error) {
	data, err := ioutil.ReadFile(fileName)

	if err != nil {
		return nil, err
	}

	rollbackConfig := &deployapi.DeploymentConfig{}
	err = json.Unmarshal(data, rollbackConfig)

	if err != nil {
		return nil, err
	}

	return rollbackConfig, nil
}
