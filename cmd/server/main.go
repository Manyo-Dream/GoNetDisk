package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/manyodream/gonetdisk/configs"
	"github.com/manyodream/gonetdisk/internal/router"
	"github.com/manyodream/gonetdisk/internal/util"
	"github.com/manyodream/gonetdisk/pkg/database"
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
		log.Fatal("加载配置文件失败:", err)
	}

	gin.SetMode(cfg.Server.Mode)

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.Charset,
		cfg.Database.ParseTime,
		cfg.Database.Loc,
	)

	db, err := database.InitDB(dsn)
	if err != nil {
		log.Fatal("初始化数据库失败:", err)
	}

	jwtManager := util.NewJWTManager(cfg.JWT.Secret, cfg.JWT.GetTokenDuration())

	// 设置路由
	r := router.SetupRouter(db, jwtManager, cfg)

	// 启动服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	log.Printf("配置文件: %s", cfgDir)
	log.Printf("存储目录: temp=%s, upload=%s", cfg.Storage.TempDir, cfg.Storage.UploadDir)
	log.Printf("运行模式: %s", cfg.Server.Mode)
	log.Printf("监听地址: %s", addr)

	if err := r.Run(addr); err != nil {
		log.Fatal("启动服务器失败:", err)
	}
}
