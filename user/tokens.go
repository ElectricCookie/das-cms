package user

import (
	"time"

	"github.com/ElectricCookie/das-cms/configLoader"
	"github.com/ElectricCookie/das-cms/db"
	jwt "github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/scrypt"
)

// GenerateRefreshToken generates a new refresh-token and saves it in the database
func GenerateRefreshToken(userID string) (*string, error) {

	// Generate token
	token, err := generateRandomString(64)

	if err != nil {
		return nil, err
	}

	user, userErr := db.GetDb().GetUserById(userID)

	if userErr != nil {
		return nil, userErr
	}

	// Hash token

	dk, cryptError := scrypt.Key([]byte(token), []byte(user.Salt), 16384, 8, 1, 32)

	if cryptError != nil {
		return nil, cryptError
	}

	db.GetDb().InsertRefreshToken(&db.RefreshToken{
		Token:  string(dk),
		UserID: userID,
	})

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.StandardClaims{
		Issuer:   "das",
		Subject:  userID,
		IssuedAt: time.Now().Unix(),
		Id:       token,
	})

	resString, signError := jwtToken.SignedString(configLoader.GetConfig().JWTSecret)

	if signError != nil {
		return nil, signError
	}

	return &resString, nil

}

// CheckRefreshToken checks if a refresh token is valid. In case of invalidity a theft is assumed and the users sessions are nuked
func CheckRefreshToken(jwtToken string) (*jwt.StandardClaims, error) {

	claims := jwt.StandardClaims{}
	_, parseError := jwt.ParseWithClaims(jwtToken, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(configLoader.GetConfig().JWTSecret), nil
	})

	refreshToken := claims.Id
	userID := claims.Subject

	if parseError != nil {
		return nil, parseError
	}

	user, userErr := db.GetDb().GetUserById(userID)

	if userErr != nil {
		return nil, userErr
	}

	// Hash token

	dk, cryptError := scrypt.Key([]byte(refreshToken), []byte(user.Salt), 16384, 8, 1, 32)

	if cryptError != nil {
		return nil, cryptError
	}

	_, retrieveError := db.GetDb().FindRefreshToken(userID, string(dk))

	if retrieveError != nil {
		return nil, retrieveError
	}

	return &claims, nil

}

// GenerateAccessToken issues a new access token based on a refresh token
func GenerateAccessToken(refreshToken string) (string, error) {

	claims, err := CheckRefreshToken(refreshToken)
	if err != nil {
		return "", err
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.StandardClaims{
		Issuer:    "das",
		IssuedAt:  time.Now().Unix(),
		Subject:   claims.Subject,
		ExpiresAt: time.Now().Add(time.Minute * 30).Unix(),
	}).SignedString(configLoader.GetConfig().JWTSecret)

}
