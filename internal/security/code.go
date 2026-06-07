package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/big"
	"strings"
	"time"

	apperrors "notes-of-ashen/internal/errors"

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
	if err := redisClient.Set(ctx, CaptchaKey(purpose, id), code, CaptchaTTL).Err(); err != nil {
		return nil, err
	}
	return &CaptchaChallenge{
		ID:        id,
		ImageData: "data:image/png;base64," + captchaPNGBase64(code),
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

func captchaPNGBase64(code string) string {
	const (
		width  = 140
		height = 48
	)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 245, G: 241, B: 232, A: 255}}, image.Point{}, draw.Src)

	drawNoise(img)
	for i, digit := range code {
		drawDigit(img, int(digit-'0'), 14+i*32, 8, color.RGBA{R: 41, G: 42, B: 37, A: 255})
	}

	var buffer strings.Builder
	encoder := base64.NewEncoder(base64.StdEncoding, &buffer)
	_ = png.Encode(encoder, img)
	_ = encoder.Close()
	return buffer.String()
}

func drawNoise(img *image.RGBA) {
	bounds := img.Bounds()
	for i := 0; i < 100; i++ {
		x := randomInt(bounds.Dx())
		y := randomInt(bounds.Dy())
		img.SetRGBA(x, y, color.RGBA{R: 190, G: 173, B: 126, A: 255})
	}
	for i := 0; i < 4; i++ {
		x1 := randomInt(bounds.Dx())
		y1 := randomInt(bounds.Dy())
		x2 := randomInt(bounds.Dx())
		y2 := randomInt(bounds.Dy())
		drawLine(img, x1, y1, x2, y2, color.RGBA{R: 166, G: 137, B: 78, A: 255})
	}
}

func drawDigit(img *image.RGBA, digit, x, y int, ink color.RGBA) {
	segments := [10][7]bool{
		{true, true, true, true, true, true, false},
		{false, true, true, false, false, false, false},
		{true, true, false, true, true, false, true},
		{true, true, true, true, false, false, true},
		{false, true, true, false, false, true, true},
		{true, false, true, true, false, true, true},
		{true, false, true, true, true, true, true},
		{true, true, true, false, false, false, false},
		{true, true, true, true, true, true, true},
		{true, true, true, true, false, true, true},
	}
	if digit < 0 || digit > 9 {
		return
	}
	rects := []image.Rectangle{
		image.Rect(x+4, y, x+22, y+5),
		image.Rect(x+22, y+4, x+27, y+18),
		image.Rect(x+22, y+22, x+27, y+36),
		image.Rect(x+4, y+36, x+22, y+41),
		image.Rect(x, y+22, x+5, y+36),
		image.Rect(x, y+4, x+5, y+18),
		image.Rect(x+4, y+18, x+22, y+23),
	}
	for i, on := range segments[digit] {
		if on {
			draw.Draw(img, rects[i], &image.Uniform{C: ink}, image.Point{}, draw.Src)
		}
	}
}

func drawLine(img *image.RGBA, x1, y1, x2, y2 int, ink color.RGBA) {
	dx := abs(x2 - x1)
	dy := -abs(y2 - y1)
	sx := -1
	if x1 < x2 {
		sx = 1
	}
	sy := -1
	if y1 < y2 {
		sy = 1
	}
	err := dx + dy
	for {
		img.SetRGBA(x1, y1, ink)
		if x1 == x2 && y1 == y2 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x1 += sx
		}
		if e2 <= dx {
			err += dx
			y1 += sy
		}
	}
}

func randomInt(max int) int {
	if max <= 0 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(n.Int64())
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
