package config

import (
	"flag"
	"os"
	"testing"
)

// Вспомогательные функции для тестов

// setEnvVars устанавливает временные переменные окружения для тестов
func setEnvVars(envVars map[string]string) func() {
	originalVars := make(map[string]string)

	for key, value := range envVars {
		originalVars[key] = os.Getenv(key)
		if value != "" {
			os.Setenv(key, value)
		} else {
			os.Unsetenv(key)
		}
	}

	// Возвращаем функцию для восстановления исходных значений
	return func() {
		for key, value := range originalVars {
			if value != "" {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		}
	}
}

// resetFlags сбрасывает флаги командной строки
func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
}

// saveArgs сохраняет текущие аргументы командной строки
func saveArgs() []string {
	return append([]string{}, os.Args...)
}

// restoreArgs восстанавливает аргументы командной строки
func restoreArgs(args []string) {
	os.Args = args
}

// createTestConfig создает новую конфигурацию для тестов
func createTestConfig() *ServerConfig {
	config := NewServerConfig()
	resetFlags()
	return config
}

// assertHostAddress проверяет равенство HostAddress
func assertHostAddress(t *testing.T, expected, actual HostAddress) {
	t.Helper()
	if expected.Host != actual.Host {
		t.Errorf("Expected host %s, got %s", expected.Host, actual.Host)
	}
	if expected.Port != actual.Port {
		t.Errorf("Expected port %d, got %d", expected.Port, actual.Port)
	}
}

