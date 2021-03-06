package extractor

import (
	encodingJson "encoding/json"
	"strconv"
	"strings"

	"github.com/bitly/go-simplejson"
	"github.com/elgs/jsonql"
	"github.com/xlvector/dlog"
)

func isJsonArray(json *simplejson.Json) (int, bool) {
	arr, err := json.Array()
	if err == nil {
		return len(arr), true
	}
	return 0, false
}

func NewSimplejson(data interface{}) *simplejson.Json {
	item := &simplejson.Json{}
	item.SetPath([]string{}, data)
	return item
}

func GetJsonArray(json *simplejson.Json) []*simplejson.Json {
	a, err := json.Array()
	if err == nil {
		jsonArr := []*simplejson.Json{}
		for index, _ := range a {
			jsonArr = append(jsonArr, NewSimplejson(a[index]))
		}
		return jsonArr
	}
	return nil
}

func getJsonPath(jsonpath []string, json *simplejson.Json) *simplejson.Json {
	for x := 0; x < len(jsonpath); x++ {
		if json == nil {
			break
		}
		cmd := jsonpath[x]
		if cmd == "[*]" {
			if jsonArr := GetJsonArray(json); jsonArr != nil {
				ret := []interface{}{}
				for _, item := range jsonArr {
					itemResult := getJsonPath(jsonpath[x+1:len(jsonpath)], item)
					if itemResult != nil {
						arr, err := itemResult.Array()
						if err != nil {
							ret = append(ret, itemResult.Interface())
						} else {
							for _, val := range arr {
								ret = append(ret, val)
							}
						}
					}
				}
				return NewSimplejson(ret)
			}
			return nil
		} else if strings.HasPrefix(cmd, "[") && strings.HasSuffix(cmd, "]") {
			indexStr := cmd[1 : len(cmd)-1]
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
				dlog.Warn("jsonKey :%s not exist in the %s", cmd, description(json))
				json = nil
			} else {
				json = temp
			}
		}
	}
	return json
}

func description(json *simplejson.Json) string {
	jsonEncode, _ := json.Encode()
	if jsonEncode != nil {
		jsonStr := string(jsonEncode)
		if len(jsonStr) > 100 {
			jsonStr = jsonStr[0:100] + "......"
		}
		return jsonStr
	}
	return ""
}

func GetJsonPath(jsonKey string, json *simplejson.Json) *simplejson.Json {
	path := strings.Split(jsonKey, ".")
	return getJsonPath(path, json)
}

func (self *Extractor) extractJson(config interface{}, json *simplejson.Json) interface{} {

	if v, ok := config.(string); ok {
		if self.Filter != nil {
			if val, isFilter := self.Filter(v); isFilter {
				return val
			} else if len(val) > 0 {
				v = val
			}
		}
		val := self.ExtractJsonSingle(v, json)
		if val == "" {
			val = nil
		}
		return val
	}

	if m, ok := config.(map[string]interface{}); ok {
		doc := json
		dataType := self.dataType(m)
		if dataType == "jsonstring" {
			doc = UnMarshal(doc)
		}
		rt := self.root(m)
		if len(rt) > 0 {
			doc = GetJsonPath(rt, doc)
			if doc == nil {
				return nil
			}
		}
		if isEmptyParse(m) {
			return doc.Interface()
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

func UnMarshal(json *simplejson.Json) *simplejson.Json {
	val, err := json.String()
	if len(val) > 0 {
		json, err = simplejson.NewFromReader(strings.NewReader(val))
		if err != nil {
			dlog.Warn("convert to string error:%s value:%s", err.Error(), val)
			return nil
		}
	}
	return json
}

func isEmptyParse(config map[string]interface{}) bool {
	for k, _ := range config {
		if !strings.HasPrefix(k, "_") {
			return false
		}
	}
	return true
}

type JsonSelector struct {
	JsonKey   string
	Template  string
	UnMarshal bool
	Regex     string
}

func NewJsonSelector(v string) *JsonSelector {
	ret := &JsonSelector{}
	if strings.Contains(v, ">|") {
		tks := strings.Split(v, ">|")
		v = tks[0]
		ret.Template = tks[1]
	}
	tks := strings.Split(v, ";")
	ret.JsonKey = tks[0]
	if len(tks) > 1 && tks[1] == "true" {
		ret.UnMarshal = true
	}
	if len(tks) > 2 {
		ret.Regex = tks[2]
	}
	return ret
}

func (self *Extractor) ExtractJsonSingle(v string, json *simplejson.Json) interface{} {
	if v == "@data" {
		return json.Interface()
	}
	var ret interface{}
	sel := NewJsonSelector(v)

	if len(sel.JsonKey) > 0 {
		b := GetJsonPath(sel.JsonKey, json)
		if sel.UnMarshal && b != nil {
			b = UnMarshal(b)
		}
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
			return ""
		}

		if len(sel.Template) > 0 {
			ret = self.DoTemplate(sel.Template, ret.(string))
		}
	}

	if len(sel.Regex) > 0 {
		ret, _ = Regex(sel.Regex, ret.(string))
	}

	return ret
}
