package configs

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Redis    RedisConfig
	Minio    MinioConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Upload   UploadConfig
}

type ServerConfig struct {
	Port int
	Host string
	Mode string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type MinioConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type DatabaseConfig struct {
	Host      string
	Port      int
	User      string
	Password  string
	Name      string
	Charset   string
	ParseTime bool
	Loc       string
}

type JWTConfig struct {
	Secret             string
	AccessExpiresMin   int
	RefreshExpiresHour int
}

type UploadConfig struct {
	MaxFileSizeMB int64
}

func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	return &config, nil
}

func (jwtc *JWTConfig) GetAccessTokenDuration() time.Duration {
	return time.Duration(jwtc.AccessExpiresMin) * time.Minute
}

func (jwtc *JWTConfig) GetRefreshTokenDuration() time.Duration {
	return time.Duration(jwtc.RefreshExpiresHour) * time.Hour
}
