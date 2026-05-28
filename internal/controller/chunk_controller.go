package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"GoNetDisk/internal/api"
	"GoNetDisk/internal/middleware"
	"GoNetDisk/internal/service"
)

type ChunkController struct {
	chunkService *service.ChunkService
}

func NewChunkController(chunkService *service.ChunkService) *ChunkController {
	return &ChunkController{chunkService: chunkService}
}

func (c *ChunkController) InitChunkUpload(ctx *gin.Context) {
	var req api.ChunkInitRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未认证用户"})
		return
	}

	resp, err := c.chunkService.InitChunkUpload(userID, &req)
	if err != nil {
		ctx.JSON(statusFromErr(err), gin.H{"error": "初始化分片上传失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

func (c *ChunkController) UploadChunk(ctx *gin.Context) {
	uploadID := ctx.PostForm("upload_id")
	chunkIndexStr := ctx.PostForm("chunk_index")
	chunkHash := ctx.PostForm("chunk_hash")

	if uploadID == "" || chunkIndexStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "upload_id 和 chunk_index 不能为空"})
		return
	}

	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "chunk_index 格式错误"})
		return
	}

	chunkFile, err := ctx.FormFile("chunk")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "获取分片文件失败: " + err.Error()})
		return
	}

	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未认证用户"})
		return
	}

	req := &api.ChunkUploadRequest{
		UploadID:   uploadID,
		ChunkIndex: chunkIndex,
		ChunkHash:  chunkHash,
	}

	resp, err := c.chunkService.UploadChunk(userID, req, chunkFile)
	if err != nil {
		ctx.JSON(statusFromErr(err), gin.H{"error": "上传分片失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

func (c *ChunkController) CompleteChunkUpload(ctx *gin.Context) {
	var req api.ChunkCompleteRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未认证用户"})
		return
	}

	resp, err := c.chunkService.CompleteChunkUpload(userID, &req)
	if err != nil {
		ctx.JSON(statusFromErr(err), gin.H{"error": "合并分片失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

func (c *ChunkController) GetChunkStatus(ctx *gin.Context) {
	var req api.ChunkStatusRequest

	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未认证用户"})
		return
	}

	resp, err := c.chunkService.GetChunkStatus(userID, &req)
	if err != nil {
		ctx.JSON(statusFromErr(err), gin.H{"error": "查询分片状态失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}
