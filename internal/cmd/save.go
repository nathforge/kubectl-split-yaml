package cmd

import (
	"fmt"
	"html/template"
	"os"

	"github.com/nathforge/kubectl-save/internal/saveresources"
	"github.com/nathforge/kubectl-save/internal/walkresources"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	saveExample = `  # save deployment resources
  kubectl get deploy -o yaml | %[1] save`
)

type SaveOptions struct {
	genericclioptions.IOStreams
	outputPath string
	template   string
	quiet      bool
}

func NewSaveOptions(streams genericclioptions.IOStreams) *SaveOptions {
	return &SaveOptions{
		IOStreams: streams,
		template:  "{{.apiVersion}}--{{.kind}}/{{.namespace}}--{{.name}}.yaml",
		quiet:     false,
	}
}

func NewCmdSave(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewSaveOptions(streams)

	cmd := &cobra.Command{
		Use:          "save [output-path] [flags]",
		Short:        "Split Kubernetes YAML output into one file per resource",
		Example:      fmt.Sprintf(saveExample, "kubectl"),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				if _, ok := err.(*yaml.TypeError); ok {
					return fmt.Errorf(
						"%w\n\n"+
							"Is your input in YAML format?\n"+
							"`kubectl get` can output YAML with the `-o yaml` option",
						err,
					)
				}
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&o.template, "template", o.template, "Filename template")
	cmd.Flags().BoolVar(&o.quiet, "quiet", o.quiet, "Don't display progress messages")

	return cmd
}

func (o *SaveOptions) Complete(cmd *cobra.Command, args []string) error {
	switch len(args) {
	case 0:
		o.outputPath = "."
	case 1:
		o.outputPath = args[0]
	default:
		return fmt.Errorf("either one or no arguments are allowed")
	}
	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *SaveOptions) Validate() error {
	return nil
}

func (o *SaveOptions) Run() error {
	filenameTemplate, err := template.New("").Parse(o.template)
	if err != nil {
		return err
	}

	saveResources, err := saveresources.New(saveresources.Options{
		OutputPath:       o.outputPath,
		FilenameTemplate: filenameTemplate,
		OnStartFile: func(filename string) {
			if !o.quiet {
				fmt.Println(filename)
			}
		},
	})
	if err != nil {
		return err
	}

	if fi, err := os.Stdin.Stat(); err != nil {
		return err
	} else {
		if !o.quiet && (fi.Mode()&os.ModeCharDevice) != 0 {
			os.Stderr.Write([]byte(
				"NOTE: kubectl-split-yaml expects input from stdin\n" +
					"      e.g $ kubectl split-yaml <resources.yaml\n" +
					"          $ kubectl get all -o yaml | kubectl split-yaml\n",
			))
		}
	}

	return walkresources.WalkReader(os.Stdin, func(resource map[interface{}]interface{}) error {
		return saveResources.Save(resource)
	})
}
