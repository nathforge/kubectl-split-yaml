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

func Walk(obj interface{}, callback callback) error {
	return walkObj(obj, callback)
}

func walkObj(obj interface{}, callback callback) error {
	if objMap, ok := obj.(map[interface{}]interface{}); ok {
		return walkObjMap(objMap, callback)
	}
	return errUnexpectedType(obj)
}

func walkObjMap(objMap map[interface{}]interface{}, callback callback) error {
	if apiVersion, kind, ok := getResourceType(objMap); ok {
		if apiVersion == "v1" && kind == "List" {
			return walkList(objMap, callback)
		}
	} else {
		return ErrNotAResource
	}
	return callback(objMap)
}

func walkList(obj map[interface{}]interface{}, callback callback) error {
	if items, ok := obj["items"].([]interface{}); ok {
		for _, itemIntf := range items {
			if item, ok := itemIntf.(map[interface{}]interface{}); ok {
				if err := walkKetallItem(item, callback); err != nil {
					if !errors.Is(err, ErrNotAKetallItem) {
						return err
					}
					if err := walkObj(item, callback); err != nil {
						return err
					}
				}
			} else {
				return errUnexpectedType(item)
			}
		}
		return nil
	}
	return ErrNotAListResource
}

func walkKetallItem(obj map[interface{}]interface{}, callback callback) error {
	if _, ok := obj["apiVersion"].(string); !ok {
		if _, ok := obj["kind"].(string); !ok {
			if itemIntfs, ok := obj["items"].([]interface{}); ok {
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
		}
	}
	return ErrNotAKetallItem
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
