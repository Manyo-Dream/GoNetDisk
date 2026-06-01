package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"GoNetDisk/internal/api"
	"GoNetDisk/internal/model"
	"GoNetDisk/internal/repository"
	"GoNetDisk/internal/util"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	userRepo   *repository.UserRepo
	jwtManager *util.JWTManager
	rdb        *redis.Client
}

func NewUserService(userRepo *repository.UserRepo, jwtManger *util.JWTManager, rdb *redis.Client) *UserService {
	return &UserService{userRepo: userRepo, jwtManager: jwtManger, rdb: rdb}
}

func (s *UserService) Register(email, username, password string) (*api.RegisterResponse, error) {
	_, err := s.userRepo.GetByEmail(email)
	if err == nil {
		return nil, util.Conflict("邮箱地址已存在")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, util.Internal(fmt.Sprintf("检查邮箱是否已存在失败: %s", err))
	}

	_, err = s.userRepo.GetByUserName(username)
	if err == nil {
		return nil, util.Conflict("用户名已存在")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, util.Internal(fmt.Sprintf("检查用户名是否已存在失败: %s", err))
	}

	err = util.ValidatePassword(password)
	if err != nil {
		return nil, util.BadRequest(fmt.Sprintf("检验用户密码失败: %s", err))
	}

	hashPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("密码加密失败: %s", err))
	}

	user := &model.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(hashPassword),
	}

	err = s.userRepo.Create(user)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("用户创建失败: %s", err))
	}

	return &api.RegisterResponse{
		Username: user.Username,
		Email:    user.Email,
		Status:   user.Status,
	}, nil
}

func (s *UserService) Login(email, password string) (*api.LoginResponse, error) {
	user, err := s.userRepo.GetByEmail(email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, util.NotFound("用户不存在")
	}
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("查询用户失败: %s", err))
	}
	if user.Status == model.UserStatusDisabled {
		return nil, util.Forbidden("用户已被禁用")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, util.Unauthorized("密码错误")
	}

	// JWT 生成 token
	userID := fmt.Sprintf("%d", user.ID)
	accessToken, err := s.jwtManager.GenerateAccessToken(fmt.Sprintf("%d", user.ID), user.Username, user.Email)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("生成 access token 失败: %s", err))
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(userID, user.Username, user.Email)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("生成 refresh token 失败: %s", err))
	}

	// 将 refresh token 写入 Redis
	ctx := context.Background()
	key := fmt.Sprintf("refresh_token:%s", userID)
	duration := time.Duration(s.jwtManager.GetRefreshTokenDuration())
	if err := s.rdb.Set(ctx, key, refreshToken, duration).Err(); err != nil {
		return nil, util.Internal(fmt.Sprintf("存储 refresh token 失败: %s", err))
	}

	return &api.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    s.jwtManager.GetAccessExpiresSeconds(),
		Username:     user.Username,
		Email:        user.Email,
	}, nil
}

func (s *UserService) RefreshToken(refreshToken string) (*api.RefreshResponse, error) {
	claims, err := s.jwtManager.VerifyToken(refreshToken)
	if err != nil {
		return nil, util.Unauthorized("refresh token 无效或已过期")
	}

	userID := claims.RegisteredClaims.Subject
	ctx := context.Background()
	key := fmt.Sprintf("refresh_token:%s", userID)

	storeapiken, err := s.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, util.Unauthorized("refresh token 已失效，请重新登录")
	}
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("查询 refresh token 失败: %s", err))
	}
	if storeapiken != refreshToken {
		return nil, util.Unauthorized("refresh token 不匹配，请重新登录")
	}

	accessToken, err := s.jwtManager.GenerateAccessToken(userID, claims.Username, claims.Email)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("生成 access token 失败 %s", err))
	}

	return &api.RefreshResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   s.jwtManager.GetAccessExpiresSeconds(),
	}, nil
}

func (s *UserService) GetUserInfo(userID uint64) (*api.UserInfoGetResponse, error) {
	user, err := s.userRepo.GetUserByID(userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, util.NotFound("用户不存在")
	}
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("查询用户失败: %s", err))
	}

	return &api.UserInfoGetResponse{
		Email:     user.Email,
		Username:  user.Username,
		AvatarUrl: user.AvatarURL,
	}, nil
}

func (s *UserService) UpdateUserInfo(userID uint64, username, avatarUrl *string) (*api.UserInfoUpdateResponse, error) {
	updates := make(map[string]any)

	if username != nil {
		updates["username"] = *username
	}

	if avatarUrl != nil {
		updates["avatar_url"] = *avatarUrl
	}

	if len(updates) == 0 {
		return nil, util.BadRequest("没有更新数据")
	}

	user, err := s.userRepo.UserInfoUpdate(userID, updates)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, util.NotFound("用户不存在")
		}
		return nil, util.Internal(fmt.Sprintf("更新用户信息失败: %s", err))
	}

	return &api.UserInfoUpdateResponse{
		Email:     user.Email,
		Username:  user.Username,
		AvatarUrl: user.AvatarURL,
	}, nil
}

func (s *UserService) GetUserSpace(userID uint64) (*api.UserSpaceResponse, error) {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, util.NotFound("用户不存在")
		}
		return nil, util.Internal(fmt.Sprintf("查询用户失败: %s", err))
	}

	return &api.UserSpaceResponse{
		UsedSpace:  user.UsedSpace,
		TotalSpace: user.TotalSpace,
	}, nil
}
