package model

import (
	"crypto/rand"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type UserClaims struct {
	User User
	jwt.RegisteredClaims
	ExpiresAt int64
	Id        int
	IssuedAt  int64
	Phone     string
}

type AdvisorClaims struct {
	Advisor Advisor
	jwt.RegisteredClaims
	ExpiresAt int64
	Id        int
	IssuedAt  int64
	Phone     string
}

var JwtKey = make([]byte, 32)

// JWTGenerateToken Written:by lwj
// Function:Generate a token
// Receive: User or Advisor struct
func JWTGenerateUserToken(user User, d time.Duration) string {
	expireTime := time.Now().Add(d)
	//Generate the token
	claims := UserClaims{
		User:             user,
		RegisteredClaims: jwt.RegisteredClaims{},
		ExpiresAt:        expireTime.Unix(),
		IssuedAt:         time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	if _, err := rand.Read(JwtKey); err != nil {
		panic(err.Error())
	}
	jwtStr, err := token.SignedString(JwtKey)
	if err != nil {
		panic(err.Error())
	}
	return jwtStr
}

func JWTGenerateAdvisorToken(advisor Advisor, d time.Duration) string {

	expireTime := time.Now().Add(d)
	//Generate the token
	claims := AdvisorClaims{
		Advisor:          advisor,
		RegisteredClaims: jwt.RegisteredClaims{},
		ExpiresAt:        expireTime.Unix(),
		IssuedAt:         time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	if _, err := rand.Read(JwtKey); err != nil {
		panic(err.Error())
	}
	jwtStr, err := token.SignedString(JwtKey)
	if err != nil {
		panic(err.Error())
	}
	return jwtStr

}

func JWTParseUserToken(jwtStr string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(jwtStr, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return JwtKey, nil
	})
	if jwtStr == "" {
		return nil, errors.New("no token is found in Headers")
	}
	if err != nil {
		return nil, errors.New("invalid token")
	}

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

func JWTParseAdvisorToken(jwtStr string) (*AdvisorClaims, error) {
	token, err := jwt.ParseWithClaims(jwtStr, &AdvisorClaims{}, func(token *jwt.Token) (interface{}, error) {
		return JwtKey, nil
	})
	if jwtStr == "" {
		return nil, errors.New("no token is found in Headers	")
	}
	if err != nil {
		return nil, errors.New("invalid token")
	}

	if claims, ok := token.Claims.(*AdvisorClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}
