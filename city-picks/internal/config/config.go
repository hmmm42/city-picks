package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hmmm42/city-picks/pkg/logger"
	"github.com/spf13/viper"
)

var (
	ServerOptions *ServerSetting
	MySQLOptions  *MySQLSetting
	LogOptions    *logger.LogSettings
)

type ServerSetting struct {
	RunMode      string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type MySQLSetting struct {
	User         string
	Password     string
	Host         string
	Port         string
	DBName       string
	MaxIdleConns int
	MaxOpenConns int
}

//func init() {
//	InitConfig()
//}

var once sync.Once

func InitConfig(path string) {
	once.Do(func() {
		if err := readConfigFile(path); err != nil {
			panic("Failed to read config file: " + err.Error())
		}

		if err := ReadSection("server", &ServerOptions); err != nil {
			panic("Failed to read server config: " + err.Error())
		}
		if err := ReadSection("mysql", &MySQLOptions); err != nil {
			panic("Failed to read MySQL config: " + err.Error())
		}
		if err := ReadSection("log", &LogOptions); err != nil {
			panic("Failed to read log config: " + err.Error())
		}
	})
}

func readConfigFile(path string) error {
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		_ = reloadAllSections()
		level := viper.GetString("LogLevel")
		logger.LogLevel.Set(logger.GetLogLevel(level))
	})
	return nil
}

var sections = make(map[string]any)

func ReadSection(key string, v any) error {
	err := viper.UnmarshalKey(key, v)
	if err != nil {
		return err
	}

	if _, ok := sections[key]; !ok {
		sections[key] = v
	}
	return nil
}

func reloadAllSections() error {
	for key, v := range sections {
		if err := ReadSection(key, v); err != nil {
			return err
		}
	}
	return nil
}

// 定义配置文件的相对路径
const configFileName = "config.yaml"
const configDirPath = "../configs" // 配置文件所在的目录名，相对于项目根目录
func GetDefaultConfigPath() string {
	projectRoot, err := getProjectRoot()
	if err != nil {
		// 如果无法获取项目根目录，可以返回一个空字符串或一个已知的回退路径
		// 或者直接 panic，因为这是核心功能
		panic(err)
	}
	return filepath.Join(projectRoot, configDirPath, configFileName)
}

func getProjectRoot() (string, error) {
	// 1. 优先检查当前工作目录 (go run 场景)
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}
	currentDir := wd
	for {
		goModPath := filepath.Join(currentDir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return currentDir, nil
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break // 已经到达根目录
		}
		currentDir = parentDir
	}

	// 2. 其次检查可执行文件路径 (编译后的二进制文件场景)
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	exeDir := filepath.Dir(exePath)
	currentDir = exeDir
	for {
		goModPath := filepath.Join(currentDir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return currentDir, nil
		}
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break // 已经到达根目录
		}
		currentDir = parentDir
	}

	// 3. 都没找到，返回错误
	return "", fmt.Errorf("go.mod not found in any parent directory of %s or %s", exeDir, wd)
}
