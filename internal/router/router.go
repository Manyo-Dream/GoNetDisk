package router

import (
	"time"
	"strings"

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

	fileRepo := repository.NewFileRepo(db)
	fileService := service.NewFileService(redis, minioClient, userRepo, fileRepo, jwtManager, config)
	fileController := controller.NewFileController(fileService)

	folderService := service.NewFolderService(redis, userRepo, fileRepo, jwtManager, minioClient, config)
	folderController := controller.NewFolderController(folderService)

	taskRepo := repository.NewTaskRepo(db)
	taskService := service.NewTaskService(userRepo, fileRepo, taskRepo, fileService, folderService, minioClient, jwtManager, config)
	taskController := controller.NewTaskController(taskService)

	chunkRepo := repository.NewChunkRepo(db)
	chunkService := service.NewChunkService(redis, minioClient, userRepo, fileRepo, chunkRepo, fileService, config)
	chunkController := controller.NewChunkController(chunkService)

	shareRepo := repository.NewShareRepo(db)
	shareService := service.NewShareService(shareRepo, fileRepo, userRepo, redis, minioClient, config)
	shareController := controller.NewShareController(shareService)

	limiter := middleware.NewIPRateLimiter(10 * time.Minute)

	v1 := r.Group("/api/v1")
	{
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

		fileHandler := v1.Group("/file")
		fileHandler.Use(middleware.AuthMiddleware(jwtManager, userRepo))
		{
			fileHandler.POST("/upload", fileController.UploadFile)
			fileHandler.POST("/chunk/init", chunkController.InitChunkUpload)
			fileHandler.POST("/chunk/upload", chunkController.UploadChunk)
			fileHandler.POST("/chunk/complete", chunkController.CompleteChunkUpload)
			fileHandler.GET("/chunk/status", chunkController.GetChunkStatus)
			fileHandler.GET("/download/:userfile_id", fileController.DownloadFile)
			fileHandler.DELETE("/delete/:userfile_id", fileController.MoveFileToTrash)
			fileHandler.DELETE("/remove/:userfile_id", fileController.RemoveFile)
			fileHandler.GET("/list", fileController.ReturnFileList)
			fileHandler.PUT("/rename", fileController.RenameFile)
			fileHandler.PUT("/move", fileController.MoveFile)
		}

		folderHandler := v1.Group("/folder")
		folderHandler.Use(middleware.AuthMiddleware(jwtManager, userRepo))
		{
			folderHandler.POST("/create", folderController.CreateFolder)
			folderHandler.DELETE("/delete/:userfolder_id", folderController.MoveFolderToTrash)
			folderHandler.DELETE("/remove/:userfolder_id", folderController.RemoveFolder)
			folderHandler.PUT("/rename", folderController.RenameFolder)
			folderHandler.PUT("/move", folderController.MoveFolder)
		}

		trashHandler := v1.Group("/trash")
		trashHandler.Use(middleware.AuthMiddleware(jwtManager, userRepo))
		{
			trashHandler.GET("/list", fileController.ReturnTrashList)
			trashHandler.POST("/file/:userfile_id", fileController.RestoreFile)
			trashHandler.POST("/folder/:userfolder_id", folderController.RestoreFolder)
		}

		taskHandler := v1.Group("/task")
		taskHandler.Use(middleware.AuthMiddleware(jwtManager, userRepo))
		{
			taskHandler.POST("/create", taskController.CreateTaskAndRecords)
			taskHandler.POST("/:task_id/file", taskController.UploadTaskFile)
			taskHandler.GET("/:task_id/progress", taskController.GetTaskProgress)
		}

		shareHandler := v1.Group("/share")
		{
			shareHandler.GET("/:share_code/info", shareController.GetShareInfo)
			shareHandler.GET("/:share_code/download", shareController.DownloadSharedFile)
		}
		shareHandler.Use(middleware.AuthMiddleware(jwtManager, userRepo))
		{
			shareHandler.POST("/create", shareController.CreateShare)
			shareHandler.GET("/list", shareController.ListShares)
			shareHandler.DELETE("/:share_code", shareController.RevokeShare)
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