// Тесты для функции NewServerConfig
func TestNewServerConfig(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "Создание конфигурации с значениями по умолчанию",
			test: func(t *testing.T) {
				config := NewServerConfig()

				if config == nil {
					t.Fatal("Expected config to be created, got nil")
				}

				// Проверяем, что RunAddress инициализирован
				expectedHost := "localhost"
				expectedPort := 8080

				if config.RunAddress.Host != expectedHost {
					t.Errorf("Expected RunAddress.Host to be %s, got %s", expectedHost, config.RunAddress.Host)
				}

				if config.RunAddress.Port != expectedPort {
					t.Errorf("Expected RunAddress.Port to be %d, got %d", expectedPort, config.RunAddress.Port)
				}
			},
		},
		{
			name: "Проверка инициализации полей",
			test: func(t *testing.T) {
				config := NewServerConfig()

				// Проверяем, что все поля инициализированы
				if config.AccrualSystemAddress.Host != "" || config.AccrualSystemAddress.Port != 0 {
					t.Errorf("Expected AccrualSystemAddress to be empty, got %s", config.AccrualSystemAddress.String())
				}

				if config.DatabaseURI != "" {
					t.Errorf("Expected DatabaseURI to be empty, got %s", config.DatabaseURI)
				}

				// Проверяем, что параметр-поля инициализированы нулевыми значениями
				if config.paramAccrualSystemAddress.Host != "" || config.paramAccrualSystemAddress.Port != 0 {
					t.Errorf("Expected paramAccrualSystemAddress to be empty, got %s", config.paramAccrualSystemAddress.String())
				}

				if config.paramDatabaseURI != "" {
					t.Errorf("Expected paramDatabaseURI to be empty, got %s", config.paramDatabaseURI)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

// Тесты для функции Init
func TestInit(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "Инициализация флагов",
			test: func(t *testing.T) {
				config := createTestConfig()

				// Вызываем Init для установки флагов
				config.Init()

				// Проверяем, что флаги были установлены (проверяем через CommandLine)
				if flag.CommandLine.Lookup("a") == nil {
					t.Error("Expected flag 'a' to be registered")
				}

				if flag.CommandLine.Lookup("d") == nil {
					t.Error("Expected flag 'd' to be registered")
				}

				if flag.CommandLine.Lookup("r") == nil {
					t.Error("Expected flag 'r' to be registered")
				}
			},
		},
		{
			name: "Проверка значений флагов по умолчанию",
			test: func(t *testing.T) {
				config := createTestConfig()

				// Устанавливаем флаги
				config.Init()

				// Проверяем начальные значения параметров
				if config.paramAccrualSystemAddress.Host != "" || config.paramAccrualSystemAddress.Port != 0 {
					t.Errorf("Expected paramAccrualSystemAddress to be empty, got %s", config.paramAccrualSystemAddress.String())
				}

				if config.paramDatabaseURI != "" {
					t.Errorf("Expected paramDatabaseURI to be empty, got %s", config.paramDatabaseURI)
				}

				// Проверяем RunAddress (должен иметь нулевые значения, так как флаги еще не установлены)
				// paramRunAddress - это значение флага, которое будет установлено при парсинге
				// До парсинга оно имеет нулевые значения
				if config.paramRunAddress.Host != "" {
					t.Errorf("Expected paramRunAddress.Host to be empty, got %s", config.paramRunAddress.Host)
				}

				if config.paramRunAddress.Port != 0 {
					t.Errorf("Expected paramRunAddress.Port to be 0, got %d", config.paramRunAddress.Port)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

// Тесты для функции Parse с корректными переменными окружения
func TestParseWithValidEnvVars(t *testing.T) {
	tests := []struct {
		name                   string
		envVars                map[string]string
		expectedAccrualAddress HostAddress
		expectedRunAddress     HostAddress
		expectedDatabaseURI    string
	}{
		{
			name: "Все переменные окружения установлены корректно",
			envVars: map[string]string{
				"ACCRUAL_SYSTEM_ADDRESS": "localhost:8080",
				"RUN_ADDRESS":            "localhost:9000",
				"DATABASE_URI":           "postgres://user:password@localhost:5432/dbname",
			},
			expectedAccrualAddress: HostAddress{
				Host: "localhost",
				Port: 8080,
			},
			expectedRunAddress: HostAddress{
				Host: "localhost",
				Port: 9000,
			},
			expectedDatabaseURI: "postgres://user:password@localhost:5432/dbname",
		},
		{
			name: "Переменные с IP-адресом вместо localhost",
			envVars: map[string]string{
				"ACCRUAL_SYSTEM_ADDRESS": "192.168.1.100:8080",
				"RUN_ADDRESS":            "192.168.1.100:9000",
				"DATABASE_URI":           "postgres://user:password@192.168.1.100:5432/dbname",
			},
			expectedAccrualAddress: HostAddress{
				Host: "192.168.1.100",
				Port: 8080,
			},
			expectedRunAddress: HostAddress{
				Host: "192.168.1.100",
				Port: 9000,
			},
			expectedDatabaseURI: "postgres://user:password@192.168.1.100:5432/dbname",
		},
		{
			name: "Разные порты для разных сервисов",
			envVars: map[string]string{
				"ACCRUAL_SYSTEM_ADDRESS": "localhost:8081",
				"RUN_ADDRESS":            "localhost:8082",
				"DATABASE_URI":           "postgres://user:password@localhost:5433/testdb",
			},
			expectedAccrualAddress: HostAddress{
				Host: "localhost",
				Port: 8081,
			},
			expectedRunAddress: HostAddress{
				Host: "localhost",
				Port: 8082,
			},
			expectedDatabaseURI: "postgres://user:password@localhost:5433/testdb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем переменные окружения
			cleanup := setEnvVars(tt.envVars)
			defer cleanup()

			// Создаем и инициализируем конфигурацию
			config := createTestConfig()
			config.Init()

			// Вызываем Parse
			config.Parse()

			// Проверяем результаты
			assertHostAddress(t, tt.expectedAccrualAddress, config.AccrualSystemAddress)
			assertHostAddress(t, tt.expectedRunAddress, config.RunAddress)

			if config.DatabaseURI != tt.expectedDatabaseURI {
				t.Errorf("Expected DatabaseURI %s, got %s", tt.expectedDatabaseURI, config.DatabaseURI)
			}
		})
	}
}

// Тесты для функции Parse с флагами командной строки
func TestParseWithFlags(t *testing.T) {
	tests := []struct {
		name                   string
		envVars                map[string]string
		flags                  []string
		expectedAccrualAddress HostAddress
		expectedRunAddress     HostAddress
		expectedDatabaseURI    string
	}{
		{
			name: "Только флаги командной строки",
			envVars: map[string]string{
				"ACCRUAL_SYSTEM_ADDRESS": "",
				"RUN_ADDRESS":            "",
				"DATABASE_URI":           "",
			},
			flags: []string{
				"-r", "localhost:8080",
				"-a", "localhost:9000",
				"-d", "postgres://flag:password@localhost:5432/flagdb",
			},
			expectedAccrualAddress: HostAddress{
				Host: "localhost",
				Port: 8080,
			},
			expectedRunAddress: HostAddress{
				Host: "localhost",
				Port: 9000,
			},
			expectedDatabaseURI: "postgres://flag:password@localhost:5432/flagdb",
		},
		{
			name:    "Флаги с IP-адресом",
			envVars: map[string]string{},
			flags: []string{
				"-r", "192.168.1.100:8080",
				"-a", "192.168.1.100:9000",
				"-d", "postgres://user:password@192.168.1.100:5432/dbname",
			},
			expectedAccrualAddress: HostAddress{
				Host: "192.168.1.100",
				Port: 8080,
			},
			expectedRunAddress: HostAddress{
				Host: "192.168.1.100",
				Port: 9000,
			},
			expectedDatabaseURI: "postgres://user:password@192.168.1.100:5432/dbname",
		},
		{
			name: "Частичные флаги",
			envVars: map[string]string{
				"ACCRUAL_SYSTEM_ADDRESS": "localhost:8080",
				"RUN_ADDRESS":            "",
				"DATABASE_URI":           "postgres://env:password@localhost:5432/envdb",
			},
			flags: []string{
				"-a", "localhost:9000",
			},
			expectedAccrualAddress: HostAddress{
				Host: "localhost",
				Port: 8080,
			},
			expectedRunAddress: HostAddress{
				Host: "localhost",
				Port: 9000,
			},
			expectedDatabaseURI: "postgres://env:password@localhost:5432/envdb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем переменные окружения
			cleanup := setEnvVars(tt.envVars)
			defer cleanup()

			// Сохраняем и восстанавливаем аргументы
			originalArgs := saveArgs()
			defer restoreArgs(originalArgs)

			// Создаем и инициализируем конфигурацию
			config := createTestConfig()
			config.Init()

			// Устанавливаем флаги
			os.Args = append([]string{"cmd"}, tt.flags...)

			// Вызываем Parse
			config.Parse()

			// Проверяем результаты
			assertHostAddress(t, tt.expectedAccrualAddress, config.AccrualSystemAddress)
			assertHostAddress(t, tt.expectedRunAddress, config.RunAddress)

			if config.DatabaseURI != tt.expectedDatabaseURI {
				t.Errorf("Expected DatabaseURI %s, got %s", tt.expectedDatabaseURI, config.DatabaseURI)
			}
		})
	}
}

// Тесты для проверки приоритетов (переменные окружения имеют приоритет над флагами)
func TestParsePriority(t *testing.T) {
	tests := []struct {
		name                   string
		envVars                map[string]string
		flags                  []string
		expectedAccrualAddress HostAddress
		expectedRunAddress     HostAddress
		expectedDatabaseURI    string
	}{
		{
			name: "Корректные переменные окружения имеют приоритет над флагами",
			envVars: map[string]string{
				"ACCRUAL_SYSTEM_ADDRESS": "localhost:8080",
				"RUN_ADDRESS":            "localhost:9000",
				"DATABASE_URI":           "postgres://env:password@localhost:5432/envdb",
			},
			flags: []string{
				"-r", "localhost:8081",
				"-a", "localhost:9001",
				"-d", "postgres://flag:password@localhost:5433/flagdb",
			},
			expectedAccrualAddress: HostAddress{
				Host: "localhost",
				Port: 8080,
			},
			expectedRunAddress: HostAddress{
				Host: "localhost",
				Port: 9000,
			},
			expectedDatabaseURI: "postgres://env:password@localhost:5432/envdb",
		},
		{
			name: "Некорректные переменные окружения используют флаги",
			envVars: map[string]string{
				"ACCRUAL_SYSTEM_ADDRESS": "localhost:8080",
				"RUN_ADDRESS":            "invalid-address",
				"DATABASE_URI":           "",
			},
			flags: []string{
				"-r", "localhost:8081",
				"-a", "localhost:9001",
				"-d", "postgres://flag:password@localhost:5433/flagdb",
			},
			expectedAccrualAddress: HostAddress{
				Host: "localhost",
				Port: 8080,
			},
			expectedRunAddress: HostAddress{
				Host: "localhost",
				Port: 9001,
			},
			expectedDatabaseURI: "postgres://flag:password@localhost:5433/flagdb",
		},
		{
			name: "Смешанный сценарий: часть из env, часть из флагов",
			envVars: map[string]string{
				"ACCRUAL_SYSTEM_ADDRESS": "localhost:8080",
				// RUN_ADDRESS отсутствует
				"DATABASE_URI": "postgres://env:password@localhost:5432/envdb",
			},
			flags: []string{
				"-r", "localhost:8081",
				"-a", "localhost:9001",
				"-d", "postgres://flag:password@localhost:5433/flagdb",
			},
			expectedAccrualAddress: HostAddress{
				Host: "localhost",
				Port: 8080,
			},
			expectedRunAddress: HostAddress{
				Host: "localhost",
				Port: 9001,
			},
			expectedDatabaseURI: "postgres://env:password@localhost:5432/envdb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем переменные окружения
			cleanup := setEnvVars(tt.envVars)
			defer cleanup()

			// Сохраняем и восстанавливаем аргументы
			originalArgs := saveArgs()
			defer restoreArgs(originalArgs)

			// Создаем и инициализируем конфигурацию
			config := createTestConfig()
			config.Init()

			// Устанавливаем флаги
			os.Args = append([]string{"cmd"}, tt.flags...)

			// Вызываем Parse
			config.Parse()

			// Проверяем результаты
			assertHostAddress(t, tt.expectedAccrualAddress, config.AccrualSystemAddress)
			assertHostAddress(t, tt.expectedRunAddress, config.RunAddress)

			if config.DatabaseURI != tt.expectedDatabaseURI {
				t.Errorf("Expected DatabaseURI %s, got %s", tt.expectedDatabaseURI, config.DatabaseURI)
			}
		})
	}
}

// Тесты для обработки ошибок парсинга HostAddress
func TestParseHostAddressErrors(t *testing.T) {
	tests := []struct {
		name                   string
		envVars                map[string]string
		flags                  []string
		expectedAccrualAddress HostAddress
		expectedRunAddress     HostAddress
		expectedDatabaseURI    string
	}{
		{
			name: "Некорректный формат RUN_ADDRESS (без порта)",
			envVars: map[string]string{
				"ACCRUAL_SYSTEM_ADDRESS": "localhost:8080",
				"RUN_ADDRESS":            "localhost",
				"DATABASE_URI":           "postgres://user:password@localhost:5432/dbname",
			},
			flags: []string{"-a", "localhost:9000"},
			expectedAccrualAddress: HostAddress{
				Host: "localhost",
				Port: 8080,
			},
			expectedRunAddress: HostAddress{
				Host: "localhost",
				Port: 9000,
			},
			expectedDatabaseURI: "postgres://user:password@localhost:5432/dbname",
		},
		{
			name: "Некорректный формат RUN_ADDRESS (невалидный хост)",
			envVars: map[string]string{
				"ACCRUAL_SYSTEM_ADDRESS": "localhost:8080",
				"RUN_ADDRESS":            "invalid-host:9000",
				"DATABASE_URI":           "postgres://user:password@localhost:5432/dbname",
			},
			flags: []string{"-a", "localhost:9000"},
			expectedAccrualAddress: HostAddress{
				Host: "localhost",
				Port: 8080,
			},
			expectedRunAddress: HostAddress{
				Host: "localhost",
				Port: 9000,
			},
			expectedDatabaseURI: "postgres://user:password@localhost:5432/dbname",
		},
		{
			name: "Некорректный порт в RUN_ADDRESS",
			envVars: map[string]string{
				"ACCRUAL_SYSTEM_ADDRESS": "localhost:8080",
				"RUN_ADDRESS":            "localhost:invalid",
				"DATABASE_URI":           "postgres://user:password@localhost:5432/dbname",
			},
			flags: []string{"-a", "localhost:9000"},
			expectedAccrualAddress: HostAddress{
				Host: "localhost",
				Port: 8080,
			},
			expectedRunAddress: HostAddress{
				Host: "localhost",
				Port: 9000,
			},
			expectedDatabaseURI: "postgres://user:password@localhost:5432/dbname",
		},
		{
			name: "Множественные двоеточия в RUN_ADDRESS",
			envVars: map[string]string{
				"ACCRUAL_SYSTEM_ADDRESS": "localhost:8080",
				"RUN_ADDRESS":            "localhost:9000:extra",
				"DATABASE_URI":           "postgres://user:password@localhost:5432/dbname",
			},
			flags: []string{"-a", "localhost:9000"},
			expectedAccrualAddress: HostAddress{
				Host: "localhost",
				Port: 8080,
			},
			expectedRunAddress: HostAddress{
				Host: "localhost",
				Port: 9000,
			},
			expectedDatabaseURI: "postgres://user:password@localhost:5432/dbname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем переменные окружения
			cleanup := setEnvVars(tt.envVars)
			defer cleanup()

			// Сохраняем и восстанавливаем аргументы
			originalArgs := saveArgs()
			defer restoreArgs(originalArgs)

			// Создаем и инициализируем конфигурацию
			config := createTestConfig()
			config.Init()

			// Устанавливаем флаги
			os.Args = append([]string{"cmd"}, tt.flags...)

			// Вызываем Parse
			config.Parse()

			// Проверяем результаты
			assertHostAddress(t, tt.expectedAccrualAddress, config.AccrualSystemAddress)
			assertHostAddress(t, tt.expectedRunAddress, config.RunAddress)

			if config.DatabaseURI != tt.expectedDatabaseURI {
				t.Errorf("Expected DatabaseURI %s, got %s", tt.expectedDatabaseURI, config.DatabaseURI)
			}
		})
	}
}

// Тесты для обработки ошибок пустых переменных окружения
func TestParseEmptyEnvVars(t *testing.T) {
	tests := []struct {
		name                   string
		envVars                map[string]string
		flags                  []string
		expectedAccrualAddress HostAddress
		expectedRunAddress     HostAddress
		expectedDatabaseURI    string
	}{
		{
			name: "Пустая переменная ACCRUAL_SYSTEM_ADDRESS",
			envVars: map[string]string{
				"ACCRUAL_SYSTEM_ADDRESS": "",
				"RUN_ADDRESS":            "localhost:9000",
				"DATABASE_URI":           "postgres://user:password@localhost:5432/dbname",
			},
			flags: []string{"-r", "localhost:8080"},
			expectedAccrualAddress: HostAddress{
				Host: "localhost",
				Port: 8080,
			},
			expectedRunAddress: HostAddress{
				Host: "localhost",
				Port: 9000,
			},
			expectedDatabaseURI: "postgres://user:password@localhost:5432/dbname",
		},
		{
			name: "Пустая переменная DATABASE_URI",
			envVars: map[string]string{
				"ACCRUAL_SYSTEM_ADDRESS": "localhost:8080",
				"RUN_ADDRESS":            "localhost:9000",
				"DATABASE_URI":           "",
			},
			flags: []string{"-d", "postgres://flag:password@localhost:5432/flagdb"},
			expectedAccrualAddress: HostAddress{
				Host: "localhost",
				Port: 8080,
			},
			expectedRunAddress: HostAddress{
				Host: "localhost",
				Port: 9000,
			},
			expectedDatabaseURI: "postgres://flag:password@localhost:5432/flagdb",
		},
		{
			name: "Все переменные окружения пустые",
			envVars: map[string]string{
				"ACCRUAL_SYSTEM_ADDRESS": "",
				"RUN_ADDRESS":            "",
				"DATABASE_URI":           "",
			},
			flags: []string{
				"-r", "localhost:8080",
				"-a", "localhost:9000",
				"-d", "postgres://flag:password@localhost:5432/flagdb",
			},
			expectedAccrualAddress: HostAddress{
				Host: "localhost",
				Port: 8080,
			},
			expectedRunAddress: HostAddress{
				Host: "localhost",
				Port: 9000,
			},
			expectedDatabaseURI: "postgres://flag:password@localhost:5432/flagdb",
		},
		{
			name: "Частично пустые переменные окружения",
			envVars: map[string]string{
				"ACCRUAL_SYSTEM_ADDRESS": "localhost:8080",
				"RUN_ADDRESS":            "",
				"DATABASE_URI":           "",
			},
			flags: []string{
				"-a", "localhost:9000",
				"-d", "postgres://flag:password@localhost:5432/flagdb",
			},
			expectedAccrualAddress: HostAddress{
				Host: "localhost",
				Port: 8080,
			},
			expectedRunAddress: HostAddress{
				Host: "localhost",
				Port: 9000,
			},
			expectedDatabaseURI: "postgres://flag:password@localhost:5432/flagdb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем переменные окружения
			cleanup := setEnvVars(tt.envVars)
			defer cleanup()

			// Сохраняем и восстанавливаем аргументы
			originalArgs := saveArgs()
			defer restoreArgs(originalArgs)

			// Создаем и инициализируем конфигурацию
			config := createTestConfig()
			config.Init()

			// Устанавливаем флаги
			os.Args = append([]string{"cmd"}, tt.flags...)

			// Вызываем Parse
			config.Parse()

			// Проверяем результаты
			assertHostAddress(t, tt.expectedAccrualAddress, config.AccrualSystemAddress)
			assertHostAddress(t, tt.expectedRunAddress, config.RunAddress)

			if config.DatabaseURI != tt.expectedDatabaseURI {
				t.Errorf("Expected DatabaseURI %s, got %s", tt.expectedDatabaseURI, config.DatabaseURI)
			}
		})
	}
}

// Тесты для функции Parse с некорректными переменными окружения
func TestParseWithInvalidEnvVars(t *testing.T) {
	tests := []struct {
		name                   string
		envVars                map[string]string
		flags                  []string
		expectedAccrualAddress HostAddress
		expectedRunAddress     HostAddress
		expectedDatabaseURI    string
	}{
		{
			name: "Пустые переменные окружения должны использовать флаги",
			envVars: map[string]string{
				"ACCRUAL_SYSTEM_ADDRESS": "",
				"RUN_ADDRESS":            "",
				"DATABASE_URI":           "",
			},
			flags: []string{"-r", "localhost:8080", "-a", "localhost:9000", "-d", "postgres://flag:password@localhost:5432/flagdb"},
			expectedAccrualAddress: HostAddress{
				Host: "localhost",
				Port: 8080,
			},
			expectedRunAddress: HostAddress{
				Host: "localhost",
				Port: 9000,
			},
			expectedDatabaseURI: "postgres://flag:password@localhost:5432/flagdb",
		},
		{
			name: "Некорректный RUN_ADDRESS должен использовать флаг",
			envVars: map[string]string{
				"ACCRUAL_SYSTEM_ADDRESS": "localhost:8080",
				"RUN_ADDRESS":            "invalid-address",
				"DATABASE_URI":           "postgres://user:password@localhost:5432/dbname",
			},
			flags: []string{"-a", "localhost:9000"},
			expectedAccrualAddress: HostAddress{
				Host: "localhost",
				Port: 8080,
			},
			expectedRunAddress: HostAddress{
				Host: "localhost",
				Port: 9000,
			},
			expectedDatabaseURI: "postgres://user:password@localhost:5432/dbname",
		},
		{
			name: "Отсутствующие переменные окружения должны использовать флаги",
			envVars: map[string]string{
				"ACCRUAL_SYSTEM_ADDRESS": "localhost:8080",
				// RUN_ADDRESS отсутствует
				"DATABASE_URI": "postgres://user:password@localhost:5432/dbname",
			},
			flags: []string{"-a", "localhost:9000"},
			expectedAccrualAddress: HostAddress{
				Host: "localhost",
				Port: 8080,
			},
			expectedRunAddress: HostAddress{
				Host: "localhost",
				Port: 9000,
			},
			expectedDatabaseURI: "postgres://user:password@localhost:5432/dbname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем переменные окружения
			cleanup := setEnvVars(tt.envVars)
			defer cleanup()

			// Сохраняем и восстанавливаем аргументы
			originalArgs := saveArgs()
			defer restoreArgs(originalArgs)

			// Создаем и инициализируем конфигурацию
			config := createTestConfig()
			config.Init()

			// Устанавливаем флаги
			if len(tt.flags) > 0 {
				os.Args = append([]string{"cmd"}, tt.flags...)
			}

			// Вызываем Parse
			config.Parse()

			// Проверяем результаты
			assertHostAddress(t, tt.expectedAccrualAddress, config.AccrualSystemAddress)
			assertHostAddress(t, tt.expectedRunAddress, config.RunAddress)

			if config.DatabaseURI != tt.expectedDatabaseURI {
				t.Errorf("Expected DatabaseURI %s, got %s", tt.expectedDatabaseURI, config.DatabaseURI)
			}
		})
	}
}

