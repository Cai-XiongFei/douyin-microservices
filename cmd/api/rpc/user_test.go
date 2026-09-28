package rpc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"douyin/kitex/kitex_gen/user"
)

func TestUserRPC(t *testing.T) {
	if err := InitUser(); err != nil {
		t.Fatalf("initialize user RPC client failed: %v", err)
	}

	username := fmt.Sprintf("rpc_user_%d", time.Now().UnixNano())
	password := "123456"

	registerResponse, err := Register(
		context.Background(),
		&user.UserRegisterRequest{
			Username: username,
			Password: password,
		},
	)
	if err != nil {
		t.Fatalf("RPC register failed: %v", err)
	}

	if registerResponse.StatusCode != 0 {
		t.Fatalf(
			"register returned failure: %+v",
			registerResponse,
		)
	}

	loginResponse, err := Login(
		context.Background(),
		&user.UserLoginRequest{
			Username: username,
			Password: password,
		},
	)
	if err != nil {
		t.Fatalf("RPC login failed: %v", err)
	}

	if loginResponse.StatusCode != 0 {
		t.Fatalf(
			"login returned failure: %+v",
			loginResponse,
		)
	}

	if loginResponse.UserId != registerResponse.UserId {
		t.Fatalf(
			"user ID mismatch: register=%d login=%d",
			registerResponse.UserId,
			loginResponse.UserId,
		)
	}
}
