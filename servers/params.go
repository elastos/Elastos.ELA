package servers

import (
	"strconv"

	"github.com/elastos/Elastos.ELA/common/log"
)

// MaxParamSize is the maximum parameter size.
const MaxParamSize = 1024 * 100

type Params map[string]interface{}

func FromArray(array []interface{}, fields ...string) Params {
	params := make(Params)
	for i := 0; i < len(array); i++ {
		params[fields[i]] = array[i]
	}
	return params
}

func (p Params) Int(field string) (int64, bool) {
	value, ok := p[field]
	if !ok {
		return 0, false
	}
	switch v := value.(type) {
	case float64:
		return int64(v), true
	case string:
		int, err := strconv.ParseInt(p[field].(string), 10, 64)
		if err != nil {
			return 0, false
		}
		return int, true
	default:
		return 0, false
	}
}

func (p Params) Uint(field string) (uint32, bool) {
	value, ok := p[field]
	if !ok {
		return 0, false
	}
	switch v := value.(type) {
	case float64:
		if v < 0 {
			return 0, false
		}
		return uint32(v), true
	case string:
		uint, err := strconv.ParseUint(p[field].(string), 10, 64)
		if err != nil {
			return 0, false
		}
		return uint32(uint), true
	default:
		return 0, false
	}
}

func (p Params) Float(field string) (float64, bool) {
	value, ok := p[field]
	if !ok {
		return 0, false
	}
	switch v := value.(type) {
	case float64:
		return v, true
	case string:
		float, err := strconv.ParseFloat(p[field].(string), 64)
		if err != nil {
			return 0, false
		}

		return float, true
	default:
		return 0, false
	}
}

func (p Params) Bool(key string) (bool, bool) {
	value, ok := p[key]
	if !ok {
		return false, false
	}
	switch v := value.(type) {
	case bool:
		return v, true
	default:
		return false, false
	}
}

func (p Params) String(key string, maxStringLen int) (string, bool) {
	value, ok := p[key]
	if !ok {
		return "", false
	}
	v, ok := value.(string)
	if !ok {
		log.Warn("param", v, " is not a string")
		return "", false
	}
	if len(v) > maxStringLen || len(v) > MaxParamSize {
		log.Warn("param", v, " is larger than the max allowed size")
		return "", false
	}
	return v, true
}

func (p Params) ArrayString(key string, maxStringLen int) ([]string, bool) {
	value, ok := p[key]
	if !ok {
		return nil, false
	}
	switch v := value.(type) {
	case []interface{}:
		var arrayString []string
		for _, param := range v {
			paramString, ok := param.(string)
			if !ok {
				log.Warn("param", param, " is not a string")
				return nil, false
			}
			if len(paramString) > maxStringLen || len(paramString) > MaxParamSize {
				log.Warn("param", v, " is larger than the max allowed size")
				return nil, false
			}
			arrayString = append(arrayString, paramString)
		}
		return arrayString, true

	default:
		return nil, false
	}
}