// Бенчмарки для производительности
func BenchmarkNewServerConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewServerConfig()
	}
}

func BenchmarkInit(b *testing.B) {
	for i := 0; i < b.N; i++ {
		config := NewServerConfig()
		resetFlags()
		config.Init()
	}
}

func BenchmarkParseWithEnvVars(b *testing.B) {
	envVars := map[string]string{
		"ACCRUAL_SYSTEM_ADDRESS": "localhost:8080",
		"RUN_ADDRESS":            "localhost:9000",
		"DATABASE_URI":           "postgres://user:password@localhost:5432/dbname",
	}

	cleanup := setEnvVars(envVars)
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := createTestConfig()
		config.Init()
		config.Parse()
	}
}

// Тесты для метода GetAccrualSystemURL
func TestGetAccrualSystemURL(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*ServerConfig)
		expected string
	}{
		{
			name: "URL с localhost",
			setup: func(config *ServerConfig) {
				config.AccrualSystemAddress = HostAddress{
					Host: "localhost",
					Port: 8080,
				}
			},
			expected: "http://localhost:8080",
		},
		{
			name: "URL с IP-адресом",
			setup: func(config *ServerConfig) {
				config.AccrualSystemAddress = HostAddress{
					Host: "192.168.1.100",
					Port: 9000,
				}
			},
			expected: "http://192.168.1.100:9000",
		},
		{
			name: "URL с именем хоста",
			setup: func(config *ServerConfig) {
				config.AccrualSystemAddress = HostAddress{
					Host: "accrual-system",
					Port: 8080,
				}
			},
			expected: "http://accrual-system:8080",
		},
		{
			name: "URL с другим портом",
			setup: func(config *ServerConfig) {
				config.AccrualSystemAddress = HostAddress{
					Host: "api.accrual",
					Port: 443,
				}
			},
			expected: "http://api.accrual:443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewServerConfig()
			tt.setup(config)

			result := config.GetAccrualSystemURL()

			if result != tt.expected {
				t.Errorf("Expected URL %s, got %s", tt.expected, result)
			}
		})
	}
}

func BenchmarkParseWithFlags(b *testing.B) {
	flags := []string{
		"-r", "flag-accrual:8080",
		"-a", "localhost:9000",
		"-d", "postgres://flag:password@localhost:5432/flagdb",
	}

	originalArgs := saveArgs()
	defer restoreArgs(originalArgs)
	os.Args = append([]string{"cmd"}, flags...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := createTestConfig()
		config.Init()
		config.Parse()
	}
}

func BenchmarkParseMixed(b *testing.B) {
	envVars := map[string]string{
		"ACCRUAL_SYSTEM_ADDRESS": "env-accrual:8080",
		// RUN_ADDRESS отсутствует
		"DATABASE_URI": "postgres://env:password@localhost:5432/envdb",
	}

	flags := []string{
		"-a", "localhost:9000",
	}

	cleanup := setEnvVars(envVars)
	defer cleanup()

	originalArgs := saveArgs()
	defer restoreArgs(originalArgs)
	os.Args = append([]string{"cmd"}, flags...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := createTestConfig()
		config.Init()
		config.Parse()
	}
}
