package models

// HavePermissionRequest request
type HavePermissionRequest struct {
	Service  string
	Endpoint string
	Method   string
}

type AddRemoveUserProfileRequest struct {
	UserID uint `json:"userId"`
}

type ProfileCopyRequest struct {
	Name string `json:"nome"`
}

type ProfileCreateRequest struct {
	Name        string `json:"name" validate:"required"`
	Permissions []uint `json:"permissions"`
	Users       []uint `json:"users"`
}

type ProfileUpdateRequest struct {
	Name string `json:"nome" validate:"required"`
}
