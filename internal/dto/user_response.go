package dto

type RegisterResponse struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Status   int    `json:"status"`
}

type LoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type UserInfoGetResponse struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	AvatarUrl string `json:"avatar_url"`
}

type UserInfoUpdateResponse struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	AvatarUrl string `json:"avatar_url"`
}

type UserSpaceResponse struct {
	UsedSpace  uint64 `json:"used_space"`
	TotalSpace uint64 `json:"total_space"`
}
