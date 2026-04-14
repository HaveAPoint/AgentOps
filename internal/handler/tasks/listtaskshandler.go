// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package tasks

import (
	"net/http"

	"agentops/internal/logic/tasks"
	"agentops/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ListTasksHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := tasks.NewListTasksLogic(r.Context(), svcCtx)
		resp, err := l.ListTasks()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
