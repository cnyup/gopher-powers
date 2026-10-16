package store

import (
	"path/filepath"
	"testing"
)

func TestAccountsLifecycle(t *testing.T) {
	dir := t.TempDir()
	a, err := Open(dir, "gp_guest")
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	if _, err := Open(dir, "bad_name"); err == nil {
		t.Fatal("非法 cookie 名应报错")
	}

	// create → find roundtrip。
	token, u, err := a.Create("测试玩家")
	if err != nil {
		t.Fatal(err)
	}
	if u.ID == "" || u.Name != "测试玩家" {
		t.Fatalf("user = %+v", u)
	}
	found := a.Find("gp_guest=" + token)
	if found == nil || found.ID != u.ID || found.Name != "测试玩家" {
		t.Fatalf("find 丢失: %+v", found)
	}
	if a.Find("gp_guest=wrong") != nil {
		t.Fatal("错 token 不应命中")
	}
	if a.Find("") != nil {
		t.Fatal("空 cookie 不应命中")
	}

	// rename。
	if err := a.Rename(u.ID, "新名字"); err != nil {
		t.Fatal(err)
	}
	if got := a.Find("gp_guest=" + token); got == nil || got.Name != "新名字" {
		t.Fatalf("rename 后 = %+v", got)
	}

	// settle：双人 win/loss 累计 + 幂等（同 battleId 只结算一次）。
	u2 := &User{ID: "u2", Name: "b"}
	if err := a.Settle("battle-1", [2]*User{u, u2}, "0"); err != nil {
		t.Fatal(err)
	}
	if err := a.Settle("battle-1", [2]*User{u, u2}, "0"); err != nil {
		t.Fatal(err) // 幂等不报错
	}
	got := a.Find("gp_guest=" + token)
	if got.Wins != 1 {
		t.Fatalf("wins = %d, want 1（幂等不重复计）", got.Wins)
	}
	if filepath.Base(filepath.Clean(dir)) == "" {
		t.Fatal("unreachable")
	}
}
