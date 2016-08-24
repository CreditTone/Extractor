package extractor

import (
	"encoding/json"
	"io/ioutil"
	"strings"
	"testing"
)

func TestUnicode(t *testing.T) {
	data := map[string]interface{}{"url": "&"}
	body, _ := json.Marshal(data)
	ret := strings.Replace(string(body), "\\u0026", "", -1)
	t.Log(ret)
}

func TestPPD(t *testing.T) {
	extractor := NewExtractor()
	config := `
				{
                    "_root":"p:contains('真实姓名')",
                    "url":"a;href"
                }
	`
	data, _ := ioutil.ReadFile("ppd.html")
	//t.Log(string(data))
	m := map[string]interface{}{}
	json.Unmarshal([]byte(config), &m)
	ret := extractor.Do(m, data)
	t.Log(ret)
}
