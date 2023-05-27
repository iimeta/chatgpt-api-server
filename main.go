package main

import (
	_ "chatgpt-api-server/internal/packed"

	_ "github.com/cool-team-official/cool-admin-go/contrib/drivers/mysql"
	_ "github.com/cool-team-official/cool-admin-go/contrib/drivers/sqlite"
	_ "github.com/gogf/gf/contrib/nosql/redis/v2"

	_ "chatgpt-api-server/backend-api"
	_ "chatgpt-api-server/modules"

	"github.com/gogf/gf/v2/os/gctx"

	"chatgpt-api-server/internal/cmd"
)

func main() {
	// gres.Dump()
	cmd.Main.Run(gctx.New())
}
