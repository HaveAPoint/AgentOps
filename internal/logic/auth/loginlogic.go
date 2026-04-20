// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	authutil "agentops/internal/auth"
	"agentops/internal/svc"
	"agentops/internal/types"

	"github.com/golang-jwt/jwt/v4"
	"github.com/zeromicro/go-zero/core/logx"
)

var ErrInvalidCredentials = errors.New("invalid username or password")

var errLoginFailed = errors.New("login failed")

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.LoginResp, err error) {
	if req == nil {
		return nil, ErrInvalidCredentials
	}

	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	if username == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := l.svcCtx.UserModel.FindByUsername(l.ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		l.Errorf("find user by username failed: %v", err)
		return nil, errLoginFailed
	}

	ok, err := authutil.VerifyPassword(password, user.PasswordHash)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if !ok {
		return nil, ErrInvalidCredentials
	}

	expiresIn := l.svcCtx.Config.Auth.AccessExpire
	if expiresIn <= 0 {
		return nil, errors.New("auth access expire must be greater than 0")
	}

	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"iat":        now.Unix(),
		"exp":        now.Add(time.Duration(expiresIn) * time.Second).Unix(),
		"userId":     user.ID,
		"username":   user.Username,
		"systemRole": user.SystemRole,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte(l.svcCtx.Config.Auth.AccessSecret))
	if err != nil {
		return nil, err
	}

	return &types.LoginResp{
		AccessToken: accessToken,
		ExpiresIn:   expiresIn,
	}, nil
}
