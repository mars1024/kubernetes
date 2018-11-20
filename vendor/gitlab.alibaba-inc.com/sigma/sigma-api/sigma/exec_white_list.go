package sigma

//AUTHOR： 智清
//需求背景：http://docs.alibaba-inc.com/pages/viewpage.action?pageId=656933898
//EXEC执行白名单

type ExecWhiteList struct {
	UpdateTime string   `json:",omitempty"` // "UpdateTime": "2016-06-29 13:14:16","2016-06-29 13:14:16"
	ExecCmds   []string `json:",omitempty"` // "ExecCmds": "2013-02-25 22:39:58"
}
