// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package tasks

import (
	"net/http"

	"agentops/internal/logic/tasks"
	"agentops/internal/svc"
	"agentops/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ApproveTaskHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ApproveTaskReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := tasks.NewApproveTaskLogic(r.Context(), svcCtx)
		resp, err := l.ApproveTask(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
