package jwt

import (
	"errors"
	"fmt"

	jwtlib "github.com/golang-jwt/jwt"
)

type JWT struct {
	signingKey []byte
}

type CustomClaims struct {
	Id int64 `json:"id"`
	jwtlib.StandardClaims
}

func NewJWT(signingKey []byte) *JWT {
	return &JWT{
		signingKey: signingKey,
	}
}

func (j *JWT) CreateToken(claims CustomClaims) (string, error) {
	token := jwtlib.NewWithClaims(
		jwtlib.SigningMethodHS256,
		claims,
	)

	return token.SignedString(j.signingKey)
}

func (j *JWT) ParseToken(tokenString string) (*CustomClaims, error) {
	token, err := jwtlib.ParseWithClaims(
		tokenString,
		&CustomClaims{},
		func(token *jwtlib.Token) (interface{}, error) {
			if token.Method != jwtlib.SigningMethodHS256 {
				return nil, fmt.Errorf(
					"不支持的签名算法：%v",
					token.Header["alg"],
				)
			}

			return j.signingKey, nil
		},
	)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, errors.New("无效的 token")
	}

	return claims, nil
}
