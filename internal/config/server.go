package config

import (
	"flag"
	"reflect"
	"strings"

	"github.com/caarlos0/env/v11"
)

type ServerConfigEnv struct {
	AccrualSystemAddress string      `env:"ACCRUAL_SYSTEM_ADDRESS,notEmpty"`
	RunAddress           HostAddress `env:"RUN_ADDRESS,notEmpty"`
	DatabaseURI          string      `env:"DATABASE_URI,notEmpty"`
	JWTSecret            string      `env:"JWT_SECRET"`
}

type ServerConfig struct {
	envs                 ServerConfigEnv
	AccrualSystemAddress string
	RunAddress           HostAddress
	DatabaseURI          string
	JWTSecret            string

	paramAccrualSystemAddress string
	paramRunAddress           HostAddress
	paramDatabaseURI          string
	paramJWTSecret            string
}

func NewServerConfig() *ServerConfig {
	return &ServerConfig{
		RunAddress: *NewHostAddress(),
	}
}

func (se *ServerConfig) Init() {
	se.paramRunAddress = HostAddress{
		Host: "localhost",
		Port: 8080,
	}

	flag.Var(&se.paramRunAddress, "a", "Net address host:port")
	flag.StringVar(&se.paramDatabaseURI, "d", "", "db uri")
	flag.StringVar(&se.paramAccrualSystemAddress, "r", "http://localhost:8081", "Net accrual address http://host:port")
	flag.StringVar(&se.paramJWTSecret, "j", "default-secret-key", "JWT secret key")
}

func (se *ServerConfig) Parse() {
	err := env.ParseWithOptions(&se.envs, env.Options{
		FuncMap: map[reflect.Type]env.ParserFunc{
			reflect.TypeOf(HostAddress{}): func(v string) (interface{}, error) {

				ha := NewHostAddress()
				err := ha.Set(v)

				return *ha, err
			},
		},
	})

	problemVars := make(map[string]bool)

	if err != nil {
		if err, ok := err.(env.AggregateError); ok {
			for _, v := range err.Errors {
				if err1, ok := v.(env.EmptyVarError); ok {
					problemVars[err1.Key] = true
				}

				if err2, ok := v.(env.ParseError); ok {
					problemVars[err2.Name] = true
				}

				if _, ok := v.(HostAddressParseError); ok {
					// Определяем, для какого адреса произошла ошибка
					// Проверяем, если ошибка связана с RUN_ADDRESS
					if strings.Contains(v.Error(), "RUN_ADDRESS") {
						problemVars["RUN_ADDRESS"] = true
					}
				}
			}
		}
	}

	flag.Parse()

	_, ok1 := problemVars["ACCRUAL_SYSTEM_ADDRESS"]
	_, ok2 := problemVars["AccrualSystemAddress"]
	if !ok1 && !ok2 {
		se.AccrualSystemAddress = se.envs.AccrualSystemAddress
	} else {
		se.AccrualSystemAddress = se.paramAccrualSystemAddress
	}

	_, ok1 = problemVars["RUN_ADDRESS"]
	_, ok2 = problemVars["RunAddress"]
	if !ok1 && !ok2 {
		se.RunAddress = se.envs.RunAddress
	} else {
		se.RunAddress = se.paramRunAddress
	}

	_, ok1 = problemVars["DATABASE_URI"]
	_, ok2 = problemVars["DatabaseURI"]
	if !ok1 && !ok2 {
		se.DatabaseURI = se.envs.DatabaseURI
	} else {
		se.DatabaseURI = se.paramDatabaseURI
	}

	_, ok1 = problemVars["JWT_SECRET"]
	_, ok2 = problemVars["JWTSecret"]
	if !ok1 && !ok2 {
		se.JWTSecret = se.envs.JWTSecret
	} else {
		se.JWTSecret = se.paramJWTSecret
	}
}
