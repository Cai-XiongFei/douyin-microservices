package jwt

import (
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt"
)

func TestCreateAndParseToken(t *testing.T) {
	jwtService := NewJWT([]byte("signingKey"))

	claims := CustomClaims{
		Id: 1001,
		StandardClaims: jwtlib.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		},
	}

	token, err := jwtService.CreateToken(claims)
	if err != nil {
		t.Fatalf("创建 token 失败：%v", err)
	}

	parsedClaims, err := jwtService.ParseToken(token)
	if err != nil {
		t.Fatalf("解析 token 失败：%v", err)
	}

	if parsedClaims.Id != 1001 {
		t.Fatalf("期望用户 ID 为 1001，实际为 %d", parsedClaims.Id)
	}
}
