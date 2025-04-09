package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// @@@ 해답은 Issuer 부분에 입력할 값을 함수 바깥에서 const로 선언함
type TokenType string

const (
	// TokenTypeAccess -
	TokenTypeAccess TokenType = "chirpy-access" // @@@ 해답은 이 값을 ValidateJWT에서도 사용해서 부정토큰(토큰 정규발급자가 아닌 자가 위조한 토큰)을 가려내는데 사용
)

// 암호를 받아서 hash로 변환해주는 함수
func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	// cost가 높을 수록 해쉬 처리 단계가 늘어나 뚫기 어려워진다 @@@ 기본값 10은 bcrypt.DefaultCost 로 입력 가능
	// https://www.reddit.com/r/PHPhelp/comments/1114hl3/the_optimal_bcrypt_cost/?rdt=42369
	// []byte가 쓰이는 이유
	// https://stackoverflow.com/questions/8881291/why-is-char-preferred-over-string-for-passwords
	return string(hashed), err
}

// hash 와 입력된 암호를 비교하는 함수 nil이면 암호일치, nil이 아니면 불일치
func CheckPasswordHash(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// JWT(JSON Web Token) 생성함수
func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	now := time.Now()

	// JWT 토큰 생성
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		// Issuer:    "chirpy",
		Issuer: string(TokenTypeAccess),
		// @@@ Issuer가 다른 부정토큰을 ValidateJWT에서 걸러내기 위해 함수 외부에 const로 issuer 저장
		IssuedAt:  jwt.NewNumericDate(now), // jwt.NewNumericDate 함수는 time.Time을 담는 jwt.NumericDate 구조체의 포인터를 반환
		ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
		Subject:   userID.String(), // uuid.UUID의 string() method는 uuid의 string 버전 반환
	},
	)

	// 토큰 생성 시에 유저가 제공한 tokenSecret을 같이 사용해서 생성한다.
	return token.SignedString([]byte(tokenSecret))
	// HS256은 key에 []byte 타입 입력해야한다
	// https://golang-jwt.github.io/jwt/usage/signing_methods/#signing-methods-and-key-types
}

// JWT 검증함수
func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	// MakeJWT 함수에서 사용한 jwt.Claims 구현 타입을 그대로 사용
	claims := jwt.RegisteredClaims{}
	// ??? jwt.NewWithClaims로 생성된 token은 Claims 필드에 함수 인자로 제공된 claim이 저장되고
	// jwt.ParseWithClaims는 tokenString으로부터 token(*jwt.Token)을 다시 얻어내는 과정에서 token의 필드 Claims도 복원되는데
	// 이때 복호화(decode)된 데이터들을 다시 담을 claim은 생성 당시 claim과 동일한 구조체여야 한다
	// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
	// claims := jwt.MapClaims{} // type MapClaims map[string]interface{}
	// MapClaims is a claims type that uses the map[string]interface{} for JSON decoding
	// @@@ 만약 토큰 생성시의 claim 구조체의 구조(JSON 구조)를 모를 경우
	// @@@ 어떠한 형태건 받아들일 수 있는 map[string]interface{} 사용가능
	// @@@ map[json_key]json_value 형태, 임의의 모든 타입은 interface{} 구현
	// @@@ jwt.MapClaims{}을 쓸경우 ParseWithClaims에 인자로 입력할 때 & 있거나 없거나 문제없음
	// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@

	// tokenString decode해서 token(*jwt.Token) 반환
	// @@@ 두번째인자 claims는 jwt.MapClaims{}를 쓰지않는 경우 pointer를 입력해야 에러가 안난다.
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	})
	// 3번째 인자 keyFunc는 tokenSecret 처리에 쓰이는 함수로
	// 그냥 원본 그대로 사용시에는 함수 시그니처 만족하면서 []byte(tokenSecret) 반환하는 함수를 인자로 입력하면 된다.
	if err != nil { // 토큰이 invalid하거나 expired일 경우 err != nil
		return uuid.UUID{}, fmt.Errorf("error token is invalid or expired: %w", err)
		// @@@ uuid.Nil 사용 가능
	}

	idString, err := token.Claims.GetSubject()
	if err != nil { // 토큰이 invalid하거나 expired일 경우 err != nil
		return uuid.UUID{}, fmt.Errorf("error getting string uuid: %w", err)
	}
	// jwt.Claims 인터페이스 구현 조건에는 GetSubject() (string, error) 가 존재

	// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
	// @@@ 해답의 예외처리 반영
	// @@@ 부정토큰(토큰 정규발급자가 아닌 자가 위조한 토큰)을 걸러내기
	issuer, err := token.Claims.GetIssuer()
	if err != nil {
		return uuid.Nil, err
	}
	if issuer != string(TokenTypeAccess) {
		return uuid.Nil, errors.New("invalid issuer")
	}
	// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@

	id, err := uuid.Parse(idString)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("error parsing string uuid: %w", err)
	}

	return id, nil
}

// Authorization header에 들어있는 인증 정보에서 tokenString만 추출해서 반환하는 함수
func GetBearerToken(headers http.Header) (string, error) {
	// header에서 정보 불러오기
	auth := headers.Get("Authorization")
	// authorization 헤더 내용은 "Bearer <tokenString>" 형태
	if auth == "" {
		return "", errors.New("can't find authorization header")
		// @@@ 해답은 함수 바깥에서 var ErrNoAuthHeaderIncluded = errors.New("no auth header included in request") 정의한 후 여기서 사용
	}

	// @@@ 헤더 내용에 Bearer 단어는 고정임 ==> 예외 처리
	splitAuth := strings.Split(auth, " ")
	if len(splitAuth) != 2 || splitAuth[0] != "Bearer" {
		return "", errors.New("invalid authorization header")
	}

	tokenString := strings.Trim(splitAuth[1], " ")

	return tokenString, nil
}
