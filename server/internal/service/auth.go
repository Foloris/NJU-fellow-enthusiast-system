package service

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"campus-fellow/server/internal/model"
	"campus-fellow/server/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type claims struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type Tokens struct{ secret []byte }

func NewTokens(secret string) *Tokens { return &Tokens{secret: []byte(secret)} }
func (t *Tokens) Sign(user model.User) (string, error) {
	now := time.Now()
	c := claims{ID: user.ID, Username: user.Username, Role: user.Role, RegisteredClaims: jwt.RegisteredClaims{Subject: strconv.FormatInt(user.ID, 10), IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour))}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(t.secret)
}
func (t *Tokens) Verify(raw string) (int64, error) {
	c := &claims{}
	token, err := jwt.ParseWithClaims(raw, c, func(_ *jwt.Token) (any, error) { return t.secret, nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil {
		return 0, err
	}
	if !token.Valid || c.ID < 1 {
		return 0, errors.New("invalid token")
	}
	return c.ID, nil
}

func (s *Service) Authenticate(ctx context.Context, raw string) (*model.User, error) {
	if raw == "" {
		return nil, nil
	}
	id, err := s.Tokens.Verify(raw)
	if err != nil {
		return nil, nil
	}
	u, err := s.Repo.UserByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if u.Status != "active" {
		return nil, nil
	}
	u.Status = ""
	return &u, nil
}

type RegisterInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
}
type LoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type ProfileInput struct {
	Nickname string  `json:"nickname"`
	Bio      string  `json:"bio"`
	Avatar   *string `json:"avatar"`
}

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_\x{4e00}-\x{9fa5}]+$`)

func (s *Service) Register(ctx context.Context, in RegisterInput) (model.User, error) {
	in.Username = strings.TrimSpace(in.Username)
	in.Nickname = strings.TrimSpace(in.Nickname)
	if !length(in.Username, 3, 24) || !usernamePattern.MatchString(in.Username) {
		return model.User{}, fail(400, "账号须为3—24个字符，仅包含中英文、数字或下划线")
	}
	if !length(in.Password, 8, 72) || len(in.Password) > 72 {
		return model.User{}, fail(400, "密码至少8个字符，且不超过72字节")
	}
	if !length(in.Nickname, 1, 30) {
		return model.User{}, fail(400, "昵称应为1—30个字符")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), 12)
	if err != nil {
		return model.User{}, err
	}
	u, err := s.Repo.CreateUser(ctx, in.Username, string(hash), in.Nickname)
	if repository.Duplicate(err) {
		return u, fail(409, "这个账号已经被使用")
	}
	u.Status = ""
	return u, err
}

func (s *Service) Login(ctx context.Context, in LoginInput) (model.User, error) {
	in.Username = strings.TrimSpace(in.Username)
	if in.Username == "" || in.Password == "" {
		return model.User{}, fail(400, "请输入账号和密码")
	}
	u, err := s.Repo.UserByUsername(ctx, in.Username)
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, fail(401, "账号或密码不正确")
	}
	if err != nil {
		return u, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)) != nil {
		return model.User{}, fail(401, "账号或密码不正确")
	}
	if u.Status == "banned" {
		return model.User{}, fail(403, "该账号已被封禁，请联系管理员申诉")
	}
	u.Status = ""
	return u, nil
}

func (s *Service) UpdateProfile(ctx context.Context, user model.User, in ProfileInput) (model.User, error) {
	in.Nickname = strings.TrimSpace(in.Nickname)
	in.Bio = strings.TrimSpace(in.Bio)
	trimPointer(&in.Avatar)
	if !length(in.Nickname, 1, 30) || !length(in.Bio, 0, 200) || !optionalLength(in.Avatar, 500) {
		return model.User{}, fail(400, "个人资料长度不符合要求")
	}
	if in.Avatar == nil {
		in.Avatar = user.Avatar
	}
	u, err := s.Repo.UpdateProfile(ctx, user.ID, in.Nickname, in.Bio, in.Avatar)
	u.Status = ""
	return u, err
}

type PublicUser struct {
	ID            int64   `json:"id"`
	Username      string  `json:"username"`
	Nickname      string  `json:"nickname"`
	Avatar        *string `json:"avatar"`
	Bio           string  `json:"bio"`
	Role          string  `json:"role"`
	CreatedAt     string  `json:"createdAt"`
	ExchangeCount int     `json:"exchangeCount"`
	MateCount     int     `json:"mateCount"`
}

func (s *Service) PublicUser(ctx context.Context, id int64) (PublicUser, error) {
	u, err := s.Repo.UserByID(ctx, id)
	if err != nil {
		return PublicUser{}, missing(err, "用户不存在")
	}
	if u.Status != "active" {
		return PublicUser{}, fail(404, "用户不存在")
	}
	exchanges, mates, err := s.Repo.UserCounts(ctx, id)
	return PublicUser{ID: u.ID, Username: u.Username, Nickname: u.Nickname, Avatar: u.Avatar, Bio: u.Bio, Role: u.Role, CreatedAt: u.CreatedAt, ExchangeCount: exchanges, MateCount: mates}, err
}
