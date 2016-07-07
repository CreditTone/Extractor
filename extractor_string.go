package extractor

import (
	"strings"
)

func (self *Extractor) extractString(config interface{}, body string) interface{} {
	if v, ok := config.(string); ok {
		if self.Filter != nil {
			if val, isFilter := self.Filter(v); isFilter {
				return val
			}
		}
		val, array := Regex(v, body)
		if array != nil {
			return array
		}
		if len(val) == 0 {
			return nil
		}
		return val
	}

	if m, ok := config.(map[string]interface{}); ok {
		ret := make(map[string]interface{})
		for key, val := range m {
			if strings.HasPrefix(key, "_") {
				continue
			}
			ret[key] = self.extractString(val, body)
		}
		return ret
	}
	return nil
}
