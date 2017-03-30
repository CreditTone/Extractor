package extractor

import (
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"

	"zhongguo/authcrawler/dlog"

	"github.com/bitly/go-simplejson"
)

func FindResult(reg, body string) [][]string {
	matcher := regexp.MustCompile(reg)
	result := matcher.FindAllStringSubmatch(body, -1)
	return result
}

func FindGroupsByIndex(reg, body string, index int) []string {
	groups := make([]string, 0)
	results := FindResult(reg, body)
	for _, mat := range results {
		if mat != nil && len(mat) > index {
			groups = append(groups, mat[index])
		}
	}
	return groups
}

func FindGroup(reg, body string) []string {
	matcher := regexp.MustCompile(reg)
	result := matcher.FindAllStringSubmatch(body, 1)
	if len(result) > 0 {
		group := result[0]
		return group
	}
	return nil
}

func FindGroupByIndex(reg, body string, index int) string {
	group := FindGroup(reg, body)
	if group != nil && len(group) > index {
		return group[index]
	}
	return ""
}

func FindGroupOne(reg, body string) string {
	return FindGroupByIndex(reg, body, 1)
}

func FindAttribute(attributeName, expression string) (string, bool) {
	attributeExpression := "@" + attributeName
	if !strings.Contains(expression, attributeExpression) {
		return "", false
	}
	reg := attributeExpression + "\\s*=\\s*([^\\s]+)\\s*"
	val := FindGroupOne(reg, expression)
	return val, true
}

func FindIntAttribute(attributeName, expression string) (int, bool) {
	val, ok := FindAttribute(attributeName, expression)
	if ok {
		ret, err := strconv.Atoi(val)
		if err != nil {
			dlog.Warn("convert to int err :%s", err.Error())
			return 0, false
		}
		return ret, true
	}
	return 0, false
}

func CleanAttribute(attributeName, expression string) string {
	reg := "(@" + attributeName + "\\s*=\\s*[^\\s]+)\\s*"
	val := FindGroupOne(reg, expression)
	if len(val) > 0 {
		return strings.Replace(expression, val, "", 1)
	}
	return expression
}

func FilterJSONP(s string) string {
	p0 := strings.Index(s, "{")
	p1 := strings.Index(s, "(")
	if p1 < 0 {
		return s
	} else if p1 >= 0 && p1 > p0 {
		return s
	}
	p1 += 1
	p2 := strings.LastIndex(s, ")")
	if p2 <= p1 {
		return s
	}
	return strings.Trim(s[p1:p2], "\"")
}

func EncodeString(json *simplejson.Json) string {
	data, err := json.Encode()
	if err == nil {
		ret := string(data)
		ret = strings.TrimLeft(ret, "\"")
		ret = strings.TrimRight(ret, "\"")
		return ret
	} else {
		dlog.Warn("json encode error %v", err)
	}
	return ""
}

func ReadFile(path string) []byte {
	f, err := os.Open(path)
	if err != nil {
		dlog.Error("fail to open %s: %v", path, err)
		return nil
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		dlog.Error("fail to read: %v", err)
		return nil
	}
	return b
}
