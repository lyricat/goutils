package structs

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"sort"
)

type (
	JSONList []interface{}

	JSONMap map[string]interface{}
)

func (a JSONList) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *JSONList) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a)
}

func (a *JSONList) LoadFromStringArray(arr []string) {
	for _, item := range arr {
		*a = append(*a, item)
	}
}

func (a *JSONList) ToStringArray() []string {
	var arr []string
	for _, item := range *a {
		if str, ok := item.(string); ok {
			arr = append(arr, str)
		}
	}
	return arr
}

func (a *JSONList) ToInt64Array() []int64 {
	var arr []int64
	for _, item := range *a {
		if val, ok := item.(float64); ok {
			arr = append(arr, int64(val))
		}
	}
	return arr
}

func (a *JSONList) ToUint64Array() []uint64 {
	var arr []uint64
	for _, item := range *a {
		if val, ok := item.(float64); ok {
			arr = append(arr, uint64(val))
		}
	}
	return arr
}

func (a *JSONList) Contains(item interface{}) bool {
	for _, _item := range *a {
		if item == _item {
			return true
		}
	}
	return false
}

func sortMapKeys(data any) any {
	switch data := data.(type) {
	case map[string]any:
		keys := make([]string, 0, len(data))
		for key := range data {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		result := make(map[string]interface{}, len(data))
		for _, key := range keys {
			result[key] = sortMapKeys(data[key])
		}
		return result
	default:
		return data
	}
}
