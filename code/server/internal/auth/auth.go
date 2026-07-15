package auth

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"time"
)

type Service struct {
	username, hash, secret string
	ttl                    time.Duration
}

func New(username, hash, secret string, ttl time.Duration) *Service {
	return &Service{username, hash, secret, ttl}
}
func (s *Service) Login(username, password string) (string, time.Time, error) {
	if username != s.username || bcrypt.CompareHashAndPassword([]byte(s.hash), []byte(password)) != nil {
		return "", time.Time{}, errors.New("invalid credentials")
	}
	exp := time.Now().Add(s.ttl)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": username, "exp": exp.Unix()})
	out, err := token.SignedString([]byte(s.secret))
	return out, exp, err
}
func (s *Service) Verify(tokenString string) error {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(s.secret), nil
	})
	if err != nil || !token.Valid {
		return errors.New("invalid token")
	}
	return nil
}
