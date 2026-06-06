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
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/internal/validator"
	"notes-of-ashen/model"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

const refreshPrefix = "notes-of-ashen:refresh:"

func Register(ctx context.Context, svcCtx *svc.ServiceContext, req types.RegisterReq, meta types.RequestMeta) (*types.TokenPair, error) {
	req.Account = trim(req.Account)
	req.Email = trim(req.Email)
	req.Nickname = trim(req.Nickname)
	req.AvatarURL = trim(req.AvatarURL)
	if err := validateRegister(req); err != nil {
		return nil, err
	}

	total, err := svcCtx.Store.CountUsers(ctx)
	if err != nil {
		return nil, err
	}
	role := "user"
	if total == 0 {
		role = "admin"
	} else {
		settings, err := svcCtx.Store.SiteSettings(ctx)
		if err != nil {
			return nil, err
		}
		if !settings.RegistrationEnabled {
			return nil, apperrors.Forbidden("registration is disabled")
		}
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	id, err := svcCtx.Store.CreateUser(ctx, model.UserCreate{
		Account:      req.Account,
		PasswordHash: string(passwordHash),
		Email:        req.Email,
		AvatarURL:    req.AvatarURL,
		Nickname:     req.Nickname,
		Role:         role,
	})
	if err != nil {
		if logicutil.IsDuplicate(err) {
			return nil, apperrors.Conflict("account or email already exists")
		}
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
	if err := validator.Required(req.Account, "account"); err != nil {
		return nil, err
	}
	if err := validator.Required(req.Password, "password"); err != nil {
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

func Refresh(ctx context.Context, svcCtx *svc.ServiceContext, req types.RefreshReq) (*types.TokenPair, error) {
	req.RefreshToken = trim(req.RefreshToken)
	if err := validator.Required(req.RefreshToken, "refreshToken"); err != nil {
		return nil, err
	}
	hash := authutil.HashRefreshToken(req.RefreshToken)
	userIDRaw, err := svcCtx.Redis.Get(ctx, refreshKey(hash)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, apperrors.Unauthorized("refresh token is invalid")
		}
		return nil, err
	}

	token, err := svcCtx.Store.FindRefreshToken(ctx, hash)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	if token.RevokedAt != nil || time.Now().After(token.ExpiresAt) {
		return nil, apperrors.Unauthorized("refresh token is expired")
	}
	redisUserID, err := strconv.ParseUint(userIDRaw, 10, 64)
	if err != nil || redisUserID != token.UserID {
		return nil, apperrors.Unauthorized("refresh token is invalid")
	}
	user, err := svcCtx.Store.FindUserByID(ctx, token.UserID)
	if err != nil {
		return nil, logicutil.MapError(err)
	}
	if user.Status != "active" {
		return nil, apperrors.Forbidden("user is disabled")
	}

	if err := svcCtx.Store.RevokeRefreshToken(ctx, hash); err != nil {
		return nil, err
	}
	_ = svcCtx.Redis.Del(ctx, refreshKey(hash)).Err()
	return issueTokens(ctx, svcCtx, user.ID, user.Role)
}

func Logout(ctx context.Context, svcCtx *svc.ServiceContext, req types.RefreshReq, meta types.RequestMeta) error {
	userID, err := authutil.UserID(ctx)
	if err != nil {
		return err
	}
	req.RefreshToken = trim(req.RefreshToken)
	if err := validator.Required(req.RefreshToken, "refreshToken"); err != nil {
		return err
	}
	hash := authutil.HashRefreshToken(req.RefreshToken)
	if err := svcCtx.Store.RevokeRefreshToken(ctx, hash); err != nil {
		return err
	}
	_ = svcCtx.Redis.Del(ctx, refreshKey(hash)).Err()
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
