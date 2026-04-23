// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package config

import (
	"time"

	"github.com/zeromicro/go-zero/rest"
)

const DefaultExecutorTimeoutSeconds int64 = 30

type PostgresConf struct {
	Host                   string
	Port                   int
	User                   string
	Password               string
	DBName                 string
	SSLMode                string
	MaxOpenConns           int
	MaxIdleConns           int
	ConnMaxLifetimeSeconds int
}

type AuthConf struct {
	AccessSecret string
	AccessExpire int64
}

type ExecutorConf struct {
	Provider         string
	Command          string
	Args             []string
	TimeoutSeconds   int64
	AllowedRepoPaths []string
}

func (c ExecutorConf) Timeout() time.Duration {
	if c.TimeoutSeconds <= 0 {
		return time.Duration(DefaultExecutorTimeoutSeconds) * time.Second
	}
	return time.Duration(c.TimeoutSeconds) * time.Second
}

type Config struct {
	rest.RestConf
	Auth     AuthConf
	Postgres PostgresConf
	Executor ExecutorConf
}
