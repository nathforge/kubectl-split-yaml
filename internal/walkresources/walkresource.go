package walkresources

import (
	"errors"
	"fmt"
)

var ErrNotAResource = errors.New("object is not a resource (missing apiVersion or kind)")
var ErrNotAListResource = errors.New("object is not a list resource")
var ErrNotAKetallItem = errors.New("not a ketall item")
var ErrUnexpectedType = errors.New("unexpected type")

type callback = func(map[interface{}]interface{}) error

// Walk takes a parsed YAML object and calls callback() for each Kubernetes
// resource
func Walk(obj interface{}, callback callback) error {
	return walkObj(obj, callback)
}

// walkObj dispatches based on obj type
func walkObj(obj interface{}, callback callback) error {
	if objMap, ok := obj.(map[interface{}]interface{}); ok {
		return walkObjMap(objMap, callback)
	}
	return errUnexpectedType(obj)
}

// walkObjMap dispatches to walkList() if it looks like a v1/List object, or
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
			return errUnexpectedType(item)
		}

		// Try to handle ketall plugin output
		if err := walkKetallItem(item, callback); err != nil {
			if !errors.Is(err, ErrNotAKetallItem) {
				return err
			}
			// This isn't a ketall item - assume it's a normal resource
			if err := walkObj(item, callback); err != nil {
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
	// top-level item. Returns ErrNotAKetallItem if unsuccessful, signalling
	// the item should be processed as a standard Kubernetes resource.

	if _, ok := obj["apiVersion"].(string); ok {
		return ErrNotAKetallItem
	}

	if _, ok := obj["kind"].(string); ok {
		return ErrNotAKetallItem
	}

	itemIntfs, ok := obj["items"].([]interface{})
	if !ok {
		return ErrNotAKetallItem
	}

	for _, itemIntf := range itemIntfs {
		if item, ok := itemIntf.(map[interface{}]interface{}); ok {
			if err := callback(item); err != nil {
				return err
			}
		} else {
			return errUnexpectedType(itemIntf)
		}
	}
	return nil
}

func errUnexpectedType(value interface{}) error {
	return fmt.Errorf("%w: %T", ErrUnexpectedType, value)
}

func getResourceType(obj map[interface{}]interface{}) (string, string, bool) {
	if apiVersion, ok := obj["apiVersion"].(string); ok {
		if kind, ok := obj["kind"].(string); ok {
			return apiVersion, kind, true
		}
	}
	return "", "", false
}
