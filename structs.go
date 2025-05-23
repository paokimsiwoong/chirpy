package main

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/paokimsiwoong/chirpy/internal/database"
)

type apiConfig struct {
	// standard-library type that allows us to safely increment and read an integer value across multiple goroutines (HTTP requests in this project)
	fileserverHits atomic.Int32
	// db 쿼리함수 접근을 위한 포인터
	ptrDB *database.Queries
	// dev냐 일반유저냐에 따라 몇몇 페이지 제한 여부가 갈림
	platform string
	// JWT 생성에 사용할 시크릿 키
	tokenSecret string
	// polka webhook 인증에 쓰이는 api 키
	polkaKey string
}

// 이 wrapper method로 http.Handler를 감싸는 새로운 http.Handler 반환
// 이 새 http.Handler는 원본 http.Handler의 ServeHTTP메소드를 그대로 호출하면서 추가로 fileserverHits 가 1씩 증가
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		// atomic.Int32 타입의 method Add (func (x *atomic.Int32) Add(delta int32) (new int32))로 값 변경
		next.ServeHTTP(w, r)
	})
	// @@@@@ type HandlerFunc func(ResponseWriter, *Request)은 일반 함수를 http.Handler로 취급할 수 있게 해주는 일종의 툴
	// @@@@@ http.HandlerFunc(함수)로 함수를 HandlerFunc 타입으로 형변환을 하면 이 HandlerFunc 타입은 ServeHTTP 메소드를 가지고 있으므로 http.Handler 인터페이스를 구현한다
	// @@@@@ 단 함수 시그니처가 func(ResponseWriter, *Request)여야 한다
	// @@@@@@@@ HandlerFunc 타입의 ServeHTTP 메소드 : func (f http.HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) { f(w,r) }
	// @@@@@@@@ ===> 단순히 HandlerFunc 타입인 함수 자기 자신을 호출
	// https://pkg.go.dev/net/http#HandlerFunc
}

// @@@ 여러개의 함수에서 사용되는 구조체들은 structs.go에 저장
type cReqBody struct {
	Body string `json:"body"`
	// UserID uuid.UUID `json:"user_id"` //@@@ auth.ValidateJWT가 토큰정보를 받아 uuid 반환하므로 삭제
}

type cResBodySuccess struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

type uReqBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	// ExpiresInSeconds int    `json:"expires_in_seconds"` jwt 수명 고정
}

type uResBodySuccess struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	IsChirpyRed  bool      `json:"is_chirpy_red"`
	// @@@ hashed password는 절대 response로 반환하면 안된다 => 보안문제
}
