package router

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/manyodream/gonetdisk/configs"
	"github.com/manyodream/gonetdisk/internal/controller"
	"github.com/manyodream/gonetdisk/internal/middleware"
	"github.com/manyodream/gonetdisk/internal/repository"
	"github.com/manyodream/gonetdisk/internal/service"
	"github.com/manyodream/gonetdisk/internal/util"
	"github.com/minio/minio-go/v7"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

func SetupRouter(db *gorm.DB, minioClient *minio.Client, jwtManager *util.JWTManager, config *configs.Config) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.CORSMiddleware())

	userRepo := repository.NewUserRepo(db)
	userService := service.NewUserService(userRepo, jwtManager)
	userController := controller.NewUserController(userService)

	fileRepo := repository.NewFileRepo(db)
	fileService := service.NewFileService(minioClient, userRepo, fileRepo, jwtManager, config)
	fileController := controller.NewFileController(fileService)

	folderService := service.NewFolderService(userRepo, fileRepo, jwtManager, minioClient, config)
	folderController := controller.NewFolderController(folderService)

	taskRepo := repository.NewTaskRepo(db)
	taskService := service.NewTaskService(userRepo, fileRepo, taskRepo, fileService, folderService, minioClient, jwtManager, config)
	taskController := controller.NewTaskController(taskService)

	shareRepo := repository.NewShareRepo(db)
	shareService := service.NewShareService(shareRepo, fileRepo, userRepo, minioClient, config)
	shareController := controller.NewShareController(shareService)

	limiter := middleware.NewIPRateLimiter(10 * time.Minute)

	v1 := r.Group("/api/v1")
	{
		userHandler := v1.Group("/user")
		{
			userHandler.POST("/register",
				limiter.RateLimit("register", rate.Every(15*time.Minute), 3),
				userController.Register)
			userHandler.POST("/login",
				limiter.RateLimit("login", rate.Every(10*time.Minute), 3),
				userController.Login)
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
	r.Static("/css", "./front/css")
	r.Static("/js", "./front/js")
	r.GET("/", func(ctx *gin.Context) { ctx.File("./front/index.html") })
	r.NoRoute(func(ctx *gin.Context) { ctx.File("./front/index.html") })
	return r
}
