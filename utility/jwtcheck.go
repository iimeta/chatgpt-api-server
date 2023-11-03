package utility

import (
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
)

func parseJWTWithoutValidation(tokenString string) (jwt.MapClaims, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	} else {
		return nil, err
	}
}

func CheckAccessToken(tokenString string) error {
	claims, err := parseJWTWithoutValidation(tokenString)
	if err != nil {
		return err
	}
	// 验证是否过期
	if !claims.VerifyExpiresAt(time.Now().Unix(), true) {
		return errors.New("token is expired")
	}
	return nil
}
