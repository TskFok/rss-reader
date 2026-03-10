package services

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ushopal/rss-reader/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUserExists    = errors.New("用户名已存在")
	ErrUserLocked    = errors.New("用户已被锁定，请联系管理员解锁")
	ErrInvalidCreds  = errors.New("用户名或密码错误")
)

// AuthService 认证服务
type AuthService struct {
	DB          *gorm.DB
	jwtSecret   []byte
	expireHours int
	superAdmin  string
}

// NewAuthService 创建认证服务
func NewAuthService(db *gorm.DB, jwtSecret string, expireHours int, superAdminUsername string) *AuthService {
	return &AuthService{
		DB:          db,
		jwtSecret:   []byte(jwtSecret),
		expireHours: expireHours,
		superAdmin:  superAdminUsername,
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

// Register 注册新用户，默认 locked
func (s *AuthService) Register(req RegisterRequest) (*models.User, error) {
	var count int64
	if err := s.DB.Model(&models.User{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrUserExists
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	isSuperAdmin := false
	if s.DB.Model(&models.User{}).Count(&count); count == 0 {
		isSuperAdmin = true
	} else if s.superAdmin != "" && req.Username == s.superAdmin {
		isSuperAdmin = true
	}
	user := &models.User{
		Username:     req.Username,
		PasswordHash: string(hash),
		Status:       models.UserStatusLocked,
		IsSuperAdmin: isSuperAdmin,
	}
	if err := s.DB.Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResult 登录结果
type LoginResult struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

// Login 登录，仅 active 用户可登录
func (s *AuthService) Login(req LoginRequest) (*LoginResult, error) {
	var user models.User
	if err := s.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCreds
		}
		return nil, err
	}
	if user.Status != models.UserStatusActive {
		return nil, ErrUserLocked
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCreds
	}
	token, err := s.generateToken(&user)
	if err != nil {
		return nil, err
	}
	return &LoginResult{Token: token, User: &user}, nil
}

type jwtClaims struct {
	UserID uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func (s *AuthService) generateToken(user *models.User) (string, error) {
	claims := jwtClaims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(s.expireHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(s.jwtSecret)
}

// GenerateTokenForUser 对外暴露的生成 token 方法，供第三方登录复用
func (s *AuthService) GenerateTokenForUser(user *models.User) (string, error) {
	return s.generateToken(user)
}

// ValidateToken 验证 token 并返回用户 ID
func (s *AuthService) ValidateToken(tokenString string) (uint, error) {
	t, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(t *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})
	if err != nil {
		return 0, err
	}
	claims, ok := t.Claims.(*jwtClaims)
	if !ok || !t.Valid {
		return 0, errors.New("invalid token")
	}
	return claims.UserID, nil
}
