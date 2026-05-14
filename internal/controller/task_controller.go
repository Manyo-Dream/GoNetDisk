package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/manyodream/gonetdisk/internal/dto"
	"github.com/manyodream/gonetdisk/internal/middleware"
	"github.com/manyodream/gonetdisk/internal/service"
)

type TaskController struct {
	TaskServicer *service.TaskService
}

func NewTaskController(taskServicer *service.TaskService) *TaskController {
	return &TaskController{TaskServicer: taskServicer}
}

func (tc *TaskController) CreateTaskAndRecords(ctx *gin.Context) {
	var req dto.BatchUploadRequest

	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未认证用户"})
		return
	}

	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	resp, err := tc.TaskServicer.CreateBatchUploadTask(userID, req)
	if err != nil {
		ctx.JSON(statusFromErr(err), gin.H{
			"error": "创建任务和记录失败: " + err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

func (tc *TaskController) UploadTaskFile(ctx *gin.Context) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未认证用户"})
		return
	}

	taskID := ctx.Param("task_id")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "任务ID为空"})
		return
	}

	indexStr := ctx.PostForm("index")
	fileIndex, err := strconv.Atoi(indexStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "文件序号无效"})
		return
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "获取文件失败: " + err.Error()})
		return
	}

	err = tc.TaskServicer.UploadTaskFile(userID, taskID, fileIndex, fileHeader)
	if err != nil {
		ctx.JSON(statusFromErr(err), gin.H{
			"error": "上传文件失败: " + err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
	})
}

func (tc *TaskController) GetTaskProgress(ctx *gin.Context) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未认证用户"})
		return
	}

	taskID := ctx.Param("task_id")
	if taskID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "任务ID为空"})
		return
	}

	resp, err := tc.TaskServicer.GetTaskProgress(userID, taskID)
	if err != nil {
		ctx.JSON(statusFromErr(err), gin.H{
			"error": "查询进度失败: " + err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}
