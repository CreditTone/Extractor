package extractor

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/bitly/go-simplejson"
	"github.com/xlvector/dlog"
)

const (
	SET_DEFINE      = "_v"
	TYPE_DEFINE     = "_type"
	JSONTYPE_DEFINE = "_jsontype"
	ROOT_DEFINE     = "_root"
	ERROR_DEFINE    = "_error"
	SOURCE_DEFINE   = "_source"
)

type Extractor struct {
	Filter     func(config string) (string, bool)
	DoTemplate func(template, v string) string
}

func NewExtractor() *Extractor {
	instance := Extractor{}
	return &instance
}

func (self *Extractor) root(config map[string]interface{}) string {
	xpath, ok := config[ROOT_DEFINE]
	if !ok {
		return ""
	}
	v, ok := self.Filter(xpath.(string))
	if ok || len(v) > 0 {
		xpath = v
	}
	return xpath.(string)
}

func (self *Extractor) dataType(config map[string]interface{}) string {
	dataType, ok := config[TYPE_DEFINE]
	if !ok {
		dataType, ok = config[JSONTYPE_DEFINE]
	}
	if !ok {
		return "html"
	}
	return dataType.(string)
}

func (self *Extractor) errorDetector(config map[string]interface{}) string {
	xpath, ok := config["_error"]
	if !ok {
		return ""
	}
	v, ok := self.Filter(xpath.(string))
	if ok {
		xpath = v
	}
	return xpath.(string)
}

func (self *Extractor) source(config map[string]interface{}) ([]byte, bool) {
	source, ok := config[SOURCE_DEFINE]
	if !ok {
		return nil, false
	}
	val, ok := self.Filter(source.(string))
	if ok {
		body := []byte(val)
		return body, ok
	}
	return nil, false
}

func (self *Extractor) Do(config interface{}, body []byte) interface{} {
	var ret interface{}
	if m, ok := config.(map[string]interface{}); ok {
		val, ok := self.source(m)
		if ok {
			body = val
		}
		dataType := self.dataType(m)
		if dataType == "json" {
			jsonBody := FilterJSONP(string(body))
			json, err := simplejson.NewFromReader(strings.NewReader(jsonBody))
			if err != nil {
				dlog.Warn("%s: %s", jsonBody, err.Error())
				return nil
			}
			ret = self.extractJson(m, json)
		} else if dataType == "html" {
			doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
			if err != nil {
				dlog.Warn("%s", err.Error())
				return nil
			}
			ret = self.extract(config, doc.First())
		} else if dataType == "xml" {
		} else if dataType == "string" {
			ret = self.extractString(config, string(body))
		}
	}
	return ret
}

func (self *Extractor) extract(config interface{}, s *goquery.Selection) interface{} {
	if v, ok := config.(string); ok {
		filterValue, isFilter := self.Filter(v)
		if isFilter {
			return filterValue
		} else if len(filterValue) > 0 {
			v = filterValue
		}
		val := self.extractSingle(v, s)
		if val == "" {
			val = nil
		}
		return val
	}

	if m, ok := config.(map[string]interface{}); ok {
		doc := s
		isArray := false

		ed := self.errorDetector(m)
		if len(ed) > 0 {
			edoc := queryXpath(ed, s)
			if edoc == nil || edoc.Size() == 0 {
				return map[string]string{
					"error": "页面错误",
				}
			}
		}

		rt := self.root(m)
		if len(rt) > 0 {
			isArray = strings.Contains(rt, "@array")
			rt = strings.Replace(rt, "@array", "", 1)
			doc = queryXpath(rt, s)
		}
		if doc == nil || doc.Size() == 0 {
			if isArray {
				return []map[string]interface{}{}
			}
			return nil
		} else if isArray || doc.Size() > 1 {
			ret := []map[string]interface{}{}
			doc.Each(func(i int, stmp *goquery.Selection) {
				sub := self.extractContainKey(m, stmp)
				ret = append(ret, sub)
			})
			return ret
		} else if doc.Size() == 1 {
			return self.extractContainKey(m, doc)
		}
	}
	return nil
}

