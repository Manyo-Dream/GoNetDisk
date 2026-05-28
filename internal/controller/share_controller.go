package controller

import (
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"

	"GoNetDisk/internal/api"
	"GoNetDisk/internal/middleware"
	"GoNetDisk/internal/service"

	"github.com/gin-gonic/gin"
)

type ShareController struct {
	ShareService *service.ShareService
}

func NewShareController(shareService *service.ShareService) *ShareController {
	return &ShareController{ShareService: shareService}
}

func (c *ShareController) CreateShare(ctx *gin.Context) {
	var req api.ShareCreateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未认证用户"})
		return
	}

	resp, err := c.ShareService.CreateShare(userID, req.UserFileID, req.Code, req.ExpireDays)
	if err != nil {
		ctx.JSON(statusFromErr(err), gin.H{"error": "创建分享失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

func (c *ShareController) ListShares(ctx *gin.Context) {
	pageStr := ctx.DefaultQuery("page", "1")
	pageSizeStr := ctx.DefaultQuery("page_size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未认证用户"})
		return
	}

	resp, err := c.ShareService.GetUserShareList(userID, page, pageSize)
	if err != nil {
		ctx.JSON(statusFromErr(err), gin.H{"error": "获取分享列表失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

func (c *ShareController) RevokeShare(ctx *gin.Context) {
	shareCode := ctx.Param("share_code")
	if shareCode == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "分享码为空"})
		return
	}

	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未认证用户"})
		return
	}

	err := c.ShareService.RevokeShare(userID, shareCode)
	if err != nil {
		ctx.JSON(statusFromErr(err), gin.H{"error": "取消分享失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
	})
}

func (c *ShareController) GetShareInfo(ctx *gin.Context) {
	shareCode := ctx.Param("share_code")
	if shareCode == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "分享码为空"})
		return
	}

	code := ctx.Query("code")

	resp, err := c.ShareService.GetShareInfo(shareCode, code)
	if err != nil {
		ctx.JSON(statusFromErr(err), gin.H{"error": "获取分享信息失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

func (c *ShareController) DownloadSharedFile(ctx *gin.Context) {
	userID, _ := middleware.GetUserID(ctx)

	shareCode := ctx.Param("share_code")
	if shareCode == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "分享码为空"})
		return
	}

	code := ctx.Query("code")

	fileMeta, file, err := c.ShareService.DownloadSharedFile(shareCode, code, userID)
	if err != nil {
		ctx.JSON(statusFromErr(err), gin.H{"error": "下载文件失败: " + err.Error()})
		return
	}
	defer file.Close()

	contentType := mime.TypeByExtension(filepath.Ext(fileMeta.FileName))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	escapedName := url.PathEscape(fileMeta.FileName)
	disposition := "attachment; filename=\"" + escapedName + "\"; filename*=utf-8''" + escapedName

	ctx.Header("Content-Disposition", disposition)
	ctx.Header("X-Content-Type-Options", "nosniff")

	ctx.DataFromReader(http.StatusOK, fileMeta.FileSize, contentType, file, nil)
}
