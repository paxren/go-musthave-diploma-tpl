package config

import (
	"flag"
	"reflect"
	"strings"

	"github.com/caarlos0/env/v11"
)

type ServerConfigEnv struct {
	AccrualSystemAddress HostAddress `env:"ACCRUAL_SYSTEM_ADDRESS,notEmpty"`
	RunAddress           HostAddress `env:"RUN_ADDRESS,notEmpty"`
	DatabaseURI          string      `env:"DATABASE_URI,notEmpty"`
}

type ServerConfig struct {
	envs                 ServerConfigEnv
	AccrualSystemAddress HostAddress
	RunAddress           HostAddress
	DatabaseURI          string

	paramAccrualSystemAddress HostAddress
	paramRunAddress           HostAddress
	paramDatabaseURI          string
}

func NewServerConfig() *ServerConfig {

	acrual := &HostAddress{
		Host: "localhost",
		Port: 8081,
	}

	return &ServerConfig{
		RunAddress:           *NewHostAddress(),
		AccrualSystemAddress: *acrual,
	}

}

func (se *ServerConfig) Init() {
	// fmt.Printf("start init:\n\n")
	// fmt.Println("======BEFORE PARAMS PARSE-----")
	// fmt.Printf("paramStoreInterval = %v\n", se.paramStoreInterval)
	// fmt.Printf("paramFileStoragePath = %v\n", se.paramFileStoragePath)
	// fmt.Printf("paramRestore = %v\n", se.paramRestore)
	// fmt.Printf("paramAdress = %v\n", se.paramAddress)
	// fmt.Printf("StoreInterval = %v\n", se.StoreInterval)
	// fmt.Printf("FileStoragePath = %v\n", se.FileStoragePath)
	// fmt.Printf("Restore = %v\n", se.Restore)
	// fmt.Printf("Adress = %v\n", se.Address)

	se.paramAccrualSystemAddress = HostAddress{
		Host: "localhost",
		Port: 8081,
	}
	se.paramRunAddress = HostAddress{
		Host: "localhost",
		Port: 8080,
	}

	flag.Var(&se.paramRunAddress, "a", "Net address host:port")
	flag.StringVar(&se.paramDatabaseURI, "d", "", "db uri")
	flag.Var(&se.paramAccrualSystemAddress, "r", "Net accrual address host:port")

	// fmt.Println("======AFTER PARAMS PARSE-----")
	// fmt.Printf("paramStoreInterval = %v\n", se.paramStoreInterval)
	// fmt.Printf("paramFileStoragePath = %v\n", se.paramFileStoragePath)
	// fmt.Printf("paramRestore = %v\n", se.paramRestore)
	// fmt.Printf("paramAdress = %v\n", se.paramAddress)
	// fmt.Printf("StoreInterval = %v\n", se.StoreInterval)
	// fmt.Printf("FileStoragePath = %v\n", se.FileStoragePath)
	// fmt.Printf("Restore = %v\n", se.Restore)
	// fmt.Printf("Adress = %v\n", se.Address)
}

func (se *ServerConfig) Parse() {

	// fmt.Println("======BEFORE ENV PARSE-----")
	// fmt.Printf("paramStoreInterval = %v\n", se.paramStoreInterval)
	// fmt.Printf("paramFileStoragePath = %v\n", se.paramFileStoragePath)
	// fmt.Printf("paramRestore = %v\n", se.paramRestore)
	// fmt.Printf("paramAdress = %v\n", se.paramAddress)
	// fmt.Printf("StoreInterval = %v\n", se.StoreInterval)
	// fmt.Printf("FileStoragePath = %v\n", se.FileStoragePath)
	// fmt.Printf("Restore = %v\n", se.Restore)
	// fmt.Printf("Adress = %v\n", se.Address)

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
		// fmt.Printf("err type %T:\n\n", err)
		if err, ok := err.(env.AggregateError); ok {
			// fmt.Printf("err.Errors: %v\n\n", err.Errors)

			for _, v := range err.Errors {
				//fmt.Printf("err.Error: %T\n", v)
				//fmt.Printf("err.Error: %v\n", v)

				if err1, ok := v.(env.EmptyVarError); ok {
					// fmt.Printf("err.EmptyVarError: %v\n", err1)
					// fmt.Printf("err.EmptyVarError.Key: %v\n", err1.Key)

					problemVars[err1.Key] = true
				}

				if err2, ok := v.(env.ParseError); ok {
					// fmt.Printf("err.ParseError: %v\n", err2)
					// fmt.Printf("err.ParseError.Name: %v\n", err2.Name)
					// fmt.Printf("err.ParseError.Type: %v\n", err2.Type)
					// fmt.Printf("err.ParseError.Err: %v\n", err2.Err)

					problemVars[err2.Name] = true
				}

				if _, ok := v.(HostAddressParseError); ok {
					// Определяем, для какого адреса произошла ошибка
					// Проверяем, если ошибка связана с RUN_ADDRESS
					if strings.Contains(v.Error(), "RUN_ADDRESS") {
						problemVars["RUN_ADDRESS"] = true
					} else {
						// Иначе считаем, что ошибка связана с ACCRUAL_SYSTEM_ADDRESS
						problemVars["ACCRUAL_SYSTEM_ADDRESS"] = true
					}
				}

				//fmt.Println("----------------------")
			}

		}
	}

	//fmt.Printf("problemVars = %v", problemVars)
	flag.Parse()

	// fmt.Println("======FLAG PARSED-----")
	// fmt.Printf("paramStoreInterval = %v\n", se.paramStoreInterval)
	// fmt.Printf("paramFileStoragePath = %v\n", se.paramFileStoragePath)
	// fmt.Printf("paramRestore = %v\n", se.paramRestore)
	// fmt.Printf("paramAdress = %v\n", se.paramAddress)
	// fmt.Printf("StoreInterval = %v\n", se.StoreInterval)
	// fmt.Printf("FileStoragePath = %v\n", se.FileStoragePath)
	// fmt.Printf("Restore = %v\n", se.Restore)
	// fmt.Printf("Adress = %v\n", se.Address)

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

	// fmt.Println("======RESULT-----")
	// fmt.Printf("paramStoreInterval = %v\n", se.paramStoreInterval)
	// fmt.Printf("paramFileStoragePath = %v\n", se.paramFileStoragePath)
	// fmt.Printf("paramRestore = %v\n", se.paramRestore)
	// fmt.Printf("paramAdress = %v\n", se.paramAddress)
	// fmt.Printf("StoreInterval = %v\n", se.StoreInterval)
	// fmt.Printf("FileStoragePath = %v\n", se.FileStoragePath)
	// fmt.Printf("Restore = %v\n", se.Restore)
	// fmt.Printf("Adress = %v\n", se.Address)
}

// GetAccrualSystemURL возвращает полный URL для системы начисления баллов с схемой http
func (se *ServerConfig) GetAccrualSystemURL() string {
	return "http://" + se.AccrualSystemAddress.String()
}
