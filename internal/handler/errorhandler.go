package handler

import (
	"context"
	"errors"
	"net/http"

	authctx "agentops/internal/auth"
	authlogic "agentops/internal/logic/auth"
	tasklogic "agentops/internal/logic/tasks"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func init() {
	httpx.SetErrorHandlerCtx(func(_ context.Context, err error) (int, any) {
		status := http.StatusBadRequest

		switch {
		case errors.Is(err, authctx.ErrUnauthenticated),
			errors.Is(err, authlogic.ErrInvalidCredentials):
			status = http.StatusUnauthorized
		case errors.Is(err, tasklogic.ErrPermissionDenied):
			status = http.StatusForbidden
		case errors.Is(err, tasklogic.ErrTaskNotFound):
			status = http.StatusNotFound
		}

		return status, map[string]string{
			"error": err.Error(),
		}
	})
}
