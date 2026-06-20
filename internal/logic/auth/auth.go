package auth

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"notes-of-ashen/internal/authutil"
	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/logicutil"
	"notes-of-ashen/internal/mq"
	"notes-of-ashen/internal/security"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/internal/validator"
	"notes-of-ashen/model"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/crypto/bcrypt"
)

const refreshPrefix = "notes-of-ashen:refresh:"

func Captcha(ctx context.Context, svcCtx *svc.ServiceContext, req types.CaptchaReq) (*types.CaptchaResp, error) {
	req.Purpose = trim(req.Purpose)
	if req.Purpose == "" {
		req.Purpose = "login"
	}
	challenge, err := security.NewCaptcha(ctx, svcCtx.Redis, req.Purpose)
	if err != nil {
		return nil, err
	}
	return &types.CaptchaResp{
		CaptchaID: challenge.ID,
		ImageData: challenge.ImageData,
		ExpiresIn: challenge.ExpiresIn,
	}, nil
}

func SendVerifyCode(ctx context.Context, svcCtx *svc.ServiceContext, req types.SendVerifyCodeReq, meta types.RequestMeta) error {
	req.Email = security.NormalizeEmail(req.Email)
	req.Purpose = trim(req.Purpose)
	req.CaptchaID = trim(req.CaptchaID)
	req.CaptchaCode = trim(req.CaptchaCode)
	if err := validator.Required(req.Email, "email"); err != nil {
		return err
	}
	if err := validator.Email(req.Email); err != nil {
		return err
	}
	purpose, err := security.NormalizeEmailPurpose(req.Purpose)
	if err != nil {
		return err
	}
	if purpose != "register" && purpose != "reset_password" {
		return apperrors.BadRequest("purpose is invalid")
	}
	if err := checkPublicVerifyCodePurpose(ctx, svcCtx, purpose, req.Email); err != nil {
		return err
	}
	if err := security.VerifyCaptcha(ctx, svcCtx.Redis, purpose, req.CaptchaID, req.CaptchaCode); err != nil {
		return err
	}

	code, err := security.RandomDigits(6)
	if err != nil {
		return err
	}
	if err := security.StoreEmailCode(ctx, svcCtx.Redis, purpose, req.Email, code); err != nil {
		return err
	}
	if err := svcCtx.Mailer.SendVerifyCode(ctx, req.Email, purpose, code); err != nil {
		_ = security.ClearEmailCode(ctx, svcCtx.Redis, purpose, req.Email)
		return err
	}
	publishEvent(ctx, svcCtx, mq.Event{
		EventType:    "auth.verify_code_sent",
		ResourceType: "email",
		Metadata: map[string]string{
			"purpose": purpose,
		},
		IP:        meta.IP,
		UserAgent: meta.UserAgent,
	})
	return nil
}

