package extractor

import (
	"encoding/json"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"strings"
	"testing"
)

func createExtractor() *Extractor {
	filter := func(config string) (interface{}, bool) {
		return nil, false
	}
	fill := func(config string) (string, bool) {
		return "", false
	}
	return NewExtractor(filter, fill)
}

func TestMultiPattern(t *testing.T) {
	extractor := createExtractor()
	body, err := ioutil.ReadFile("hotsell.htm")
	if err != nil {
		t.Error(err)
	}
	//t.Log(string(body))
	config :=
		`
		{	
			"sale_num":";html;@multi 已售：<[^>]+>([\\d]+)",
			"c_price":";html;@multi c-price[^>]+>([\\d\\.]+)",
			"pingjia":";html;@multi 评论\\([^\\)]+>(\\d+)<[^\\)]+\\)",
			"id":";html;@multi item-name[^>]+item.htm\\?id=(\\d+)"
		}
		`
	c := map[string]interface{}{}
	err = json.Unmarshal([]byte(config), &c)
	if err != nil {
		t.Error(err.Error())
	}
	ret := extractor.Do(c, body)
	json, _ := json.Marshal(ret)
	t.Log(string(json))
}

func TestDupkey(t *testing.T) {
	html := `
		<html>
			<body>
				<div class="loan-mng-chart-container" style="background:none">
					<div class="top-title">使用中的贷款情况<span>(未还本金)</span>：</div>
					<div class="total-content">
						<p><strong class="cOrange">13</strong>笔贷款总额：</p>
						<p>
							<strong class="total-Amt cGreen">115,585.13</strong>元
						</p>
					</div>
				</div>
			</body>
		</html>
	`
	ex := createExtractor()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Error(err)
		return
	}
	ret := ex.extract(map[string]interface{}{
		"@dupkey_1 使用中的贷款笔数": ".loan-mng-chart-container .cOrange",
		"@dupkey_0 使用中的贷款笔数": ".loan-mng-normal-content .cOrange",
		"@dupkey_0 贷款总额":     ".loan-mng-normal-content .cGreen",
		"@dupkey_1 贷款总额":     ".loan-mng-chart-container .total-content .cGreen",
	}, doc.First())
	t.Log(ret)
	mret := ret.(map[string]interface{})
	if val, ok := mret["使用中的贷款笔数"]; val.(string) != "13" || !ok {
		t.Error()
	}
}

func TestSingleSelect(t *testing.T) {
	html := `
		<html>
			<body>
				<p class="hello">World</p>
			</body>
		</html>
	`
	ex := createExtractor()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Error(err)
		return
	}
	ret := ex.extractSingle(".hello;;;;World=China", doc.First())
	if ret != "China" {
		t.Error(ret)
	}
	ret = ex.extractSingle(".hello;;;;World2=China", doc.First())
	if ret != "World" {
		t.Error(ret)
	}
}

func TestExtractor(t *testing.T) {
	body, err := ioutil.ReadFile("viewReport2.htm")
	if err != nil {
		t.Error(err)
	}
	//t.Log(string(body))
	config :=
		`
		{
        "creditReportInfo":{
          "crptId":";html;报告编号：\\s*(\\d+)\\s*",
          "searchTime":";html;查询时间：([^<]+) <",
          "reportTime":";html;报告时间：([^<]+) <",
          "realName":";html;姓名：\\s*([^<]+) <",
          "cardType":";html;证件类型：([^<]+) <",
          "cardId":";html;证件号码：([^<]+) <",
          "mariStat":";html;>([已未]婚) <"
        },
        "creditSummary":{
          "_root":"table[width='100%'][border='1'][height='155'][cellspacing='0'][bordercolor='#e5e1e1'] tr:contains('数')",
          "creditCard":"td @index=1",
          "homeLoan":"td @index=2",
          "otherLoan":"td @index=3"
        },
        "creditCardList":{
          "_root":"ol[class='p olstyle'] li:contains('贷记卡')",
          "currency":";html;（(\\S+)账户）",
          "giveDate":";html;\\d+年\\d+月\\d+日",
          "abortDate ":";html;截至(\\d+年\\d+月)",
          "limitAmount":";html;信用额度[^\\d]*([\\d,]+)",
          "bankName":";html;\\d+年\\d+月\\d+日(\\S+)发放",
          "usedAmount ":";html;已使用额度([\\d,]+)",
          "destroyDate ":";html;截至(\\d+年\\d+月)已销户",
          "overdueAmount ":";html;透支额度([\\d,]+)",
          "line":""
        },
        "creditLoanList":{
          "_root":"ol[class='p olstyle'] li:contains('贷款')",
          "loanType":";html;）([^，]+)，",
          "giveDate":";html;(\\d+年\\d+月\\d+日)\\S+发放的",
          "expireDate":";html;(\\d+年\\d+月\\d+日)到期",
          "abortDate":";html;截至(\\d+年\\d+月)",
          "bankName ":";html;\\d+年\\d+月\\d+日(\\S+)发放",
          "loanContractAmount":";html;发放的([\\d,]+)",
          "loanBalance":";html;余额([\\d,]+)",
          "recently5YExpireTimes":";html;最近5年内有(\\d+)个月处于逾期状态",
          "line":""
        },
        "creditSearchList":{
          "_root":"table[width='980'][style='margin-top: 12px']:contains('机构查询记录明细') tr:contains('201')",
          "searchId":"td @index=0",
          "searchDate":"td @index=1",
          "operator":"td @index=2",
          "searchReason":"td @index=3"
        },
        "guaranteeList":{

        }
      }
	`
	c := map[string]interface{}{}
	err = json.Unmarshal([]byte(config), &c)
	if err != nil {
		t.Error(err.Error())
	}
	extractor := Extractor{}
	ret := extractor.Do(c, body)
	json, _ := json.Marshal(ret)
	t.Log(string(json))
}
