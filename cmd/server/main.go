package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"GoNetDisk/configs"
	"GoNetDisk/internal/router"
	"GoNetDisk/internal/util"
	"GoNetDisk/pkg/database"
	"GoNetDisk/pkg/storage"

	"github.com/gin-gonic/gin"
)

func getConfigPath() string {
	// 1. 环境变量优先
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" && fileExists(envPath) {
		log.Printf("使用环境变量配置: %s", envPath)
		return envPath
	}

	// 2. 搜索可能的路径
	searchPaths := []string{
		// 基于当前工作目录
		"./configs/config.yaml",
		"../configs/config.yaml",
		"../../configs/config.yaml",

		// 基于可执行文件目录
		getExecutableRelativePath("configs/config.yaml"),
		getExecutableRelativePath("../configs/config.yaml"),

		// 项目的可能位置
		"./config/config.yaml",
		"./conf/config.yaml",
	}

	for _, path := range searchPaths {
		if fileExists(path) {
			log.Printf("找到配置文件: %s", path)
			return path
		}
	}

	// 3. 返回默认路径
	defaultPath := "./configs/config.yaml"
	log.Printf("警告: 未找到配置文件，使用默认路径: %s", defaultPath)
	return defaultPath
}

// 辅助函数：检查文件是否存在
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// 辅助函数：获取相对于可执行文件的路径
func getExecutableRelativePath(relativePath string) string {
	exePath, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(exePath), relativePath)
}

func main() {
	cfgDir := getConfigPath()

	cfg, err := configs.LoadConfig(cfgDir)

	if err != nil {
		log.Fatalf("加载配置文件失败: %s", err)
	}

	gin.SetMode(cfg.Server.Mode)

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
	)

	db, err := database.InitDB(dsn, &cfg.Database)
	if err != nil {
		log.Fatal("初始化 Postgres 失败:", err)
	}

	rdb, err := database.InitRedis(
		cfg.Redis.Host,
		cfg.Redis.Port,
		cfg.Redis.Password,
		cfg.Redis.DB,
	)
	if err != nil {
		log.Fatal("初始化 Redis 失败:", err)
	}

	minioClient, err := storage.NewMinioClient(cfg.Minio)
	if err != nil {
		log.Fatalf("初始化 Minio 失败: %s", err)
	}

	jwtManager := util.NewJWTManager(
		cfg.JWT.Secret,
		cfg.JWT.GetAccessTokenDuration(),
		cfg.JWT.GetRefreshTokenDuration(),
	)

	// 设置路由
	r := router.SetupRouter(db, rdb, minioClient, jwtManager, cfg)

	// 启动服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	log.Printf("配置文件路径: %s", cfgDir)
	log.Printf("Postgres 端点: %d", cfg.Database.Port)
	log.Printf("Redis 端点: %d", cfg.Redis.Port)
	log.Printf("MinIO 端点: %s, Bucket: %s", cfg.Minio.Endpoint, cfg.Minio.Bucket)
	log.Printf("运行模式: %s", cfg.Server.Mode)
	log.Printf("监听地址: %s", addr)

	if err := r.Run(addr); err != nil {
		log.Fatal("启动服务器失败:", err)
	}
}
