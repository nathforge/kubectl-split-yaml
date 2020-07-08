package walkresources

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gopkg.in/yaml.v2"
)

type expectedCall = map[interface{}]interface{}

func TestData(t *testing.T) {
	dataPath := "testdata"

	fileInfos, err := ioutil.ReadDir(dataPath)
	if err != nil {
		t.Error(err)
	}

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			t.Run(fileInfo.Name(), func(t *testing.T) {
				if err := testDataFolder(filepath.Join(dataPath, fileInfo.Name())); err != nil {
					t.Error(err)
				}
			})
		}
	}
}

func testDataFolder(path string) error {
	decodeFile := func(filename string, dest interface{}) error {
		f, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer f.Close()

		decoder := yaml.NewDecoder(f)
		return decoder.Decode(dest)
	}

	inputFilename := filepath.Join(path, "input.yaml")
	input, err := os.Open(inputFilename)
	if err != nil {
		return err
	}
	defer input.Close()

	expectedCalls := []expectedCall{}
	for i := 0; ; i++ {
		outputFilename := filepath.Join(path, fmt.Sprintf("output%d.yaml", i))
		if _, err := os.Stat(outputFilename); err != nil {
			if os.IsNotExist(err) {
				break
			}
			return err
		}
		output := map[interface{}]interface{}{}
		if err := decodeFile(outputFilename, output); err != nil {
			return fmt.Errorf("%s: %w", outputFilename, err)
		}
		expectedCalls = append(expectedCalls, expectedCall(output))
	}

	return expectWalkCallbacks(input, expectedCalls)
}

func expectWalkCallbacks(input io.Reader, expectedCalls []expectedCall) error {
	nextCallIndex := 0

	err := WalkReader(input, func(actual map[interface{}]interface{}) error {
		callIndex := nextCallIndex

		if callIndex >= len(expectedCalls) {
			return errors.New("too many calls")
		}

		expected := expectedCalls[callIndex]
		nextCallIndex++

		if !reflect.DeepEqual(actual, expected) {
			return fmt.Errorf(
				"mismatch for call %[1]d\n"+
					"Actual:   (%[2]T) %#[2]v\n"+
					"Expected: (%[3]T) %#[3]v",
				callIndex, actual, expected,
			)
		}
		return nil
	})
	if err != nil {
		return err
	}

	remainingCalls := len(expectedCalls) - nextCallIndex
	if remainingCalls > 0 {
		return fmt.Errorf("expected %d more call(s)", remainingCalls)
	}

	return nil
}
