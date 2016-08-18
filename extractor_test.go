package extractor

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUnicode(t *testing.T) {
	data := map[string]interface{}{"url": "&"}
	body, _ := json.Marshal(data)
	ret := strings.Replace(string(body), "\\u0026", "", -1)
	t.Log(ret)
}