func Register(ctx context.Context, svcCtx *svc.ServiceContext, req types.RegisterReq, meta types.RequestMeta) (*types.TokenPair, error) {
	req.Account = trim(req.Account)
	req.Email = security.NormalizeEmail(req.Email)
	req.Nickname = trim(req.Nickname)
	req.AvatarURL = trim(req.AvatarURL)
	req.EmailCode = trim(req.EmailCode)
	if err := validateRegister(req); err != nil {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	var id uint64
	var role string
	if err := svcCtx.Store.WithUserRegistrationLock(ctx, func(ctx context.Context) error {
		total, err := svcCtx.Store.CountUsers(ctx)
		if err != nil {
			return err
		}
		isFirstUser := total == 0
		role = "user"
		if isFirstUser {
			role = "admin"
		} else {
			settings, err := svcCtx.Store.SiteSettings(ctx)
			if err != nil {
				return err
			}
			if !settings.RegistrationEnabled {
				return apperrors.Forbidden("registration is disabled")
			}
		}
		if _, err := svcCtx.Store.FindUserByAccount(ctx, req.Account); err == nil {
			return apperrors.Conflict("account or email already exists")
		} else if !errors.Is(err, model.ErrNotFound) {
			return err
		}
		if _, err := svcCtx.Store.FindUserByEmail(ctx, req.Email); err == nil {
			return apperrors.Conflict("account or email already exists")
		} else if !errors.Is(err, model.ErrNotFound) {
			return err
		}
		if logicutil.RegistrationEmailCodeRequired(isFirstUser, svcCtx.Config.Email.Enabled) {
			if err := security.ConsumeEmailCode(ctx, svcCtx.Redis, "register", req.Email, req.EmailCode); err != nil {
				return err
			}
		}
		createdID, err := svcCtx.Store.CreateUser(ctx, model.UserCreate{
			Account:      req.Account,
			PasswordHash: string(passwordHash),
			Email:        req.Email,
			AvatarURL:    req.AvatarURL,
			Nickname:     req.Nickname,
			Role:         role,
		})
		if err != nil {
			if logicutil.IsDuplicate(err) {
				return apperrors.Conflict("account or email already exists")
			}
			return err
		}
		id = createdID
		return nil
	}); err != nil {
		return nil, err
	}

	pair, err := issueTokens(ctx, svcCtx, id, role)
	if err != nil {
		return nil, err
	}
	publishEvent(ctx, svcCtx, mq.Event{
		UserID:       id,
		EventType:    "user.registered",
		ResourceType: "user",
		ResourceID:   id,
		IP:           meta.IP,
		UserAgent:    meta.UserAgent,
	})
	return pair, nil
}

func Login(ctx context.Context, svcCtx *svc.ServiceContext, req types.LoginReq, meta types.RequestMeta) (*types.TokenPair, error) {
	req.Account = trim(req.Account)
	req.CaptchaID = trim(req.CaptchaID)
	req.CaptchaCode = trim(req.CaptchaCode)
	if err := validator.Required(req.Account, "account"); err != nil {
		return nil, err
	}
	if err := validator.Required(req.Password, "password"); err != nil {
		return nil, err
	}
	if err := security.VerifyCaptcha(ctx, svcCtx.Redis, "login", req.CaptchaID, req.CaptchaCode); err != nil {
		return nil, err
	}

	user, err := svcCtx.Store.FindUserByAccountOrEmail(ctx, req.Account)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, apperrors.Unauthorized("account or password is incorrect")
		}
		return nil, err
	}
	if user.Status != "active" {
		return nil, apperrors.Forbidden("user is disabled")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, apperrors.Unauthorized("account or password is incorrect")
	}

	pair, err := issueTokens(ctx, svcCtx, user.ID, user.Role)
	if err != nil {
		return nil, err
	}
	publishEvent(ctx, svcCtx, mq.Event{
		UserID:       user.ID,
		EventType:    "user.logged_in",
		ResourceType: "user",
		ResourceID:   user.ID,
		IP:           meta.IP,
		UserAgent:    meta.UserAgent,
	})
	return pair, nil
}

func ResetPassword(ctx context.Context, svcCtx *svc.ServiceContext, req types.ResetPasswordReq, meta types.RequestMeta) error {
	req.Email = security.NormalizeEmail(req.Email)
	req.EmailCode = trim(req.EmailCode)
	if err := validator.Required(req.Email, "email"); err != nil {
		return err
	}
	if err := validator.Email(req.Email); err != nil {
		return err
	}
	if err := validator.Length(req.NewPassword, "newPassword", 8, 128); err != nil {
		return err
	}

	user, err := svcCtx.Store.FindUserByEmail(ctx, req.Email)
	if err != nil {
		return logicutil.MapError(err)
	}
	if user.Status != "active" {
		return apperrors.Forbidden("user is disabled")
	}
	if err := security.ConsumeEmailCode(ctx, svcCtx.Redis, "reset_password", req.Email, req.EmailCode); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := svcCtx.Store.UpdateUserPassword(ctx, user.ID, string(hash)); err != nil {
		return err
	}
	if err := svcCtx.Store.RevokeUserRefreshTokens(ctx, user.ID); err != nil && !errors.Is(err, model.ErrNotFound) {
		return err
	}
	publishEvent(ctx, svcCtx, mq.Event{
		UserID:       user.ID,
		EventType:    "user.password_reset",
		ResourceType: "user",
		ResourceID:   user.ID,
		IP:           meta.IP,
		UserAgent:    meta.UserAgent,
	})
	return nil
}

