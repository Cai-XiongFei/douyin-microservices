package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"douyin/kitex/kitex_gen/user"
)

func TestRegisterAndLogin(t *testing.T) {
	service := NewUserServiceImpl("test-signing-key")

	// 每次测试生成不同的用户名，防止和数据库中的旧数据重复
	username := fmt.Sprintf("test_user_%d", time.Now().UnixNano())
	password := "123456"

	registerResponse, err := service.Register(
		context.Background(),
		&user.UserRegisterRequest{
			Username: username,
			Password: password,
		},
	)
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	if registerResponse.StatusCode != 0 {
		t.Fatalf(
			"register returned failure: %+v",
			registerResponse,
		)
	}

	if registerResponse.UserId == 0 {
		t.Fatal("register did not return user id")
	}

	if registerResponse.Token == "" {
		t.Fatal("register did not return token")
	}

	loginResponse, err := service.Login(
		context.Background(),
		&user.UserLoginRequest{
			Username: username,
			Password: password,
		},
	)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if loginResponse.StatusCode != 0 {
		t.Fatalf(
			"login returned failure: %+v",
			loginResponse,
		)
	}

	if loginResponse.UserId != registerResponse.UserId {
		t.Fatalf(
			"user id mismatch: register=%d login=%d",
			registerResponse.UserId,
			loginResponse.UserId,
		)
	}

	if loginResponse.Token == "" {
		t.Fatal("login did not return token")
	}

	userInfoResponse, err := service.UserInfo(
		context.Background(),
		&user.UserInfoRequest{
			UserId: registerResponse.UserId,
			Token:  loginResponse.Token,
		},
	)
	if err != nil {
		t.Fatalf("get user info failed: %v", err)
	}

	if userInfoResponse.StatusCode != 0 {
		t.Fatalf(
			"get user info returned failure: %+v",
			userInfoResponse,
		)
	}

	if userInfoResponse.User == nil {
		t.Fatal("user info response does not contain user")
	}

	if userInfoResponse.User.Name != username {
		t.Fatalf(
			"username mismatch: want=%s got=%s",
			username,
			userInfoResponse.User.Name,
		)
	}

	if userInfoResponse.User.Avatar == "" {
		t.Fatal("avatar URL is empty")
	}

	if userInfoResponse.User.BackgroundImage == "" {
		t.Fatal("background image URL is empty")
	}
}
