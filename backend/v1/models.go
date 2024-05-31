package v1

import (
	baseservice "github.com/cool-team-official/cool-admin-go/modules/base/service"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
)

type OpenAIModel struct {
	ID         string `json:"id"`
	Object     string `json:"object"`
	Created    int64  `json:"created"`
	OwnedBy    string `json:"owned_by"`
	Permission []struct {
		ID                 string `json:"id"`
		Object             string `json:"object"`
		Created            int64  `json:"created"`
		AllowCreateEngine  bool   `json:"allow_create_engine"`
		AllowSampling      bool   `json:"allow_sampling"`
		AllowLogprobs      bool   `json:"allow_logprobs"`
		AllowSearchIndices bool   `json:"allow_search_indices"`
		AllowView          bool   `json:"allow_view"`
		AllowFineTuning    bool   `json:"allow_fine_tuning"`
		Organization       string `json:"organization"`
		Group              string `json:"group"`
		IsBlocking         bool   `json:"is_blocking"`
	} `json:"permission"`
	Root   string `json:"root"`
	Parent string `json:"parent"`
}

type OpenAIModelsResponse struct {
	Data    []OpenAIModel `json:"data"`
	Success bool          `json:"success"`
}

func Models(r *ghttp.Request) {
	ctx := r.GetCtx()
	modelMapStr, err := baseservice.NewBaseSysParamService().DataByKey(ctx, "modelmap")
	if err != nil {
		r.Response.Status = 500
		r.Response.WriteJsonExit(g.Map{
			"succeed": false,
			"message": "获取模型列表失败",
		})
	}
	modelMap := gconv.MapStrStr(gjson.New(modelMapStr))
	models := make([]OpenAIModel, 0)
	for key, _ := range modelMap {
		models = append(models, OpenAIModel{
			ID:      key,
			Object:  "model",
			Created: gtime.Now().Timestamp(),
		})
	}
	r.Response.WriteJsonExit(OpenAIModelsResponse{
		Data:    models,
		Success: true,
	})

}
