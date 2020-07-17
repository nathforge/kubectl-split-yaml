package cmd

import (
	"fmt"
	"html/template"
	"io"
	"os"

	"github.com/nathforge/kubectl-split-yaml/internal/saveresources"
	"github.com/nathforge/kubectl-split-yaml/internal/walkresources"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	splitYAMLExample = `  # save deployment resources
  kubectl get deploy -o yaml | %[1]s split-yaml -p deployments

  # split single file into multiple files
  %[1]s split-yaml -f bigfile.yaml -p smallerfiles`
)

type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer
}

type SplitYAMLOptions struct {
	ioStreams     IOStreams
	inputFilename string
	outputPath    string
	template      string
	quiet         bool
}

func NewSplitYAMLOptions(streams IOStreams) *SplitYAMLOptions {
	return &SplitYAMLOptions{
		ioStreams:     streams,
		inputFilename: "-",
		outputPath:    "split-yaml",
		template:      "{{.apiVersion}}--{{.kind}}/{{.namespace}}--{{.name}}.yaml",
		quiet:         false,
	}
}

func NewCmdSplitYAML(streams IOStreams) *cobra.Command {
	o := NewSplitYAMLOptions(streams)

	cmd := &cobra.Command{
		Use:          "kubectl-split-yaml [flags]",
		Short:        "Split Kubernetes YAML output into one file per resource",
		Example:      fmt.Sprintf(splitYAMLExample, "kubectl"),
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Validate(); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				// Is this a YAML decoding error?
				// Add a hint - user may have forgotten to pass `-o yaml` to
				// `kubectl get`
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

	cmd.Flags().StringVarP(&o.inputFilename, "input", "f", o.inputFilename, "Input filename; use \"-\" for stdin")
	cmd.Flags().StringVarP(&o.outputPath, "output-path", "p", o.outputPath, "Output path")
	cmd.Flags().StringVarP(&o.template, "template", "t", o.template, "Filename template")
	cmd.Flags().BoolVar(&o.quiet, "quiet", o.quiet, "Don't show status messages")

	return cmd
}

// Validate ensures that all required arguments and flag values are provided
func (o *SplitYAMLOptions) Validate() error {
	return nil
}

func (o *SplitYAMLOptions) Run() error {
	var inputReader io.Reader
	if o.inputFilename == "-" {
		inputReader = o.ioStreams.In
	} else {
		var err error
		inputReader, err = os.Open(o.inputFilename)
		if err != nil {
			return err
		}
	}

	filenameTemplate, err := template.New("").Parse(o.template)
	if err != nil {
		return err
	}

	saveResources, err := saveresources.New(saveresources.Options{
		OutputPath:       o.outputPath,
		FilenameTemplate: filenameTemplate,
		OnStartFile: func(filename string) {
			if !o.quiet {
				fmt.Fprintf(o.ioStreams.Out, "%s\n", filename)
			}
		},
	})
	if err != nil {
		return err
	}

	// Warn user if input appears to be a terminal
	if !o.quiet && inputReader == os.Stdin {
		fi, err := os.Stdin.Stat()
		if err != nil {
			return err
		}
		if (fi.Mode() & os.ModeCharDevice) != 0 {
			_, _ = o.ioStreams.ErrOut.Write([]byte(
				"NOTE: kubectl-split-yaml is currently reading from stdin.\n" +
					"      Other options include passing a filename, e.g:\n" +
					"          $ kubectl split-yaml -f resources.yaml\n" +
					"      or piping input, e.g:\n" +
					"          $ kubectl get all -o yaml | kubectl split-yaml\n" +
					"Press Ctrl+C to exit\n",
			))
		}
	}

	return walkresources.WalkReader(inputReader, func(resource map[interface{}]interface{}) error {
		return saveResources.Save(resource)
	})
}
