package security

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image/color"
	"math/big"
	"strings"
	"time"

	apperrors "notes-of-ashen/internal/errors"

	"github.com/mojocn/base64Captcha"
	"github.com/redis/go-redis/v9"
)

const (
	EmailCodeTTL      = 5 * time.Minute
	EmailCodeCooldown = 60 * time.Second
	CaptchaTTL        = 5 * time.Minute
)

type CaptchaChallenge struct {
	ID        string
	ImageData string
	ExpiresIn int64
}

func NewCaptcha(ctx context.Context, redisClient *redis.Client, purpose string) (*CaptchaChallenge, error) {
	purpose, err := NormalizePurpose(purpose)
	if err != nil {
		return nil, err
	}
	code, err := RandomDigits(4)
	if err != nil {
		return nil, err
	}
	id, err := RandomID()
	if err != nil {
		return nil, err
	}
	// 先绘制图片成功再写 Redis（P4-10）：字体加载失败时 DrawCaptcha 报错，
	// 此处直接返回 error（不写 Redis、handler 返回 500 让用户重试），
	// 避免用户拿到空图却已占用 Redis 验证码 key 无法重试。
	// captchaImageData 返回的是完整 data URL（库 EncodeB64string 自带 data:image/png;base64, 前缀），
	// 不要再手动拼接前缀，否则会产生 data:image/png;base64,data:image/png;base64,... 双重前缀。
	imageData, err := captchaImageData(code)
	if err != nil {
		return nil, err
	}
	if err := redisClient.Set(ctx, CaptchaKey(purpose, id), code, CaptchaTTL).Err(); err != nil {
		return nil, err
	}
	return &CaptchaChallenge{
		ID:        id,
		ImageData: imageData,
		ExpiresIn: int64(CaptchaTTL / time.Second),
	}, nil
}

func VerifyCaptcha(ctx context.Context, redisClient *redis.Client, purpose, id, code string) error {
	purpose, err := NormalizePurpose(purpose)
	if err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	code = strings.TrimSpace(code)
	if id == "" || code == "" {
		return apperrors.BadRequest("captcha is required")
	}

	key := CaptchaKey(purpose, id)
	stored, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return apperrors.BadRequest("captcha is expired")
		}
		return err
	}
	if !strings.EqualFold(stored, code) {
		return apperrors.BadRequest("captcha is incorrect")
	}
	return redisClient.Del(ctx, key).Err()
}

func StoreEmailCode(ctx context.Context, redisClient *redis.Client, purpose, email, code string) error {
	purpose, err := NormalizeEmailPurpose(purpose)
	if err != nil {
		return err
	}
	email = NormalizeEmail(email)
	cooldownKey := EmailCodeCooldownKey(purpose, email)
	ok, err := redisClient.SetNX(ctx, cooldownKey, "1", EmailCodeCooldown).Result()
	if err != nil {
		return err
	}
	if !ok {
		return apperrors.TooManyRequests("verify code was sent recently")
	}
	if err := redisClient.Set(ctx, EmailCodeKey(purpose, email), code, EmailCodeTTL).Err(); err != nil {
		_ = redisClient.Del(ctx, cooldownKey).Err()
		return err
	}
	return nil
}

func ClearEmailCode(ctx context.Context, redisClient *redis.Client, purpose, email string) error {
	purpose, err := NormalizeEmailPurpose(purpose)
	if err != nil {
		return err
	}
	email = NormalizeEmail(email)
	return redisClient.Del(ctx, EmailCodeKey(purpose, email), EmailCodeCooldownKey(purpose, email)).Err()
}

func ConsumeEmailCode(ctx context.Context, redisClient *redis.Client, purpose, email, code string) error {
	purpose, err := NormalizeEmailPurpose(purpose)
	if err != nil {
		return err
	}
	email = NormalizeEmail(email)
	code = strings.TrimSpace(code)
	if code == "" {
		return apperrors.BadRequest("email code is required")
	}

	key := EmailCodeKey(purpose, email)
	stored, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return apperrors.BadRequest("email code is expired")
		}
		return err
	}
	if stored != code {
		return apperrors.BadRequest("email code is incorrect")
	}
	return redisClient.Del(ctx, key).Err()
}

func RandomDigits(length int) (string, error) {
	if length <= 0 {
		return "", apperrors.BadRequest("code length is invalid")
	}
	var builder strings.Builder
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		builder.WriteByte(byte('0' + n.Int64()))
	}
	return builder.String(), nil
}

func RandomID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func CaptchaKey(purpose, id string) string {
	return fmt.Sprintf("captcha:%s:%s", purpose, strings.TrimSpace(id))
}

func EmailCodeKey(purpose, email string) string {
	return fmt.Sprintf("verify_code:%s:%s", purpose, NormalizeEmail(email))
}

func EmailCodeCooldownKey(purpose, email string) string {
	return fmt.Sprintf("verify_code_cooldown:%s:%s", purpose, NormalizeEmail(email))
}

func RateLimitKey(name, ip string) string {
	return fmt.Sprintf("rate_limit:%s:%s", strings.TrimSpace(name), strings.TrimSpace(ip))
}

func TrafficPVKey(date string) string {
	return fmt.Sprintf("traffic:pv:%s", strings.TrimSpace(date))
}

func TrafficUVKey(date string) string {
	return fmt.Sprintf("traffic:uv:%s", strings.TrimSpace(date))
}

func TrafficRefererKey(date string) string {
	return fmt.Sprintf("traffic:referer:%s", strings.TrimSpace(date))
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func NormalizePurpose(purpose string) (string, error) {
	purpose = strings.TrimSpace(purpose)
	switch purpose {
	case "login", "register", "reset_password", "change_password", "update_email":
		return purpose, nil
	default:
		return "", apperrors.BadRequest("purpose is invalid")
	}
}

func NormalizeEmailPurpose(purpose string) (string, error) {
	purpose = strings.TrimSpace(purpose)
	switch purpose {
	case "register", "reset_password", "change_password", "update_email":
		return purpose, nil
	default:
		return "", apperrors.BadRequest("purpose is invalid")
	}
}

// captchaDriver 使用 base64Captcha 的 DriverString 绘制带扭曲/干扰的验证码图像，
// 抗 OCR 能力显著强于此前的七段数码管手绘实现。Source 限定为纯数字，与 RandomDigits 保持一致。
var captchaDriver = (&base64Captcha.DriverString{
	Height:          48,
	Width:           140,
	NoiseCount:      60,
	ShowLineOptions: base64Captcha.OptionShowHollowLine | base64Captcha.OptionShowSlimeLine,
	Length:          4,
	Source:          "0123456789",
	BgColor:         &color.RGBA{R: 245, G: 241, B: 232, A: 255},
	Fonts:           []string{"wqy-microhei.ttc"},
}).ConvertFonts()

// captchaImageData 用配置好的 DriverString 绘制给定 code，返回 data:image/png;base64,...
// 形式的完整 data URL。库 EncodeB64string 自带前缀，调用方不要再拼接前缀。
// DrawCaptcha 仅在字体加载失败等极端情况报错，此时向上返回 error，由调用方决定
// 是否写 Redis（NewCaptcha 在画图失败时不写 Redis 并返回 500 让用户重试）。
func captchaImageData(code string) (string, error) {
	item, err := captchaDriver.DrawCaptcha(code)
	if err != nil {
		return "", fmt.Errorf("draw captcha failed: %w", err)
	}
	return item.EncodeB64string(), nil
}
