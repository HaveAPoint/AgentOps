package tasks

import (
	"net/http"

	"agentops/internal/logic/tasks"
	"agentops/internal/svc"
	"agentops/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetTaskExecutionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetTaskExecutionsReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := tasks.NewGetTaskExecutionsLogic(r.Context(), svcCtx)
		resp, err := l.GetTaskExecutions(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
