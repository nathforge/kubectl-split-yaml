package main

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"

	"github.com/go-yaml/yaml"
	"github.com/urfave/cli/v2"

	"github.com/nathforge/kubectl-save/internal/saveresources"
	"github.com/nathforge/kubectl-save/internal/walkresources"
)

func main() {
	app := &cli.App{
		Name:      "kubectl-save",
		Usage:     "save Kubernetes YAML resources into multiple files",
		ArgsUsage: "output-path [file...]",
		Description: "Save Kubernetes YAML resources into multiple files\n\n" +
			"EXAMPLES:\n" +
			"   $ kubectl save MY-OUTPUT-PATH MY-RESOURCES.yaml\n" +
			"   $ kubectl get all -o yaml | kubectl save MY-OUTPUT-PATH",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "template",
				Value: "{{.apiVersion}}--{{.kind}}/{{.namespace}}--{{.name}}.yaml",
			},
			&cli.BoolFlag{
				Name:  "quiet",
				Value: false,
			},
		},
		Action: func(c *cli.Context) error {
			args := c.Args().Slice()

			if len(args) == 0 {
				return cli.Exit("Error: output path is required", 1)
			}

			outputPath := args[0]
			filenames := args[1:]
			templateStr := c.String("template")
			quiet := c.Bool("quiet")

			return run(outputPath, filenames, templateStr, quiet)
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.SetFlags(0)
		log.Fatal(err)
	}
}

func run(outputPath string, filenames []string, templateStr string, quiet bool) error {
	filenameTemplate, err := template.New("").Parse(templateStr)
	if err != nil {
		return err
	}

	saveResources, err := saveresources.New(saveresources.Options{
		OutputPath:       outputPath,
		FilenameTemplate: filenameTemplate,
		OnStartFile: func(filename string) {
			if !quiet {
				fmt.Println(filename)
			}
		},
	})
	if err != nil {
		return err
	}

	return callbackReadersFromFilenames(filenames, func(f io.Reader) error {
		return callbackDocsFromReader(f, func(doc map[interface{}]interface{}) error {
			return walkresources.Walk(doc, func(resource map[interface{}]interface{}) error {
				return saveResources.Save(resource)
			})
		})
	})
}

func callbackReadersFromFilenames(filenames []string, callback func(io.Reader) error) error {
	callbackForFilename := func(filename string) error {
		f, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer f.Close()
		return callback(f)
	}

	if len(filenames) == 0 {
		if err := callback(os.Stdin); err != nil {
			return fmt.Errorf("<stdin>: %w", err)
		}
	}

	for _, filename := range filenames {
		if err := callbackForFilename(filename); err != nil {
			return fmt.Errorf("%s: %w", filename, err)
		}
	}

	return nil
}

func callbackDocsFromReader(file io.Reader, callback func(map[interface{}]interface{}) error) error {
	decoder := yaml.NewDecoder(file)
	for {
		doc := map[interface{}]interface{}{}
		if err := decoder.Decode(doc); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		if err := callback(doc); err != nil {
			return err
		}
	}
	return nil
}
