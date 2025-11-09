package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// EnvLoader загружает конфигурацию из файлов окружения
type EnvLoader struct {
	environment string
	configDir   string
}

// NewEnvLoader создает новый загрузчик конфигурации
func NewEnvLoader(environment string, configDir string) *EnvLoader {
	if configDir == "" {
		configDir = "./config/environments"
	}
	return &EnvLoader{
		environment: environment,
		configDir:   configDir,
	}
}

// LoadEnvironment загружает переменные окружения из файла
func (el *EnvLoader) LoadEnvironment() error {
	envFile := filepath.Join(el.configDir, fmt.Sprintf("%s.env", el.environment))
	
	// Проверяем существование файла
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		return fmt.Errorf("файл конфигурации не найден: %s", envFile)
	}
	
	// Открываем файл
	file, err := os.Open(envFile)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла конфигурации %s: %v", envFile, err)
	}
	defer file.Close()
	
	// Читаем построчно
	scanner := bufio.NewScanner(file)
	lineNum := 0
	
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		
		// Пропускаем пустые строки и комментарии
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Разбираем строку KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("некорректный формат в %s:%d: %s", envFile, lineNum, line)
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		// Обрабатываем переменные-ссылки ${VAR_NAME}
		value = el.expandVariables(value)
		
		// Устанавливаем переменную окружения только если она еще не установлена
		if os.Getenv(key) == "" {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("ошибка установки переменной %s: %v", key, err)
			}
		}
	}
	
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ошибка чтения файла %s: %v", envFile, err)
	}
	
	return nil
}

// expandVariables заменяет переменные вида ${VAR_NAME} их значениями
func (el *EnvLoader) expandVariables(value string) string {
	// Простая реализация замены переменных
	// В production среде используется более сложная логика с vault/secrets
	if strings.Contains(value, "${") && strings.Contains(value, "}") {
		// Для production окружения переменные должны быть уже установлены
		if el.environment == "production" {
			return os.ExpandEnv(value)
		}
		// Для dev/test можем предоставить значения по умолчанию
		return os.ExpandEnv(value)
	}
	return value
}

// ValidateRequiredVars проверяет наличие обязательных переменных
func (el *EnvLoader) ValidateRequiredVars(requiredVars []string) error {
	var missing []string
	
	for _, varName := range requiredVars {
		if os.Getenv(varName) == "" {
			missing = append(missing, varName)
		}
	}
	
	if len(missing) > 0 {
		return fmt.Errorf("отсутствуют обязательные переменные окружения: %s", 
			strings.Join(missing, ", "))
	}
	
	return nil
}

// GetEnvironment возвращает текущее окружение
func (el *EnvLoader) GetEnvironment() string {
	return el.environment
}

// PrintLoadedVars выводит загруженные переменные (для отладки)
func (el *EnvLoader) PrintLoadedVars(vars []string) {
	fmt.Printf("Загруженная конфигурация (%s):\n", el.environment)
	for _, varName := range vars {
		value := os.Getenv(varName)
		if strings.Contains(strings.ToLower(varName), "password") || 
		   strings.Contains(strings.ToLower(varName), "secret") ||
		   strings.Contains(strings.ToLower(varName), "token") {
			// Маскируем чувствительные данные
			if len(value) > 4 {
				value = value[:2] + "****" + value[len(value)-2:]
			} else {
				value = "****"
			}
		}
		fmt.Printf("  %s=%s\n", varName, value)
	}
}

// GetEnvWithDefault возвращает значение переменной окружения или значение по умолчанию
func GetEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetIntEnv возвращает int значение переменной окружения или значение по умолчанию
func GetIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetBoolEnv возвращает bool значение переменной окружения или значение по умолчанию
func GetBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		switch strings.ToLower(value) {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		}
	}
	return defaultValue
}
