package router

import (
	"errors"
	"fmt"

	"github.com/golang/glog"
	"github.com/spf13/cobra"

	"github.com/openshift/origin/pkg/cmd/util"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"
	"github.com/openshift/origin/pkg/router"
	controllerfactory "github.com/openshift/origin/pkg/router/controller/factory"
	templateplugin "github.com/openshift/origin/plugins/router/template"
	f5plugin "github.com/openshift/origin/plugins/router/f5"
)

const longCommandDesc = `
Start an OpenShift router

This command launches a router connected to your OpenShift master. The router listens for routes and endpoints
created by users and keeps a local router configuration up to date with those changes.
`

// routerConfig represents a container for all other router config types.  Any global config belongs on this level
type routerConfig struct {
	//Config is global flags that are passed to all client commands that need to connect to master
	Config *clientcmd.Config

	//Type is the type of plugin being created
	Type	string

	//TemplateRouterConfig holds config items for template router plugins
	TemplateRouterConfig *templateRouterConfig

	//F5RouterConfig holds the configuration information for f5 router plugins
	F5RouterConfig *f5RouterConfig
}

// templateRouterConfig is the config necessary to start a template router plugin
type templateRouterConfig struct {
	TemplateFile string
	ReloadScript string
}

// f5RouterConfig isthe config necessar to start an f5 router plugin
type f5RouterConfig struct {}

const (
	//routerTypeF5 is the type value for an f5 router plugin
	routerTypeF5 = "f5"
	//routerTypeTemplate is the type value for a template router plugin
	routerTypeTemplate = "template"
)

// NewCommndTemplateRouter provides CLI handler for the template router backend
func NewCommandTemplateRouter(name string) *cobra.Command {
	cfg := &routerConfig{
		Config: clientcmd.NewConfig(),
	}

	cmd := &cobra.Command{
		Use:   fmt.Sprintf("%s%s", name, clientcmd.ConfigSyntax),
		Short: "Start an OpenShift router",
		Long:  longCommandDesc,
		Run: func(c *cobra.Command, args []string) {
			if !isValidType(cfg.Type){
				glog.Fatal("Invalid type specified.  Valid values are template, f5")
			}

			var plugin router.Plugin
			var err error

			switch cfg.Type{
			case "f5":
				plugin, err = makeF5Plugin(cfg)
			case "template":
				plugin, err = makeTemplatePlugin(cfg)
			}

			if err != nil {
				glog.Fatal(err)
			}

			//should never get here with the isValidType call above
			if plugin == nil {
				glog.Fatal("Plugin was never initialized")
			}

			if err := start(cfg.Config, plugin); err != nil {
				glog.Fatal(err)
			}
		},
	}

	flag := cmd.Flags()
	// Binds for generic config
	cfg.Config.Bind(flag)
	flag.StringVar(&cfg.Type, "type", util.Env("ROUTER_TYPE", ""), "The type of router to create.  Valid values: template, f5")

	// Binds for Template Router Config
	flag.StringVar(&cfg.TemplateRouterConfig.TemplateFile, "template", util.Env("TEMPLATE_FILE", ""), "The path to the template file to use")
	flag.StringVar(&cfg.TemplateRouterConfig.ReloadScript, "reload", util.Env("RELOAD_SCRIPT", ""), "The path to the reload script to use")

	return cmd
}

// isValidType checks that the router type specified is a known type
func isValidType(t string) bool {
	return t == routerTypeF5 || t == routerTypeTemplate
}

// makeTemplatePlugin creates a template router plugin
func makeTemplatePlugin(cfg *routerConfig) (*templateplugin.TemplatePlugin, error) {
	if cfg.TemplateRouterConfig.TemplateFile == "" {
		return nil, errors.New("Template file must be specified")
	}

	if cfg.TemplateRouterConfig.ReloadScript == "" {
		return nil, errors.New("Reload script must be specified")
	}

	return templateplugin.NewTemplatePlugin(cfg.TemplateRouterConfig.TemplateFile, cfg.TemplateRouterConfig.ReloadScript)
}

// makeF5Plugin makes an F5 router plugin
func makeF5Plugin(cfg *routerConfig) (*f5plugin.F5Plugin, error){
	//validate f5 config as necessary
	return f5plugin.NewF5Plugin("")
}

// start launches the router.
func start(cfg *clientcmd.Config, plugin router.Plugin) error {
	kubeClient, osClient, err := cfg.Clients()
	if err != nil {
		return err
	}

	factory := controllerfactory.RouterControllerFactory{kubeClient, osClient}
	controller := factory.Create(plugin)
	controller.Run()

	select {}

	return nil
}
