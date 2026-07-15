package user

import (
	"context"
	"errors"
	"strings"

	"notes-of-ashen/internal/authutil"
	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/logicutil"
	"notes-of-ashen/internal/middleware"
	"notes-of-ashen/internal/mq"
	"notes-of-ashen/internal/security"
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

	currentEmail := security.NormalizeEmail(current.Email)
	req.EmailCode = strings.TrimSpace(req.EmailCode)
	email := currentEmail
	if req.Email != nil && security.NormalizeEmail(*req.Email) != "" {
		email = security.NormalizeEmail(*req.Email)
	}
	avatarURL := current.AvatarURL
	if req.AvatarURL != nil {
		avatarURL = strings.TrimSpace(*req.AvatarURL)
	}
	nickname := current.Nickname
	if req.Nickname != nil {
		nickname = strings.TrimSpace(*req.Nickname)
	}
	if err := validator.Email(email); err != nil {
		return nil, err
	}
	if nickname != "" {
		if err := validator.Length(nickname, "nickname", 1, 64); err != nil {
			return nil, err
		}
	}
	if err := validator.OptionalHTTPURL(avatarURL, "avatarUrl"); err != nil {
		return nil, err
	}
	if email != currentEmail {
		if err := security.ConsumeEmailCode(ctx, svcCtx.Redis, "update_email", email, req.EmailCode); err != nil {
			return nil, err
		}
	}

	if err := svcCtx.Store.UpdateUserProfile(ctx, userID, model.UserUpdate{
		Email:     email,
		AvatarURL: avatarURL,
		Nickname:  nickname,
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

func SendVerifyCode(ctx context.Context, svcCtx *svc.ServiceContext, req types.UserVerifyCodeReq, meta types.RequestMeta) error {
	userID, err := authutil.UserID(ctx)
	if err != nil {
		return err
	}
	current, err := svcCtx.Store.FindUserByID(ctx, userID)
	if err != nil {
		return logicutil.MapError(err)
	}
	purpose, err := security.NormalizeEmailPurpose(req.Purpose)
	if err != nil {
		return err
	}
	if purpose != "change_password" && purpose != "update_email" {
		return apperrors.BadRequest("purpose is invalid")
	}
	currentEmail := security.NormalizeEmail(current.Email)
	email := currentEmail
	if purpose == "update_email" {
		email = security.NormalizeEmail(req.Email)
		if err := validator.Required(email, "email"); err != nil {
			return err
		}
		if err := validator.Email(email); err != nil {
			return err
		}
		if email == currentEmail {
			return apperrors.BadRequest("email is unchanged")
		}
		if existing, err := svcCtx.Store.FindUserByEmail(ctx, email); err == nil && existing.ID != userID {
			return apperrors.Conflict("email already exists")
		} else if err != nil && !errors.Is(err, model.ErrNotFound) {
			return err
		}
	}
	if current.Status != "active" {
		return apperrors.Forbidden("user is disabled")
	}
	if err := security.VerifyCaptcha(ctx, svcCtx.Redis, purpose, req.CaptchaID, req.CaptchaCode); err != nil {
		return err
	}
	code, err := security.RandomDigits(6)
	if err != nil {
		return err
	}
	if err := security.StoreEmailCode(ctx, svcCtx.Redis, purpose, email, code); err != nil {
		return err
	}
	if err := svcCtx.Mailer.SendVerifyCode(ctx, email, purpose, code); err != nil {
		_ = security.ClearEmailCode(ctx, svcCtx.Redis, purpose, email)
		return err
	}
	if svcCtx.Events != nil {
		svcCtx.Events.Publish(ctx, mq.Event{
			UserID:       userID,
			EventType:    "user.verify_code_sent",
			ResourceType: "user",
			ResourceID:   userID,
			Metadata: map[string]string{
				"purpose": purpose,
			},
			IP:        meta.IP,
			UserAgent: meta.UserAgent,
		})
	}
	return nil
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
	req.EmailCode = strings.TrimSpace(req.EmailCode)
	u, err := svcCtx.Store.FindUserByID(ctx, userID)
	if err != nil {
		return logicutil.MapError(err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.OldPassword)); err != nil {
		return apperrors.BadRequest("old password is incorrect")
	}
	if err := security.ConsumeEmailCode(ctx, svcCtx.Redis, "change_password", u.Email, req.EmailCode); err != nil {
		return err
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
	// 防御性驱逐 auth 用户缓存，与 admin 改状态/角色保持一致。
	middleware.EvictAuthUserCache(ctx, svcCtx.AuthUserCache, userID)
	return nil
}
