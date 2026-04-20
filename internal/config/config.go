// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package config

import "github.com/zeromicro/go-zero/rest"

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

type Config struct {
	rest.RestConf
	Auth     AuthConf
	Postgres PostgresConf
}
