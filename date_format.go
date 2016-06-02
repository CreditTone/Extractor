package extractor

import (
	"github.com/xlvector/dlog"
	"time"
)

type DateFormat struct {
	Input  string
	Output string
}

func NewDateFormat(input, output string) *DateFormat {
	ret := DateFormat{
		Input:  input,
		Output: output,
	}
	return &ret
}

func (self *DateFormat) Format(dateTime string) string {
	timeformatdate, err := time.Parse(self.Input, dateTime)
	if err != nil {
		dlog.Warn("format date %s", err.Error())
		return ""
	}
	convdate := timeformatdate.Format(self.Output)
	return convdate
}
