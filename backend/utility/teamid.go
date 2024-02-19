package utility

import (
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
)

// GetTeamIdByAccountInfo 根据账号信息获取teamId
func GetTeamIdByAccountInfo(ctx g.Ctx, accountInfo string) (teamIds []string) {
	accountInfoJson := gjson.New(accountInfo)
	// accountInfoJson.Dump()
	// 获取 account_ordering 文本数组
	accountOrdering := accountInfoJson.Get("account_ordering").Strings()
	for _, v := range accountOrdering {
		// teamInfo:= accountInfoJson.GetJson(v + ".team_info")
		is_deactivated := accountInfoJson.Get("accounts." + v + ".account.is_deactivated").String()
		plan_type := accountInfoJson.Get("accounts." + v + ".account.plan_type").String()
		g.Log().Debug(ctx, "GetTeamIdByAccountInfo", "account_ordering", v, "is_deactivated", is_deactivated, "plan_type", plan_type)
		// 如果 is_deactivated 为 false 并且 plan_type 为 team 就获取 teamId
		if is_deactivated == "false" && plan_type == "team" {
			teamIds = append(teamIds, v)
		}
	}

	return
}
