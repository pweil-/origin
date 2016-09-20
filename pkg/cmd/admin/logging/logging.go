package logging

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	kcmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"

	"github.com/openshift/origin/pkg/cmd/admin"
	cmdutil "github.com/openshift/origin/pkg/cmd/util"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"
	configcmd "github.com/openshift/origin/pkg/config/cmd"
)

const loggingExample = `
`

const loggingLongDesc = `
Install the Aggregated Logging (EFK) components

This command installs EFK components to support the cluster by creating the
required resources to support Aggregated Logging.
`

// Config contains the configuration parameters
// to deploy the aggregated logging stack
type Config struct {
	Action configcmd.BulkAction

	DryRun           bool
	ImagesPrefix     string
	UseLatestImages  bool
	ImagesPullSecret string

	CACrt               string
	CAKey               string
	CuratorNodeSelector []string
	ESClusterSize       uint
	ESNodeSelector      []string
	ESPvcDynamic        bool
	ESPvcPrefix         string
	ESPvcSize           string
	ESUseLocalStorage   bool
	KibanaCert          string
	KibanaKey           string
	KibanaNodeSelector  []string
	KibanaHostname      string
	PublicMasterURL     string
	ServerTLSJSON       string
	StorageGroup        uint

	CuratorOpsNodeSelector []string
	ESOpsClusterSize       uint
	ESOpsNodeSelector      []string
	ESOpsPvcPrefix         string
	ESOpsPvcSize           string
	FluentdOpsNodeSelector []string
	KibanaOpsHostname      string
	KibanaOpsCert          string
	KibanaOpsKey           string
	KibanaOpsNodeSelector  []string
}

