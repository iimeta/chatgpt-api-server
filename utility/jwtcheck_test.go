package utility

import (
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/stretchr/testify/assert"
)

func TestParseJWTWithoutValidation(t *testing.T) {
	t.Run("valid token", func(t *testing.T) {
		// Create a token string
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"foo": "bar",
		})
		tokenString, _ := token.SignedString([]byte("secret"))

		// Call the function
		claims, err := parseJWTWithoutValidation(tokenString)
		g.Dump(claims)
		// 验证是否过期
		if !claims.VerifyExpiresAt(time.Now().Unix(), true) {
			t.Error("token is expired")
		}

		// Assert the results
		assert.NoError(t, err)
		assert.Equal(t, "bar", claims["foo"])
	})

	t.Run("invalid token", func(t *testing.T) {
		// Call the function with an invalid token string
		_, err := parseJWTWithoutValidation("invalid token")

		// Assert the results
		assert.Error(t, err)
	})
}
