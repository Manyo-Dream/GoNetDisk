package api

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type UserInfoUpdateRequest struct {
	Username  *string `json:"username" binding:"omitempty,min=3,max=32"`
	AvatarURL *string `json:"avatar_url" binding:"omitempty"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type RegisterResponse struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Status   int    `json:"status"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Username     string `json:"username"`
	Email        string `json:"email"`
}

type RefreshResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type UserInfoGetResponse struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	AvatarUrl string `json:"avatar_url"`
}

type UserInfoUpdateResponse struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	AvatarUrl string `json:"avatar_url"`
}

type UserSpaceResponse struct {
	UsedSpace  uint64 `json:"used_space"`
	TotalSpace uint64 `json:"total_space"`
}