func Refresh(ctx context.Context, svcCtx *svc.ServiceContext, req types.RefreshReq) (*types.TokenPair, error) {
	req.RefreshToken = trim(req.RefreshToken)
	if req.RefreshToken == "" {
		// refreshToken 通常由 HttpOnly Cookie 携带；缺失表示未登录或 Cookie 失效。
		return nil, apperrors.Unauthorized("refresh token is required")
	}
	hash := authutil.HashRefreshToken(req.RefreshToken)
	// Redis 仅作缓存，DB 为准。缓存缺失或出错时以 DB 为准并继续后续校验，
	// 避免 Redis 抖动直接 500 导致刷新失败；token 旋转成功后由 issueTokens 写入新缓存。
	token, err := svcCtx.Store.FindRefreshToken(ctx, hash)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	if token.RevokedAt != nil || time.Now().After(token.ExpiresAt) {
		return nil, apperrors.Unauthorized("refresh token is expired")
	}
	// 校验 Redis 缓存中的 userID 是否与 DB 一致；缓存缺失或非 Nil 错误降级为 miss（不 fail-closed）。
	if cachedUserID, err := svcCtx.Redis.Get(ctx, refreshKey(hash)).Result(); err == nil {
		redisUserID, parseErr := strconv.ParseUint(cachedUserID, 10, 64)
		if parseErr != nil || redisUserID != token.UserID {
			return nil, apperrors.Unauthorized("refresh token is invalid")
		}
	} else if !errors.Is(err, redis.Nil) {
		logx.Errorf("refresh token cache read failed, fallback to db: %v", err)
	}
	user, err := svcCtx.Store.FindUserByID(ctx, token.UserID)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	if user.Status != "active" {
		return nil, apperrors.Forbidden("user is disabled")
	}

	if err := svcCtx.Store.RevokeRefreshToken(ctx, hash); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, apperrors.Unauthorized("refresh token is invalid")
		}
		return nil, err
	}
	// token 已在 DB 撤销，缓存残留由 TTL 自然过期，不影响安全；删除失败仅记 warn。
	if err := svcCtx.Redis.Del(ctx, refreshKey(hash)).Err(); err != nil && !errors.Is(err, redis.Nil) {
		logx.Errorf("refresh token cache delete failed after revoke: %v", err)
	}
	return issueTokens(ctx, svcCtx, user.ID, user.Role)
}

func Logout(ctx context.Context, svcCtx *svc.ServiceContext, req types.RefreshReq, meta types.RequestMeta) error {
	userID, err := authutil.UserID(ctx)
	if err != nil {
		return err
	}
	req.RefreshToken = trim(req.RefreshToken)
	// refreshToken 现由 HttpOnly Cookie 携带；缺失时视为已登出，幂等返回成功，
	// 避免前端在 Cookie 过期/清理后登出反复失败。
	if req.RefreshToken == "" {
		return nil
	}
	hash := authutil.HashRefreshToken(req.RefreshToken)
	token, err := svcCtx.Store.FindRefreshToken(ctx, hash)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return apperrors.Unauthorized("refresh token is invalid")
		}
		return err
	}
	if err := validateLogoutRefreshToken(token, userID, time.Now()); err != nil {
		return err
	}
	if err := svcCtx.Store.RevokeRefreshTokenForUser(ctx, hash, userID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return apperrors.Unauthorized("refresh token is invalid")
		}
		return err
	}
	if err := svcCtx.Redis.Del(ctx, refreshKey(hash)).Err(); err != nil && !errors.Is(err, redis.Nil) {
		logx.Errorf("refresh token cache delete failed after logout revoke: %v", err)
	}
	publishEvent(ctx, svcCtx, mq.Event{
		UserID:       userID,
		EventType:    "user.logged_out",
		ResourceType: "user",
		ResourceID:   userID,
		IP:           meta.IP,
		UserAgent:    meta.UserAgent,
	})
	return nil
}

