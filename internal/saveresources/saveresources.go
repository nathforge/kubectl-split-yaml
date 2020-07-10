package saveresources

import (
	"bytes"
	"errors"
	"html/template"
	"os"
	"path/filepath"
	"regexp"

	"github.com/go-yaml/yaml"
)

const defaultNamespace = "default"

var unsafeBasenameCharsRegexp *regexp.Regexp

func init() {
	unsafeBasenameCharsRegexp = regexp.MustCompile("[^-.0-9a-zA-Z]")
}

type Options struct {
	OutputPath       string
	FilenameTemplate *template.Template
	OnStartFile      func(filename string)
}

func New(options Options) (*SaveResources, error) {
	options.FilenameTemplate.Option("missingkey=error")

	s := &SaveResources{
		outputPath:       options.OutputPath,
		filenameTemplate: options.FilenameTemplate,
		onStartFile:      options.OnStartFile,
	}

	if err := s.testFilenameTemplate(); err != nil {
		return nil, err
	}

	return s, nil
}

type SaveResources struct {
	outputPath       string
	filenameTemplate *template.Template
	onStartFile      func(filename string)
}

func (s *SaveResources) Save(resource map[interface{}]interface{}) error {
	filename, err := s.getFilenameForResource(resource)
	if err != nil {
		return err
	}

	if s.onStartFile != nil {
		s.onStartFile(filename)
	}

	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := yaml.NewEncoder(f)
	defer encoder.Close()

	if err := encoder.Encode(resource); err != nil {
		return err
	}

	return nil
}

func (s *SaveResources) testFilenameTemplate() error {
	_, err := s.getFilename("v1", "Test", "namespace", "name")
	return err
}

func (s *SaveResources) getFilename(apiVersion, kind, namespace, name string) (string, error) {
	if namespace == "" {
		namespace = defaultNamespace
	}
	filename := &bytes.Buffer{}
	err := s.filenameTemplate.Execute(filename, map[string]interface{}{
		"apiVersion": sanitiseBasename(apiVersion),
		"kind":       sanitiseBasename(kind),
		"namespace":  sanitiseBasename(namespace),
		"name":       sanitiseBasename(name),
	})
	if err != nil {
		return "", err
	}
	return filepath.Join(s.outputPath, filename.String()), nil
}

func (s *SaveResources) getFilenameForResource(resource map[interface{}]interface{}) (string, error) {
	apiVersion, ok := resource["apiVersion"].(string)
	if !ok {
		return "", errors.New("apiVersion is missing or not a string")
	}

	kind, ok := resource["kind"].(string)
	if !ok {
		return "", errors.New("kind is missing or not a string")
	}

	metadata, ok := resource["metadata"].(map[interface{}]interface{})
	if !ok {
		return "", errors.New("metadata is missing or not a map")
	}

	namespace, ok := metadata["namespace"].(string)
	if !ok {
		if _, ok := metadata["namespace"]; ok {
			return "", errors.New("namespace is not a string")
		}
	}

	name, ok := metadata["name"].(string)
	if !ok {
		return "", errors.New("name is missing or not a string")
	}

	return s.getFilename(apiVersion, kind, namespace, name)
}

func sanitiseBasename(s string) string {
	return unsafeBasenameCharsRegexp.ReplaceAllString(s, "_")
}
