package controller

import (
	"net/http"

	"GoNetDisk/internal/api"
	"GoNetDisk/internal/middleware"
	"GoNetDisk/internal/service"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	UserService *service.UserService
}

func NewUserController(userService *service.UserService) *UserController {
	return &UserController{UserService: userService}
}

func (c *UserController) Register(ctx *gin.Context) {
	var req api.RegisterRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "请求参数错误:" + err.Error(),
		})
		return
	}

	resp, err := c.UserService.Register(req.Email, req.Username, req.Password)
	if err != nil {
		ctx.JSON(statusFromErr(err), gin.H{
			"error": "用户注册失败:" + err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

func (c *UserController) Login(ctx *gin.Context) {
	var req api.LoginRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "请求参数错误:" + err.Error(),
		})
		return
	}

	resp, err := c.UserService.Login(req.Email, req.Password)
	if err != nil {
		ctx.JSON(statusFromErr(err), gin.H{
			"error": "用户登录失败:" + err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

func (c *UserController) RefreshToken(ctx *gin.Context) {
	var req api.RefreshRequest

	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	resp, err := c.UserService.RefreshToken(req.RefreshToken)
	if err != nil {
		ctx.JSON(statusFromErr(err), gin.H{"error": "刷新 token 失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

func (c *UserController) GetUserInfo(ctx *gin.Context) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "未认证用户",
		})
		return
	}

	resp, err := c.UserService.GetUserInfo(userID)
	if err != nil {
		ctx.JSON(statusFromErr(err), gin.H{
			"error": "用户信息获取失败:" + err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

func (c *UserController) UpdateUserInfo(ctx *gin.Context) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "未认证用户",
		})
		return
	}

	var req api.UserInfoUpdateRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "请求参数错误:" + err.Error(),
		})
		return
	}

	resp, err := c.UserService.UpdateUserInfo(userID, req.Username, req.AvatarURL)
	if err != nil {
		ctx.JSON(statusFromErr(err), gin.H{
			"error": "用户信息更新失败:" + err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}

func (c *UserController) GetUserSpace(ctx *gin.Context) {
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "未认证用户"})
		return
	}

	resp, err := c.UserService.GetUserSpace(userID)
	if err != nil {
		ctx.JSON(statusFromErr(err), gin.H{"error": "获取空间信息失败: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": resp,
	})
}
