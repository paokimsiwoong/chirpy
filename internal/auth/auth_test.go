package auth

import (
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func TestHash(t *testing.T) {
	cases := []struct {
		input   string
		attempt string
	}{
		{input: "password", attempt: "pass"},
		{input: "testING2@", attempt: "testing22"},
	}

	passCount := 0
	failCount := 0

	for _, c := range cases {
		hashed, err := HashPassword(c.input)
		if err != nil {
			log.Fatalf("HashPassword error: %v\n", err)
		}
		fmt.Println("---------------------------------")

		fmt.Printf("Password: %s\n", c.input)
		fmt.Printf("Hashed: %s\n", hashed)

		expectedHash, err := bcrypt.GenerateFromPassword([]byte(c.input), 12)
		if err != nil {
			log.Fatalf("bcrypt.GenerateFromPassword error: %v\n", err)
		}

		// 		if hashed != string(expectedHash) {
		// 			failCount++
		// 			t.Errorf(
		// 				`---------------------------------
		// HashPassword fail
		// Inputs:     "%v"
		// Expecting:   %v
		// Actual:      %v
		// Fail
		// `,
		// 				c.input, string(expectedHash), hashed,
		// 			)
		// 			continue
		// 		}
		// @@@ 해쉬 생성시에 들어가는 salt가 달라져서 비교 불가능

		// 틀린 비밀번호와 해쉬 비교할 경우
		errC := CheckPasswordHash(hashed, c.attempt)
		errE := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(c.attempt))

		if errC != errE {
			failCount++
			t.Errorf(
				`---------------------------------
CheckPasswordHash fail1
Input hash:         "%v"
Input password:     "%v"
Expecting:           %v
Actual:              %v
Fail
`,
				hashed, c.attempt, errE, errC,
			)
			continue
		}

		// 제대로된 비밀번호와 해쉬 비교할 경우
		errCC := CheckPasswordHash(hashed, c.input)
		errEE := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(c.input))

		if errCC != errEE {
			failCount++
			t.Errorf(
				`---------------------------------
CheckPasswordHash fail2
Input hash:         "%v"
Input password:     "%v"
Expecting:           %v
Actual:              %v
Fail
`,
				hashed, c.input, errE, errC,
			)
			continue
		}

		passCount++
		fmt.Printf(
			`---------------------------------
HashPassword and CheckPasswordHash pass
Input password:                 "%v"
Input attempt:                  "%v"
Expecting hash:                  %v
Hash:                            %v
Expecting check error:           %v
Actual check error:              %v
Pass
`,
			c.input, c.attempt, string(expectedHash), hashed, errE, errC,
		)

	}

	fmt.Println("---------------------------------")
	fmt.Printf("%d passed, %d failed\n", passCount, failCount)

	// @@@ TODO: t.Run() 활용해보기
}

func TestJWT(t *testing.T) {
	// uuid, tokenSecret(string), expiresIn으로 테스트에 사용할 tokenString 생성
	userID := uuid.New()
	tokenSecret := "testSceret"
	expiresIn, _ := time.ParseDuration("2s")
	originalTokenString, _ := MakeJWT(userID, tokenSecret, expiresIn)

	waitTime, _ := time.ParseDuration("1ms")

	tests := []struct {
		name        string
		tokenSecret string
		waitTime    time.Duration
		tokenString string
		wantErr     bool
	}{
		{
			name:        "Correct tokenSecret",
			tokenSecret: tokenSecret,
			waitTime:    waitTime,
			tokenString: originalTokenString,
			wantErr:     false,
		},
		{
			name:        "Incorrect tokenSecret",
			tokenSecret: "wrongSecret",
			waitTime:    waitTime,
			tokenString: originalTokenString,
			wantErr:     true,
		},
		{
			name:        "Invalid tokenString",
			tokenSecret: tokenSecret,
			waitTime:    waitTime,
			tokenString: "invalid",
			wantErr:     true,
		},
		{
			name:        "token expired",
			tokenSecret: tokenSecret,
			waitTime:    2 * expiresIn,
			tokenString: originalTokenString,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			time.Sleep(tt.waitTime)
			_, err := ValidateJWT(tt.tokenString, tt.tokenSecret)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJWT() error = %v, wantErr %v", err, tt.wantErr)
			}
			// @@@ 해답은 위의 if block에 return을 넣고
			// @@@ tt에 예상되는 userID 반환값 필드를 추가해 그 부분도 테스트
		})
	}
}

// @@@ 해답 GetBearerToken 테스트 함수
func TestGetBearerToken(t *testing.T) {
	tests := []struct {
		name      string
		headers   http.Header
		wantToken string
		wantErr   bool
	}{
		{
			name: "Valid Bearer token",
			headers: http.Header{
				"Authorization": []string{"Bearer valid_token"},
			},
			wantToken: "valid_token",
			wantErr:   false,
		},
		{
			name:      "Missing Authorization header",
			headers:   http.Header{},
			wantToken: "",
			wantErr:   true,
		},
		{
			name: "Malformed Authorization header",
			headers: http.Header{
				"Authorization": []string{"InvalidBearer token"},
			},
			wantToken: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotToken, err := GetBearerToken(tt.headers)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBearerToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotToken != tt.wantToken {
				t.Errorf("GetBearerToken() gotToken = %v, want %v", gotToken, tt.wantToken)
			}
		})
	}
}
