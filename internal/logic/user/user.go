package user

import (
	"context"
	"errors"
	"strings"

	"notes-of-ashen/internal/authutil"
	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/logicutil"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/internal/validator"
	"notes-of-ashen/model"

	"golang.org/x/crypto/bcrypt"
)

func Me(ctx context.Context, svcCtx *svc.ServiceContext) (*types.UserResp, error) {
	userID, err := authutil.UserID(ctx)
	if err != nil {
		return nil, err
	}
	u, err := svcCtx.Store.FindUserByID(ctx, userID)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	resp := logicutil.UserResp(*u)
	return &resp, nil
}

func UpdateMe(ctx context.Context, svcCtx *svc.ServiceContext, req types.UpdateMeReq) (*types.UserResp, error) {
	userID, err := authutil.UserID(ctx)
	if err != nil {
		return nil, err
	}
	current, err := svcCtx.Store.FindUserByID(ctx, userID)
	if err != nil {
		return nil, logicutil.MapError(err)
	}

	req.Email = strings.TrimSpace(req.Email)
	req.AvatarURL = strings.TrimSpace(req.AvatarURL)
	req.Nickname = strings.TrimSpace(req.Nickname)
	if req.Email == "" {
		req.Email = current.Email
	}
	if err := validator.Email(req.Email); err != nil {
		return nil, err
	}
	if req.Nickname != "" {
		if err := validator.Length(req.Nickname, "nickname", 1, 64); err != nil {
			return nil, err
		}
	}
	if err := validator.OptionalHTTPURL(req.AvatarURL, "avatarUrl"); err != nil {
		return nil, err
	}

	if err := svcCtx.Store.UpdateUserProfile(ctx, userID, model.UserUpdate{
		Email:     req.Email,
		AvatarURL: req.AvatarURL,
		Nickname:  req.Nickname,
	}); err != nil {
		if logicutil.IsDuplicate(err) {
			return nil, apperrors.Conflict("email already exists")
		}
		return nil, err
	}
	updated, err := svcCtx.Store.FindUserByID(ctx, userID)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	resp := logicutil.UserResp(*updated)
	return &resp, nil
}

func ChangePassword(ctx context.Context, svcCtx *svc.ServiceContext, req types.ChangePasswordReq) error {
	userID, err := authutil.UserID(ctx)
	if err != nil {
		return err
	}
	if err := validator.Required(req.OldPassword, "oldPassword"); err != nil {
		return err
	}
	if err := validator.Length(req.NewPassword, "newPassword", 8, 128); err != nil {
		return err
	}
	u, err := svcCtx.Store.FindUserByID(ctx, userID)
	if err != nil {
		return logicutil.MapError(err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.OldPassword)); err != nil {
		return apperrors.Unauthorized("old password is incorrect")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := svcCtx.Store.UpdateUserPassword(ctx, userID, string(hash)); err != nil {
		return err
	}
	if err := svcCtx.Store.RevokeUserRefreshTokens(ctx, userID); err != nil && !errors.Is(err, model.ErrNotFound) {
		return err
	}
	return nil
}
