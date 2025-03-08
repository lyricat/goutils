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

func NewJSONMap() JSONMap {
	return make(JSONMap)
}

func NewFromMap(m map[string]interface{}) JSONMap {
	a := NewJSONMap()
	for k, v := range m {
		a[k] = v
	}
	return a
}

func NewFromJSONString(jsonString string) JSONMap {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(jsonString), &m); err != nil {
		return nil
	}
	return NewFromMap(m)
}

func (a JSONMap) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *JSONMap) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a)
}

func (a *JSONMap) HasKey(key string) bool {
	_, ok := (*a)[key]
	return ok
}

func (a *JSONMap) GetString(key string) string {
	if val, ok := (*a)[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return ""
}

func (a *JSONMap) GetInt64(key string) int64 {
	if val, ok := (*a)[key]; ok {
		if intVal, ok := val.(int64); ok {
			return intVal
		} else if intVal, ok := val.(float64); ok {
			return int64(intVal)
		}
	}
	return 0
}

func (a *JSONMap) GetBool(key string) bool {
	if val, ok := (*a)[key]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return false
}

func (a *JSONMap) GetMap(key string) *JSONMap {
	ret := NewJSONMap()
	if val, ok := (*a)[key]; ok {
		if mapVal, ok := val.(JSONMap); ok {
			ret = mapVal
		}
		if mapVal, ok := val.(map[string]interface{}); ok {
			ret = NewFromMap(mapVal)
		}
	}
	return &ret
}

func (a *JSONMap) GetFloat64(key string) float64 {
	if val, ok := (*a)[key]; ok {
		if floatVal, ok := val.(float64); ok {
			return floatVal
		}
	}
	return 0
}

func (a *JSONMap) GetArray(key string) []any {
	if val, ok := (*a)[key]; ok {
		if arrVal, ok := val.([]any); ok {
			return arrVal
		}
	}
	return nil
}

func (a *JSONMap) SetValue(key string, value interface{}) {
	(*a)[key] = value
}

func (a *JSONMap) Delete(key string) {
	delete(*a, key)
}

func (a *JSONMap) Dump() string {
	json, err := json.Marshal(a)
	if err != nil {
		return ""
	}
	return string(json)
}

func (a *JSONMap) Size() int {
	return len(*a)
}

func (a *JSONMap) Split(size int) []JSONMap {
	// split the json map into size number of json maps
	if size <= 0 {
		// If size is invalid, return the original map as a single item
		return []JSONMap{*a}
	}

	totalEntries := a.Size()
	if totalEntries == 0 {
		// If the map is empty, return an empty slice
		return []JSONMap{}
	}

	// Calculate how many maps we need
	numMaps := (totalEntries + size - 1) / size // Ceiling division
	result := make([]JSONMap, 0, numMaps)

	currentMap := NewJSONMap()
	currentSize := 0

	// Distribute entries among the maps
	for key, value := range *a {
		currentMap[key] = value
		currentSize++

		// When we reach the size limit, add the current map to results and create a new one
		if currentSize >= size {
			result = append(result, currentMap)
			currentMap = NewJSONMap()
			currentSize = 0
		}
	}

	// Add the last map if it contains any entries
	if currentSize > 0 {
		result = append(result, currentMap)
	}

	return result
}

func (a *JSONMap) Merge(b JSONMap) {
	for k, v := range b {
		(*a)[k] = v
	}
}

func (a *JSONMap) SortByKey() {
	(*a) = sortMapKeys(*a).(JSONMap)
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