// NewCmdLogging implements the OpenShift CLI admin logging command
func NewCmdLogging(f *clientcmd.Factory, parentName, name string, out io.Writer) *cobra.Command {
	cfg := &Config{}
	cmd := &cobra.Command{
		Use:     fmt.Sprintf("%s", name),
		Short:   "Install Aggregated Logging Stack",
		Long:    loggingLongDesc,
		Example: fmt.Sprintf(loggingExample),
		Run: func(cmd *cobra.Command, args []string) {
			err := runCmdLogging(f, cmd, out, cfg, args)
			if err != cmdutil.ErrExit {
				kcmdutil.CheckErr(err)
			} else {
				os.Exit(1)
			}
		},
	}

	cmd.Flags().BoolVar(&cfg.DryRun, admin.FlagDryRun, false, "The the result of the operation without executing it.")
	cmd.Flags().StringVar(&cfg.ImagesPrefix, admin.FlagImages, "openshift/origin-", "The image prefix to use for retrieving the images. This will be used or all components (e.g. openshift/origin-fluentd:v1.3)")
	cmd.Flags().StringVar(&cfg.ImagesPullSecret, admin.FlagImagesPullSecret, "", "The name of an existing secret to be used for pulling component images from an authenticated registry.")

	cmd.Flags().StringVar(&cfg.CACrt, "ca-crt", "", "The filename to a certificate for a CA that will be used to sign any generated certificates.")
	cmd.Flags().StringVar(&cfg.CAKey, "ca-key", "", "The filename to a key that matches the certificate specified by --ca-crt")
	cmd.Flags().StringSliceVar(&cfg.CuratorNodeSelector, "curator-nodeselector", []string{}, "A node selector that specifies which nodes are eligible targets for deploying curator instances.")
	cmd.Flags().UintVar(&cfg.ESClusterSize, "es-cluster-size", 1, "The number of Elasticsearch nodes to deploy. A minimum of 3 is required for redundacy.")
	cmd.Flags().StringSliceVar(&cfg.ESNodeSelector, "es-nodeselector", []string{}, "A node selector that specifies which nodes are eligible targets for deploying Elasticsearch instances.")
	cmd.Flags().BoolVar(&cfg.ESPvcDynamic, "es-pvc-dynamic", false, "Set to true to dynamically provision the pvc backing storage (if available to the cluster)")
	cmd.Flags().StringVar(&cfg.ESPvcPrefix, "es-pvc-prefix", "logging-es", "Prefix for the names of the pvc for backing storage. A number will be appended per Elasticsearch instance (e.g. logging-es-1).  The pvc will be created if it does not exist with the size defined by --es-pvc-size.")
	cmd.Flags().StringVar(&cfg.ESPvcSize, "es-pvc-size", "", "The size of the pvc to create per Elasticsearch instance (e.g. 100G).  No pvc will be created if ommited and ephemeral volumes are used instead.")
	cmd.Flags().BoolVar(&cfg.ESUseLocalStorage, "es-use-local-storage", false, "Prepare Elasticsearch with the appropriate permissions to use local storage (i.e direct volume or pvc).")
	cmd.Flags().StringVar(&cfg.KibanaHostname, "kibana-hostname", "kibana.example.com", "The external host name for web clients to reach Kibana.")
	cmd.Flags().StringVar(&cfg.KibanaCert, "kibana-crt", "", "The filename to a browser facing certificate to the Kibana user interface. Generated if not provided.")
	cmd.Flags().StringVar(&cfg.KibanaKey, "kibana-key", "", "The filename to a key to be used with the Kibana certificate.")
	cmd.Flags().StringSliceVar(&cfg.KibanaNodeSelector, "kibana-nodeselector", []string{}, "A node selector that specifies which nodes are eligible targets for deploying Kibana instances.")
	cmd.Flags().StringVar(&cfg.PublicMasterURL, "public-master-url", "", "The external URL for the master used to perform OAuth authorizations (e.g. accessing Kibana).  Defaults to the value in kubeconfig.")
	cmd.Flags().StringVar(&cfg.ServerTLSJSON, "server-tls-json", "", "The filename to a JSON file specifying Node.js TLS options to override the Kibana proxy server defaults.")
	cmd.Flags().UintVar(&cfg.StorageGroup, "storage-group", 65534, "The number of a supplemental group ID for access to Elasticsearch storage volumes; backing volumes should allow access by this group ID.")

	cmd.Flags().StringSliceVar(&cfg.CuratorOpsNodeSelector, "curator-ops-nodeselector", []string{}, "A node selector that specifies which nodes are eligible targets for deploying curator instances to support operations.")
	cmd.Flags().UintVar(&cfg.ESOpsClusterSize, "es-ops-cluster-size", 1, "The number of Elasticsearch nodes to deploy to support operations. A minimum of 3 is required for redundacy.")
	cmd.Flags().StringSliceVar(&cfg.ESOpsNodeSelector, "es-nodeselector", []string{}, "A node selector that specifies which nodes are eligible targets for deploying Elasticsearch instances to support operations.")
	cmd.Flags().StringVar(&cfg.ESOpsPvcPrefix, "es-ops-pvc-prefix", "logging-ops-es", "Prefix for the names of the pvc for backing storage to support operations. A number will be appended per Elasticsearch instance (e.g. logging-es-1).  The pvc will be created if it does not exist with the size defined by --es-pvc-size.")
	cmd.Flags().StringVar(&cfg.ESOpsPvcSize, "es-ops-pvc-size", "", "The size of the pvc to create per Elasticsearch instance (e.g. 100G) to support operations.  No pvc will be created if ommited and ephemeral volumes are used instead.")
	cmd.Flags().StringVar(&cfg.KibanaOpsHostname, "kibana-ops-hostname", "kibana.ops.example.com", "The external host name for web clients to reach Kibana to support operations.")
	cmd.Flags().StringVar(&cfg.KibanaOpsCert, "kibana-ops-crt", "", "The filename to a browser facing certificate to the Kibana user interface to support operations. Generated if not provided.")
	cmd.Flags().StringVar(&cfg.KibanaOpsKey, "kibana-ops-key", "", "The filename to a key to be used with the Kibana ops certificate.")
	cmd.Flags().StringSliceVar(&cfg.KibanaOpsNodeSelector, "kibana-ops-nodeselector", []string{}, "A node selector that specifies which nodes are eligible targets for deploying Kibana instances to support operations.")

	cfg.Action.BindForOutput(cmd.Flags())
	return cmd
}

func runCmdLogging(f *clientcmd.Factory, cmd *cobra.Command, out io.Writer, cfg *Config, args []string) error {
	return nil
}
