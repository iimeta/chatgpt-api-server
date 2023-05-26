package modules

import (
	_ "github.com/cool-team-official/cool-admin-go/modules/base"
	_ "github.com/cool-team-official/cool-admin-go/modules/dict"
	_ "github.com/cool-team-official/cool-admin-go/modules/space"
	_ "github.com/cool-team-official/cool-admin-go/modules/task"

	_ "chatgpt-api-server/modules/chatgpt" // 引入chatgpt模块
)
