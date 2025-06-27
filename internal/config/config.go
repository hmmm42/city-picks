package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/hmmm42/city-picks/pkg/logger"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	ServerOptions *ServerSetting
	MySQLOptions  *MySQLSetting
	RedisOptions  *RedisSetting
	LogOptions    *logger.LogSettings
	JWTOptions    *JWTSetting
)

type Options struct {
	Server *ServerSetting
	MySQL  *MySQLSetting
	Redis  *RedisSetting
	Log    *logger.LogSettings
	JWT    *JWTSetting
}

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

type RedisSetting struct {
	Host     string
	Port     string
	PoolSize int
}

type JWTSetting struct {
	Secret string
	Issuer string
	Expire time.Duration
}

func NewOptions() (*Options, error) {
	// 使用 pflag 读取命令行参数中的配置文件路径
	configPath := pflag.StringP("config", "c", GetDefaultConfigPath(), "path to config file")
	pflag.Parse()

	vp := viper.New()
	vp.SetConfigFile(*configPath)

	// 绑定环境变量，特别是JWT Secret
	_ = vp.BindEnv("jwt.secret", "JWT_SECRET")
	vp.AutomaticEnv()

	if err := vp.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var opts Options
	if err := vp.Unmarshal(&opts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 将读取到的配置赋值给全局变量
	ServerOptions = opts.Server
	MySQLOptions = opts.MySQL
	RedisOptions = opts.Redis
	LogOptions = opts.Log
	JWTOptions = opts.JWT

	// 配置热更新逻辑
	vp.WatchConfig()
	vp.OnConfigChange(func(in fsnotify.Event) {
		slog.Info("Config file changed:", "event", in.Name)
		// 重新加载所有配置到全局变量
		var updatedOpts Options
		if err := vp.Unmarshal(&updatedOpts); err != nil {
			slog.Error("Failed to re-unmarshal config on change:", "err", err)
			return
		}
		ServerOptions = updatedOpts.Server
		MySQLOptions = updatedOpts.MySQL
		RedisOptions = updatedOpts.Redis
		LogOptions = updatedOpts.Log
		JWTOptions = updatedOpts.JWT

		// 特别处理日志级别热更新
		if newLevel := vp.GetString("log.level"); newLevel != "" {
			logger.LogLevel.Set(logger.GetLogLevel(newLevel))
		}
	})

	return &opts, nil
}

//var once sync.Once
//
//func InitConfig(path string) {
//	once.Do(func() {
//		if err := readConfigFile(path); err != nil {
//			panic("Failed to read config file: " + err.Error())
//		}
//
//		initSettings(map[string]any{
//			"server": &ServerOptions,
//			"mysql":  &MySQLOptions,
//			"redis":  &RedisOptions,
//			"log":    &LogOptions,
//			"jwt":    &JWTOptions,
//		})
//		JWTOptions.Secret = viper.Get("jwt.Secret").(string)
//
//		logger.InitLogger(LogOptions)
//	})
//}
//
//func readConfigFile(path string) error {
//	viper.SetConfigFile(path)
//	_ = viper.BindEnv("jwt.Secret", "JWT_SECRET")
//	viper.AutomaticEnv()
//	if err := viper.ReadInConfig(); err != nil {
//		return err
//	}
//	viper.WatchConfig()
//	// 热更新: 用于更新日志等级
//	viper.OnConfigChange(func(in fsnotify.Event) {
//		_ = reloadAllSections()
//		level := viper.GetString("LogLevel")
//		logger.LogLevel.Set(logger.GetLogLevel(level))
//	})
//	return nil
//}
//
//var sections = make(map[string]any)
//
//func initSettings(settings map[string]any) {
//	for key, setting := range settings {
//		if err := ReadSection(key, setting); err != nil {
//			panic(fmt.Sprintf("Failed to read %s config: %s", key, err.Error()))
//		}
//	}
//}
//
//func ReadSection(key string, v any) error {
//	err := viper.UnmarshalKey(key, v)
//	if err != nil {
//		return err
//	}
//
//	if _, ok := sections[key]; !ok {
//		sections[key] = v
//	}
//	return nil
//}
//
//func reloadAllSections() error {
//	for key, v := range sections {
//		if err := ReadSection(key, v); err != nil {
//			return err
//		}
//	}
//	return nil
//}

// 定义配置文件的相对路径
const configFileName = "config.yaml"
const configDirPath = "./configs" // 配置文件所在的目录名，相对于项目根目录
func GetDefaultConfigPath() string {
	projectRoot, err := getProjectRoot()
	if err != nil {
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
