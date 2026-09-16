package handler

import (
	"testing"
	"time"

	jwtutil "zzdzz-blog/server/pkg/jwt"
)

func TestIsNoisePath(t *testing.T) {
	cases := []struct {
		path  string
		noise bool
	}{
		{"/", false},
		{"/api/v1/articles", false},
		{"/article/my-post", false},
		{"/api/v1/ping", true},
		{"/assets/index-a1b2c3.js", true},
		{"/favicon.ico", true},
		{"/favicon-32.png", true},
		{"/icons.svg", true},
		{"/robots.txt", true},
		{"/fonts/serif.woff2", true},
	}
	for _, c := range cases {
		if got := isNoisePath(c.path); got != c.noise {
			t.Errorf("isNoisePath(%q) = %v, want %v", c.path, got, c.noise)
		}
	}
}

func TestVisitUserID(t *testing.T) {
	secret := "test-secret-32-bytes-for-unit-test!"
	uid, _ := jwtutil.Generate(secret, time.Hour, 42, "alice", false)
	other, _ := jwtutil.Generate("another-secret-entirely-different!", time.Hour, 42, "alice", false)

	cases := []struct {
		name   string
		secret string
		header string
		want   uint64
		isNil  bool
	}{
		{"无 Authorization 头", secret, "", 0, true},
		{"非 Bearer 前缀", secret, "Basic abc", 0, true},
		{"合法 token", secret, "Bearer " + uid, 42, false},
		{"密钥不匹配", secret, "Bearer " + other, 0, true},
		{"垃圾 token", secret, "Bearer not-a-jwt", 0, true},
	}
	for _, c := range cases {
		got := visitUserID(c.secret, c.header)
		if c.isNil {
			if got != nil {
				t.Errorf("%s: 期望匿名(nil), 实得 %d", c.name, *got)
			}
			continue
		}
		if got == nil || *got != c.want {
			t.Errorf("%s: 期望 %d, 实得 %v", c.name, c.want, got)
		}
	}
}
