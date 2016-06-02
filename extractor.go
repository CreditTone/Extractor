package extractor

import (
	"bytes"
	"encoding/json"
	"github.com/PuerkitoBio/goquery"
	"github.com/bitly/go-simplejson"
	"github.com/xlvector/dlog"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Extractor struct {
	Filter      func(config string) (interface{}, bool)
	Fill        func(config string) (string, bool)
	DateFormats map[string]*DateFormat
}

func NewExtractor(filter func(config string) (interface{}, bool), fill func(config string) (string, bool)) *Extractor {
	instance := Extractor{
		DateFormats: make(map[string]*DateFormat, 0),
	}
	instance.Filter = filter
	instance.Fill = fill
	return &instance
}

func (self *Extractor) AddDateFormat(key, input, output string) {
	self.DateFormats[key] = NewDateFormat(input, output)
}

func (self *Extractor) GetDateFormat(key string) *DateFormat {
	return self.DateFormats[key]
}

func (self *Extractor) root(config map[string]interface{}) string {
	xpath, ok := config["_root"]
	if !ok {
		return ""
	}
	fillValue, isFill := self.fillExpression(xpath.(string))
	if isFill {
		xpath = fillValue
	}
	return xpath.(string)
}

func (self *Extractor) errorDetector(config map[string]interface{}) string {
	xpath, ok := config["_error"]
	if !ok {
		return ""
	}
	fillValue, isFill := self.fillExpression(xpath.(string))
	if isFill {
		xpath = fillValue
	}
	return xpath.(string)
}

func (self *Extractor) source(config map[string]interface{}) string {
	source, ok := config["_source"]
	if !ok {
		return ""
	}
	return source.(string)
}

func (self *Extractor) Do(config interface{}, body []byte) map[string]interface{} {
	var ret interface{}
	if m, ok := config.(map[string]interface{}); ok {
		source := self.source(m)
		if len(source) > 0 {
			val, ok := self.Filter(source)
			if ok {
				body = []byte(val.(string))
			}
		}
		rt := self.root(m)
		if rt == "json" {
			jsonBody := FilterJSONP(string(body))
			json, err := simplejson.NewFromReader(strings.NewReader(jsonBody))
			if err != nil {
				dlog.Warn("%s: %s", jsonBody, err.Error())
				return nil
			}
			ret = self.extractJson(m, json)
		} else {
			doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
			if err != nil {
				dlog.Warn("%s", err.Error())
				return nil
			}
			ret = self.extract(config, doc.First())
		}
	}

	if ret == nil {
		return make(map[string]interface{}, 0)
	}
	mret, ok := ret.(map[string]interface{})
	if !ok {
		dlog.Warn("extractorConvertError:%v", ret)
		return nil
	}
	return mret
}

func (self *Extractor) filterExpression(v string) (interface{}, bool) {
	if self.Filter != nil {
		if val, isFilter := self.Filter(v); isFilter {
			return val, true
		}
	}
	return nil, false
}

func (self *Extractor) fillExpression(v string) (string, bool) {
	if self.Fill != nil {
		if val, isFill := self.Fill(v); isFill {
			return val, true
		}
	}
	return "", false
}

func (self *Extractor) extract(config interface{}, s *goquery.Selection) interface{} {
	if v, ok := config.(string); ok {
		filterValue, isFilter := self.filterExpression(v)
		if isFilter {
			return filterValue
		}

		fillValue, isFill := self.fillExpression(v)
		if isFill {
			v = fillValue
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

type Selector struct {
	Xpath      string
	Attr       string
	Regex      string
	Type       string
	Condition  string
	Default    string
	JsonKey    string
	DateFormat *DateFormat
}

func (p *Selector) String() string {
	b, _ := json.Marshal(p)
	return string(b)
}

func NewSelector(v string, extractor *Extractor) *Selector {
	tks := strings.Split(v, ";")
	ret := &Selector{}
	ret.Xpath = tks[0]
	if len(tks) > 1 {
		ret.Attr = tks[1]
	}
	if len(tks) > 2 {
		ret.Regex = tks[2]
	}
	if len(tks) > 3 {
		ret.Type = tks[3]
	}
	if len(tks) > 4 {
		ret.Condition = tks[4]
	}
	if len(tks) > 5 {
		ret.Default = tks[5]
	}
	if len(tks) > 6 {
		ret.DateFormat = extractor.GetDateFormat(tks[6])
	}
	return ret
}

func NewJsonSelector(v string, extractor *Extractor) *Selector {
	tks := strings.Split(v, ";")
	ret := &Selector{}
	ret.JsonKey = tks[0]
	if len(tks) > 1 {
		ret.Type = tks[1]
	}
	if len(tks) > 2 {
		ret.Default = tks[2]
	}
	if len(tks) > 3 {
		ret.DateFormat = extractor.GetDateFormat(tks[3])
	}
	return ret
}

func (self *Selector) extract(buf string) interface{} {
	buf = TrimBeginEndSpace(buf)
	if len(self.Regex) > 0 && strings.Contains(self.Regex, "@multi ") {
		reg := strings.Replace(self.Regex, "@multi ", "", 1)
		return FindGroupsByIndex(reg, buf, 1)
	} else if len(self.Regex) > 0 {
		reg := regexp.MustCompile(self.Regex)
		result := reg.FindAllStringSubmatch(buf, 1)
		if len(result) > 0 {
			group := result[0]
			if len(group) > 1 {
				buf = group[1]
			} else {
				buf = group[0]
			}
		} else {
			dlog.Warn("Regex not found value %s", self.Regex)
			buf = ""
		}
	}
	if len(buf) > 0 && len(self.Condition) > 0 {
		return self.extractCondition(buf)
	}
	if len(buf) > 0 && len(self.Default) > 0 {
		buf = self.Default
	}
	if len(buf) > 0 && self.DateFormat != nil {
		buf = self.DateFormat.Format(buf)
	}
	return self.convertType(buf)
}

func (self *Selector) convertType(content string) interface{} {
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

func (self *Selector) extractCondition(content string) interface{} {
	if strings.HasPrefix(self.Condition, "@function") {
		if strings.HasPrefix(self.Default, "@content") {
			content = self.Default[9:]
		}
		return parseFunction(self.Condition[10:], content)
	}
	ret := make(map[string]string)
	con := strings.Split(self.Condition, ",")
	for _, subCon := range con {
		conResult := strings.Split(subCon, "=")
		if len(conResult) > 1 {
			ret[conResult[0]] = conResult[1]
		} else if len(conResult) == 1 {
			ret[conResult[0]] = ""
		}
	}
	for k, v := range ret {
		if strings.Contains(content, k) {
			return self.convertType(v)
		}
	}
	if len(self.Default) > 0 {
		return self.convertType(self.Default)
	}
	return content
}

func (self *Extractor) extractSingle(v string, s *goquery.Selection) interface{} {
	if v == "@data" {
		data, err := s.Html()
		if err != nil {
			dlog.Warn("MarshalHTML %s", err.Error())
			return ""
		}
		return data
	}
	sel := NewSelector(v, self)
	b := s
	if len(sel.Xpath) > 0 {
		b = queryXpath(sel.Xpath, s)
	}

	if b == nil || b.Size() == 0 {
		return sel.convertType(sel.Default)
	}
	if len(sel.Attr) > 0 {
		if sel.Attr == "html" {
			htmlText, _ := b.First().Html()
			return sel.extract(htmlText)
		}
		ret, _ := b.First().Attr(sel.Attr)
		if (sel.Attr == "href" || sel.Attr == "src") && strings.HasPrefix(ret, "//") {
			ret = "https:" + ret
		}
		return sel.extract(ret)
	} else {
		return sel.extract(b.First().Text())
	}
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

type Function struct {
	function map[string]func(string) interface{}
}

func newFunction() *Function {
	f := &Function{
		function: make(map[string]func(string) interface{}),
	}
	f.function["O_SCORE_HTML"] = O_SCORE_HTML
	f.function["O_RATE_HTML"] = O_RATE_HTML
	return f
}

func parseFunction(funcName, content string) interface{} {
	f := newFunction()
	parsefunc := f.function[funcName]
	return parsefunc(content)
}

func O_SCORE_HTML(content string) interface{} {
	startTime, err := getStartTime(content)
	if err != nil {
		return nil
	}
	matcher := regexp.MustCompile("arrYear\":\\[(.*)\\]")
	regResult := matcher.FindAllStringSubmatch(content, 1)
	if len(regResult[0]) < 2 {
		return nil
	}
	result := strings.Split(regResult[0][1], ",")
	ret := make(map[string]map[string]string)
	for i := 0; i < 12; i++ {
		sub := make(map[string]string)
		var value, diff string
		if i*30 < len(result) && len(result[i*30]) > 0 {
			value = result[i*30]
			if i == 0 {
				diff = "0"
			} else {
				nowMonth, okNow := strconv.Atoi(result[i*30])
				preMonth, okPre := strconv.Atoi(result[i*30-1])
				if okNow != nil || okPre != nil {
					dlog.Warn("O_SCORE_HTML function: string to int err")
					continue
				}
				diff = strconv.Itoa(nowMonth - preMonth)
			}
			sub["总积分"] = value
			sub["积分变动"] = diff
			ret[startTime.Format("2006.01.02")] = sub
			startTime = startTime.AddDate(0, 1, 0)
		}
	}
	return ret
}

func getStartTime(content string) (time.Time, error) {
	buf := []byte(content)
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(buf))
	if err != nil {
		dlog.Warn("goquery O_SCORE_HTML err: %s", err.Error())
		return time.Now(), nil
	}
	startYear := doc.Find(".year-ago p").Eq(0).Text()
	startMonthDay := doc.Find(".year-ago p").Eq(1).Text()
	startTime := startYear + "." + startMonthDay
	return time.Parse("2006.01.02", startTime)
}

func O_RATE_HTML(content string) interface{} {
	startTime, err := time.Parse("Mon Jan 02 15:04:05 CST 2006", content)
	if err != nil {
		return nil
	}
	duration, _ := time.ParseDuration("24h")
	startTime = startTime.Add(duration)
	return startTime.Format("2006-01-02")
}
