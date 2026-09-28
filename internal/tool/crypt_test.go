package tool

import "testing"

func TestMd5Encrypt(t *testing.T) {
	result := Md5Encrypt("123456")

	expected := "e10adc3949ba59abbe56e057f20f883e"
	if result != expected {
		t.Fatalf("期望 %s，实际 %s", expected, result)
	}
}
