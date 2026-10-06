package service

import (
	"testing"

	"zzdzz-blog/server/internal/model"
)

func TestCanView(t *testing.T) {
	uid, otherID, legacy := uint64(1), uint64(2), uint64(0)
	author := func(id uint64) *uint64 { return &id }

	cases := []struct {
		name             string
		visibility       string
		articleAuthor    *uint64
		includeNonPublic bool
		owner            *uint64
		want             bool
	}{
		{"公开文章所有人可读", "public", author(1), false, nil, true},
		{"公开文章对带 owner 的登录用户可读", "public", author(1), true, &uid, true},
		{"访客读 private 被拒", "private", author(1), false, nil, false},
		{"访客读 draft 被拒", "draft", author(1), false, nil, false},
		{"admin 读 private 放行", "private", author(1), true, nil, true},
		{"admin 读 draft 放行", "draft", author(1), true, nil, true},
		{"作者读自己的 private 放行", "private", author(1), true, &uid, true},
		{"作者读自己的 draft 放行", "draft", author(1), true, &uid, true},
		{"非作者登录用户读他人 private 拒(修 IDOR)", "private", author(1), true, &otherID, false},
		{"非作者登录用户读他人 draft 拒(修 IDOR)", "draft", author(1), true, &otherID, false},
		{"无作者的 legacy 文章对非 admin 登录用户拒", "private", &legacy, true, &uid, false},
		{"无作者 legacy 文章访客拒", "draft", nil, false, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := &model.Article{Visibility: tc.visibility, AuthorID: tc.articleAuthor}
			got := canView(a, tc.includeNonPublic, tc.owner)
			if got != tc.want {
				t.Errorf("canView(%s, includeNonPublic=%v, owner=%v) = %v, want %v",
					tc.visibility, tc.includeNonPublic, tc.owner, got, tc.want)
			}
		})
	}
}
