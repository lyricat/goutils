package structs

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
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

func (a *JSONMap) GetMap(key string) JSONMap {
	if val, ok := (*a)[key]; ok {
		if mapVal, ok := val.(JSONMap); ok {
			return mapVal
		}
	}
	return NewJSONMap()
}

func (a *JSONMap) GetFloat64(key string) float64 {
	if val, ok := (*a)[key]; ok {
		if floatVal, ok := val.(float64); ok {
			return floatVal
		}
	}
	return 0
}

func (a *JSONMap) SetValue(key string, value interface{}) {
	(*a)[key] = value
}
