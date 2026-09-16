package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"zzdzz-blog/server/internal/service"
	"zzdzz-blog/server/pkg/response"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

type createUserReq struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=6,max=64"`
}

func (h *UserHandler) List(c *gin.Context) {
	users, err := h.svc.List()
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, users)
}

func (h *UserHandler) Create(c *gin.Context) {
	var req createUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "用户名至少 3 字符, 密码至少 6 字符")
		return
	}
	u, err := h.svc.Create(req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUsernameConflict):
			response.Fail(c, 409, 4009, "username already exists")
		case errors.Is(err, service.ErrUsernameReserved):
			response.Fail(c, 409, 4009, "username is reserved")
		default:
			response.ServerError(c, err.Error())
		}
		return
	}
	response.OK(c, u)
}

type changePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=64"`
}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	uid := userIDOf(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}
	var req changePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请输入旧密码, 新密码至少 6 位")
		return
	}
	if err := h.svc.ChangePassword(uid, id, req.OldPassword, req.NewPassword); err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			response.Fail(c, 404, 4004, "user not found")
		case errors.Is(err, service.ErrInvalidOldPassword):
			response.Fail(c, 400, 4001, "old password incorrect")
		default:
			response.ServerError(c, err.Error())
		}
		return
	}
	response.OK(c, nil)
}

type setActiveReq struct {
	IsActive *bool `json:"is_active" binding:"required"`
}

func (h *UserHandler) SetActive(c *gin.Context) {
	uid := userIDOf(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}
	var req setActiveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "缺少启用状态参数")
		return
	}
	if err := h.svc.SetActive(uid, id, *req.IsActive); err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			response.Fail(c, 404, 4004, "user not found")
		case errors.Is(err, service.ErrCannotDisableSelf):
			response.Fail(c, 400, 4001, "不能停用自己的账号")
		case errors.Is(err, service.ErrLastActiveUser):
			response.Fail(c, 400, 4001, "不能停用最后一个启用中的用户")
		default:
			response.ServerError(c, err.Error())
		}
		return
	}
	response.OK(c, nil)
}