func (self *Extractor) extractContainKey(m map[string]interface{}, s *goquery.Selection) map[string]interface{} {
	ret := make(map[string]interface{})
	for key, val := range m {
		if strings.HasPrefix(key, "_") {
			continue
		}
		if strings.HasPrefix(key, "@key") {
			keyXpath := strings.Replace(key, "@key", "", -1)
			keyResult := self.extract(keyXpath, s)
			if keyResult != nil {
				key = keyResult.(string)
			} else {
				continue
			}
		}

		if strings.HasPrefix(key, "@dupkey") {
			tks := strings.SplitN(key, " ", 2)
			key = tks[len(tks)-1]
			if tmp, ok := ret[key]; !ok || tmp == nil {
				ret[key] = self.extract(val, s)
			} else {
				continue
			}
		}
		ret[key] = self.extract(val, s)
	}
	return ret
}

type HtmlSelector struct {
	Xpath    string
	Attr     string
	Regex    string
	Template string
}

func NewHtmlSelector(v string) *HtmlSelector {
	ret := &HtmlSelector{}
	if strings.Contains(v, ">|") {
		tks := strings.Split(v, ">|")
		v = tks[0]
		ret.Template = tks[1]
	}
	tks := strings.Split(v, ";")
	ret.Xpath = tks[0]
	if len(tks) > 1 {
		ret.Attr = tks[1]
	}
	if len(tks) > 2 {
		ret.Regex = tks[2]
	}
	return ret
}

func Regex(regex, buf string) (string, []string) {
	if len(regex) > 0 && strings.Contains(regex, "@multi ") {
		reg := strings.Replace(regex, "@multi ", "", 1)
		return "", FindGroupsByIndex(reg, buf, 1)
	} else if len(regex) > 0 {
		reg := regexp.MustCompile(regex)
		result := reg.FindAllStringSubmatch(buf, 1)
		if len(result) > 0 {
			group := result[0]
			if len(group) > 1 {
				buf = group[1]
			} else {
				buf = group[0]
			}
		} else {
			dlog.Warn("regex not found value %s", regex)
			buf = ""
		}
	}
	return buf, nil
}

/*
func (self *HtmlSelector) convertType(content string) interface{} {
	var ret interface{}
	var err error
	if self.Type == "" {
		return content
	} else if self.Type == "int" {
		ret, err = strconv.Atoi(content)
	} else if self.Type == "float" {
		ret, err = strconv.ParseFloat(content, 0)
	} else if self.Type == "bool" {
		ret, err = strconv.ParseBool(content)
	} else if self.Type == "json" {
		err = json.Unmarshal([]byte(content), &ret)
	}
	if err == nil {
		return ret
	} else {
		dlog.Warn("convet to %s err %s value:%s", self.Type, err.Error(), content)
		return nil
	}
}
*/

func (self *Extractor) extractSingle(v string, s *goquery.Selection) interface{} {
	if v == "@data" {
		data, err := s.Html()
		if err != nil {
			dlog.Warn("MarshalHTML %s", err.Error())
			return ""
		}
		return data
	}
	sel := NewHtmlSelector(v)
	b := s
	if len(sel.Xpath) > 0 {
		b = queryXpath(sel.Xpath, s)
	}
	var text string
	if len(sel.Attr) > 0 {
		if sel.Attr == "html" {
			text, _ = b.First().Html()
		} else {
			text, _ = b.First().Attr(sel.Attr)
			if (sel.Attr == "href" || sel.Attr == "src") && strings.HasPrefix(text, "//") {
				text = "https:" + text
			}
			text = strings.TrimSpace(text)
		}
	} else {
		text = strings.TrimSpace(b.First().Text())
	}

	text, ret := Regex(sel.Regex, text)
	if ret != nil {
		return ret
	}
	if len(sel.Template) > 0 {
		text = self.DoTemplate(sel.Template, text)
	}
	return text
}

func queryXpath(xpath string, s *goquery.Selection) *goquery.Selection {
	defer func() {
		if err := recover(); err != nil {
			dlog.Warn("queryXpath Error:%v", err)
		}
	}()

	var b *goquery.Selection
	index, ok := FindIntAttribute("index", xpath)
	parent, ok2 := FindIntAttribute("parent", xpath)
	if ok {
		xpath = CleanAttribute("index", xpath)
		b = s.Find(xpath)
		b = b.Eq(index)
	} else if ok2 {
		xpath = CleanAttribute("parent", xpath)
		b = s.Find(xpath)
		if b.Size() > 1 {
			b = b.First()
		}
		for x := 0; x < parent; x++ {
			b = b.Parent()
		}
	} else if strings.Contains(xpath, "@last") {
		xpath = strings.Replace(xpath, "@last", "", 1)
		b = s.Find(xpath)
		b = b.Last()
	} else {
		b = s.Find(xpath)
	}
	return b
}
