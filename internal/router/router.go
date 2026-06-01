package router

import (
	"strings"
	"time"

	"GoNetDisk/configs"
	"GoNetDisk/internal/controller"
	"GoNetDisk/internal/middleware"
	"GoNetDisk/internal/repository"
	"GoNetDisk/internal/service"
	"GoNetDisk/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

func SetupRouter(db *gorm.DB, redis *redis.Client, minioClient *minio.Client, jwtManager *util.JWTManager, config *configs.Config) *gin.Engine {
	r := gin.Default()

	userRepo := repository.NewUserRepo(db)
	userService := service.NewUserService(userRepo, jwtManager, redis)
	userController := controller.NewUserController(userService)

	txManager := repository.NewTxManager(db)

	fileRepo := repository.NewFileRepo(db)
	fileService := service.NewFileService(redis, minioClient, userRepo, fileRepo, jwtManager, config, txManager)
	fileController := controller.NewFileController(fileService)

	folderService := service.NewFolderService(redis, userRepo, fileRepo, jwtManager, minioClient, config, txManager)
	folderController := controller.NewFolderController(folderService)

	taskRepo := repository.NewTaskRepo(db)
	taskService := service.NewTaskService(userRepo, fileRepo, taskRepo, fileService, folderService, minioClient, jwtManager, config, txManager)
	taskController := controller.NewTaskController(taskService)

	chunkRepo := repository.NewChunkRepo(db)
	chunkService := service.NewChunkService(redis, minioClient, userRepo, fileRepo, chunkRepo, fileService, config, txManager)
	chunkController := controller.NewChunkController(chunkService)

	shareRepo := repository.NewShareRepo(db)
	shareService := service.NewShareService(shareRepo, fileRepo, userRepo, redis, minioClient, config)
	shareController := controller.NewShareController(shareService)

	limiter := middleware.NewIPRateLimiter(10 * time.Minute)

	v1 := r.Group("/api/v1")
	{
		// ── 用户资源 ──
		userHandler := v1.Group("/user")
		{
			userHandler.POST("/register",
				limiter.RateLimit("register", rate.Every(5*time.Minute), 5),
				userController.Register)
			userHandler.POST("/login",
				limiter.RateLimit("login", rate.Every(5*time.Minute), 5),
				userController.Login)
			userHandler.POST("/refresh",
				limiter.RateLimit("refresh", rate.Every(time.Minute), 5),
				userController.RefreshToken)
		}
		userHandler.Use(middleware.AuthMiddleware(jwtManager, userRepo))
		{
			userHandler.GET("/info", userController.GetUserInfo)
			userHandler.PUT("/info", userController.UpdateUserInfo)
			userHandler.GET("/space", userController.GetUserSpace)
		}

		// ── 文件资源 ──
		files := v1.Group("/files")
		files.Use(middleware.AuthMiddleware(jwtManager, userRepo))
		{
			files.POST("", fileController.UploadFile)
			files.GET("", fileController.ReturnFileList)
			files.GET("/:userfile_id", fileController.DownloadFile)
			files.PUT("/:userfile_id", fileController.RenameFile)
			files.PATCH("/:userfile_id", fileController.MoveFile)
			files.DELETE("/:userfile_id", fileController.MoveFileToTrash)

			// 分片上传
			files.POST("/chunks", chunkController.InitChunkUpload)
			files.PUT("/chunks", chunkController.UploadChunk)
			files.POST("/chunks/complete", chunkController.CompleteChunkUpload)
			files.GET("/chunks/status", chunkController.GetChunkStatus)
		}

		// ── 文件夹资源 ──
		folders := v1.Group("/folders")
		folders.Use(middleware.AuthMiddleware(jwtManager, userRepo))
		{
			folders.POST("", folderController.CreateFolder)
			folders.PUT("/:userfolder_id", folderController.RenameFolder)
			folders.PATCH("/:userfolder_id", folderController.MoveFolder)
			folders.DELETE("/:userfolder_id", folderController.MoveFolderToTrash)
		}

		// ── 回收站 ──
		trash := v1.Group("/trash")
		trash.Use(middleware.AuthMiddleware(jwtManager, userRepo))
		{
			trash.GET("", fileController.ReturnTrashList)
			trash.DELETE("/files/:userfile_id", fileController.RemoveFile)
			trash.DELETE("/folders/:userfolder_id", folderController.RemoveFolder)
			trash.POST("/files/:userfile_id/restore", fileController.RestoreFile)
			trash.POST("/folders/:userfolder_id/restore", folderController.RestoreFolder)
		}

		// ── 上传任务 ──
		tasks := v1.Group("/tasks")
		tasks.Use(middleware.AuthMiddleware(jwtManager, userRepo))
		{
			tasks.POST("", taskController.CreateTaskAndRecords)
			tasks.POST("/:task_id/files", taskController.UploadTaskFile)
			tasks.GET("/:task_id", taskController.GetTaskProgress)
		}

		// ── 分享 ──
		shares := v1.Group("/shares")
		{
			shares.GET("/:share_code", shareController.GetShareInfo)
			shares.GET("/:share_code/download", shareController.DownloadSharedFile)
		}
		shares.Use(middleware.AuthMiddleware(jwtManager, userRepo))
		{
			shares.POST("", shareController.CreateShare)
			shares.GET("", shareController.ListShares)
			shares.DELETE("/:share_code", shareController.RevokeShare)
		}
	}
	r.Static("/assets", "./front/dist/assets")
	r.StaticFile("/favicon.ico", "./front/dist/favicon.ico")
	r.GET("/", func(ctx *gin.Context) { ctx.File("./front/dist/index.html") })
	r.NoRoute(func(ctx *gin.Context) {
		if !strings.HasPrefix(ctx.Request.URL.Path, "/api") {
			ctx.File("./front/dist/index.html")
		}
	})
	return r
}