func issueTokens(ctx context.Context, svcCtx *svc.ServiceContext, userID uint64, role string) (*types.TokenPair, error) {
	accessToken, err := svcCtx.Tokens.CreateAccessToken(userID, role)
	if err != nil {
		return nil, err
	}
	refreshToken, refreshHash, expiresAt, err := svcCtx.Tokens.CreateRefreshToken()
	if err != nil {
		return nil, err
	}
	if err := svcCtx.Store.CreateRefreshToken(ctx, userID, refreshHash, expiresAt); err != nil {
		return nil, err
	}
	if err := svcCtx.Redis.Set(ctx, refreshKey(refreshHash), strconv.FormatUint(userID, 10), svcCtx.Tokens.RefreshTTL()).Err(); err != nil {
		revokeRefreshTokenBestEffort(svcCtx, refreshHash)
		return nil, err
	}
	return &types.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    svcCtx.Tokens.AccessExpiresIn(),
	}, nil
}

func validateRegister(req types.RegisterReq) error {
	if err := validator.Length(req.Account, "account", 3, 64); err != nil {
		return err
	}
	if err := validator.Length(req.Password, "password", 8, 128); err != nil {
		return err
	}
	if err := validator.Required(req.Email, "email"); err != nil {
		return err
	}
	if err := validator.Email(req.Email); err != nil {
		return err
	}
	if req.Nickname != "" {
		if err := validator.Length(req.Nickname, "nickname", 1, 64); err != nil {
			return err
		}
	}
	if err := validator.OptionalHTTPURL(req.AvatarURL, "avatarUrl"); err != nil {
		return err
	}
	return nil
}

func revokeRefreshTokenBestEffort(svcCtx *svc.ServiceContext, refreshHash string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = svcCtx.Store.RevokeRefreshToken(ctx, refreshHash)
}

func validateLogoutRefreshToken(token *model.RefreshToken, userID uint64, now time.Time) error {
	if token.UserID != userID {
		return apperrors.Unauthorized("refresh token is invalid")
	}
	if token.RevokedAt != nil || now.After(token.ExpiresAt) {
		return apperrors.Unauthorized("refresh token is expired")
	}
	return nil
}

func checkPublicVerifyCodePurpose(ctx context.Context, svcCtx *svc.ServiceContext, purpose, email string) error {
	switch purpose {
	case "register":
		total, err := svcCtx.Store.CountUsers(ctx)
		if err != nil {
			return err
		}
		if total > 0 {
			settings, err := svcCtx.Store.SiteSettings(ctx)
			if err != nil {
				return err
			}
			if !settings.RegistrationEnabled {
				return apperrors.Forbidden("registration is disabled")
			}
		}
		_, err = svcCtx.Store.FindUserByEmail(ctx, email)
		if err == nil {
			return apperrors.Conflict("email already exists")
		}
		if !errors.Is(err, model.ErrNotFound) {
			return err
		}
	case "reset_password":
		user, err := svcCtx.Store.FindUserByEmail(ctx, email)
		if err != nil {
			return logicutil.MapError(err)
		}
		if user.Status != "active" {
			return apperrors.Forbidden("user is disabled")
		}
	default:
		return apperrors.BadRequest("purpose is invalid")
	}
	return nil
}

func refreshKey(hash string) string {
	return refreshPrefix + hash
}

func publishEvent(ctx context.Context, svcCtx *svc.ServiceContext, event mq.Event) {
	if svcCtx.Events != nil {
		svcCtx.Events.Publish(ctx, event)
	}
}

func trim(value string) string {
	return strings.TrimSpace(value)
}
