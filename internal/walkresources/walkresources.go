package walkresources

import (
	"errors"
	"fmt"
	"io"

	"gopkg.in/yaml.v2"
)

var ErrNotAResource = errors.New("object is not a resource (missing apiVersion or kind)")
var ErrNotAListResource = errors.New("object is not a list resource")
var ErrUnexpectedType = errors.New("unexpected type")

var errNotAKetallItem = errors.New("not a ketall item")

type callback = func(map[interface{}]interface{}) error

// WalkReader takes YAML docs from a Reader and calls callback() for each
// Kubernetes resource
func WalkReader(reader io.Reader, callback callback) error {
	decoder := yaml.NewDecoder(reader)
	for {
		doc := map[interface{}]interface{}{}
		if err := decoder.Decode(doc); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}

		if err := WalkObj(doc, callback); err != nil {
			return err
		}
	}

	return nil
}

// WalkObj takes a parsed YAML object and calls callback() for each Kubernetes
// resource
func WalkObj(obj interface{}, callback callback) error {
	if objMap, ok := obj.(map[interface{}]interface{}); ok {
		return walkObjMap(objMap, callback)
	}

	return newErrUnexpectedType(obj)
}

// walkObjMap delegates to walkList() if it looks like a v1/List object, or
// calls callback() if not
func walkObjMap(objMap map[interface{}]interface{}, callback callback) error {
	apiVersion, kind, ok := getResourceType(objMap)
	if !ok {
		return ErrNotAResource
	}

	if apiVersion == "v1" && kind == "List" {
		return walkList(objMap, callback)
	}

	return callback(objMap)
}

// walkList calls callback() for each resource in the list.
// Has special handling for the ketall kubectl plugin
func walkList(obj map[interface{}]interface{}, callback callback) error {
	items, ok := obj["items"].([]interface{})
	if !ok {
		return ErrNotAListResource
	}

	for _, itemIntf := range items {
		item, ok := itemIntf.(map[interface{}]interface{})
		if !ok {
			return newErrUnexpectedType(item)
		}

		// Try to handle ketall plugin output
		if err := walkKetallItem(item, callback); err != nil {
			if errors.Is(err, errNotAKetallItem) {
				// This isn't a ketall item. Treat as a standard resource
				if err := WalkObj(item, callback); err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}
	return nil
}

// walkKetallItem attempts to handle an item output from the ketall kubectl
// plugin <https://github.com/corneliusweig/ketall>
func walkKetallItem(obj map[interface{}]interface{}, callback callback) error {
	// ketall plugin output is inconsistent with kubectl v1/List objects.
	//
	// kubectl would represent two items as:
	//     apiVersion: v1
	//     kind: List
	//     items:
	//     - AAA
	//     - BBB
	//
	// ketall represents the same two items as:
	//     apiVersion: v1
	//     kind: List
	//     items:
	//     - items:
	//       - AAA
	//       - BBB
	//
	// Notice the extra items objects with no apiVersion or kind.
	//
	// This function tries to callback ketall resources found within a
	// top-level item. Returns errNotAKetallItem if unsuccessful, signalling
	// the item should be processed as a standard Kubernetes resource.

	if _, ok := obj["apiVersion"].(string); ok {
		return errNotAKetallItem
	}

	if _, ok := obj["kind"].(string); ok {
		return errNotAKetallItem
	}

	itemIntfs, ok := obj["items"].([]interface{})
	if !ok {
		return errNotAKetallItem
	}

	for _, itemIntf := range itemIntfs {
		item, ok := itemIntf.(map[interface{}]interface{})
		if !ok {
			return newErrUnexpectedType(itemIntf)
		}

		if err := callback(item); err != nil {
			return err
		}
	}
	return nil
}

func newErrUnexpectedType(value interface{}) error {
	return fmt.Errorf("%w: %T", ErrUnexpectedType, value)
}

func getResourceType(obj map[interface{}]interface{}) (string, string, bool) {
	apiVersion, ok := obj["apiVersion"].(string)
	if !ok {
		return "", "", false
	}

	kind, ok := obj["kind"].(string)
	if !ok {
		return "", "", false
	}

	return apiVersion, kind, true
}
