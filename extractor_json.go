package extractor

import (
	encodingJson "encoding/json"
	"github.com/bitly/go-simplejson"
	"github.com/elgs/jsonql"
	"github.com/xlvector/dlog"
	"strconv"
	"strings"
)

func isJsonArray(json *simplejson.Json) (int, bool) {
	arr, err := json.Array()
	if err == nil {
		return len(arr), true
	}
	return 0, false
}

func GetJsonPath(jsonKey string, json *simplejson.Json) *simplejson.Json {
	path := strings.Split(jsonKey, ".")

	for x := 0; x < len(path); x++ {
		if json == nil {
			break
		}
		cmd := path[x]
		if strings.HasPrefix(cmd, "[") && strings.HasSuffix(cmd, "]") {
			indexStr := FindGroupByIndex("\\d+", cmd, 0)
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				dlog.Warn("convet int error %v", err)
				json = nil
			}
			temp := json.GetIndex(index)
			if temp != nil {
				json = temp
			} else {
				dlog.Warn("json array not found %d", index)
				json = nil
			}
		} else if strings.HasPrefix(cmd, "(") && strings.HasSuffix(cmd, ")") {
			query := cmd[1 : len(cmd)-1]
			parser := jsonql.NewQuery(json.Interface())
			m, err := parser.Query(query)
			if err != nil {
				dlog.Warn("not found %s", err.Error())
				return json
			}
			v, err := encodingJson.Marshal(m)
			if err != nil {
				dlog.Warn("Marshal json %s", err.Error())
				return json
			}
			newjson, err := simplejson.NewJson(v)
			if err != nil {
				dlog.Warn("NewJson %s", err.Error())
				return json
			}
			json = newjson
		} else {
			temp, exist := json.CheckGet(cmd)
			if !exist {
				dlog.Warn("jsonKey :%s not exist in %v", cmd, json)
				json = nil
			} else {
				json = temp
			}
		}
	}
	return json
}

func (self *Extractor) extractJson(config interface{}, json *simplejson.Json) interface{} {

	if v, ok := config.(string); ok {
		if self.Filter != nil {
			if val, isFilter := self.Filter(v); isFilter {
				return val
			}
		}
		fillValue, isFill := self.fillExpression(v)
		if isFill {
			v = fillValue
		}
		val := self.ExtractJsonSingle(v, json)
		if val == "" {
			val = nil
		}
		return val
	}

	if m, ok := config.(map[string]interface{}); ok {
		doc := json
		rt := self.root(m)
		if len(rt) > 0 && rt != "json" {
			doc = GetJsonPath(rt, doc)
			if doc == nil {
				return nil
			}
		} else if len(rt) > 0 && rt == "json" {
			val, err := doc.String()
			if len(val) > 0 {
				doc, err = simplejson.NewFromReader(strings.NewReader(val))
				if err != nil {
					dlog.Warn("convert to string error:%s value:%s", err.Error(), val)
					return nil
				}
			}
		}
		length, yes := isJsonArray(doc)
		if yes == false {
			ret := make(map[string]interface{})
			for key, val := range m {
				if strings.HasPrefix(key, "_") {
					continue
				}
				ret[key] = self.extractJson(val, doc)
			}
			return ret
		} else {
			ret := []map[string]interface{}{}
			for i := 0; i < length; i++ {
				sub := make(map[string]interface{})
				for key, val := range m {
					if strings.HasPrefix(key, "_") {
						continue
					}
					stmp := doc.GetIndex(i)
					sub[key] = self.extractJson(val, stmp)
				}
				ret = append(ret, sub)
			}
			return ret
		}
	}
	return nil
}

func (self *Extractor) ExtractJsonSingle(v string, json *simplejson.Json) interface{} {
	if v == "@data" {
		return json.Interface()
	} else if strings.Contains(v, "@key:") {
		return strings.Replace(v, "@key:", "", 1)
	}
	var ret interface{}
	sel := NewJsonSelector(v, self)

	if len(sel.JsonKey) > 0 {
		b := GetJsonPath(sel.JsonKey, json)
		if b != nil {

			if str, err := b.String(); err == nil {
				ret = str
			} else if d, err := b.Int64(); err == nil {
				ret = strconv.FormatInt(d, 10)
			} else if d, err := b.Float64(); err == nil {
				ret = strconv.FormatFloat(d, 'g', 5, 64)
			} else if boolean, err := b.Bool(); err == nil {
				ret = strconv.FormatBool(boolean)
			} else if arr, err := b.Array(); err == nil {
				ret = arr
			} else if m, err := b.Map(); err == nil {
				ret = m
			} else {
				ret = b.Interface()
			}
		}
		if ret == nil {
			dlog.Warn("path:%s not found value", v)
			return nil
		}
	}

	if ret != nil && len(sel.Default) > 0 {
		ret = sel.Default
	}

	if ret != nil && sel.DateFormat != nil {
		ret = sel.DateFormat.Format(ret.(string))
	}
	return ret
}