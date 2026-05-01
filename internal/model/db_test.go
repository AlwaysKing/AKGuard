package model

import (
	"os"
	"testing"
)

func TestDB(t *testing.T) {
	// 使用临时数据库
	dbPath := t.TempDir() + "/test.db"
	if err := InitDB(dbPath); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer DB.Close()

	t.Run("ConfigGetSet", func(t *testing.T) {
		// 不存在的 key
		if v := ConfigGet("nonexistent"); v != "" {
			t.Errorf("ConfigGet(nonexistent) = %q, want empty", v)
		}

		// 写入再读取
		ConfigSet("test_key", "test_value")
		if v := ConfigGet("test_key"); v != "test_value" {
			t.Errorf("ConfigGet(test_key) = %q, want 'test_value'", v)
		}

		// 覆盖
		ConfigSet("test_key", "new_value")
		if v := ConfigGet("test_key"); v != "new_value" {
			t.Errorf("ConfigGet(test_key) after update = %q, want 'new_value'", v)
		}
	})

	t.Run("InternalNets", func(t *testing.T) {
		// 初始为空
		if nets := GetInternalNets(); nets != nil {
			t.Errorf("GetInternalNets() = %v, want nil", nets)
		}

		// 设置
		SetInternalNets([]string{"10.0.0.0/8", "192.168.0.0/16"})
		nets := GetInternalNets()
		if len(nets) != 2 {
			t.Fatalf("GetInternalNets() returned %d items, want 2", len(nets))
		}
		if nets[0] != "10.0.0.0/8" {
			t.Errorf("nets[0] = %q, want '10.0.0.0/8'", nets[0])
		}
		if nets[1] != "192.168.0.0/16" {
			t.Errorf("nets[1] = %q, want '192.168.0.0/16'", nets[1])
		}

		// 替换
		SetInternalNets([]string{"172.16.0.0/12"})
		nets = GetInternalNets()
		if len(nets) != 1 {
			t.Fatalf("GetInternalNets() after replace returned %d items, want 1", len(nets))
		}
		if nets[0] != "172.16.0.0/12" {
			t.Errorf("nets[0] = %q, want '172.16.0.0/12'", nets[0])
		}

		// 清空
		SetInternalNets([]string{})
		if nets := GetInternalNets(); nets != nil {
			t.Errorf("GetInternalNets() after clear = %v, want nil", nets)
		}
	})

	t.Run("DomainRules", func(t *testing.T) {
		// 初始为空
		if rules := GetDomainRules(); rules != nil {
			t.Errorf("GetDomainRules() = %v, want nil", rules)
		}

		// 设置
		SetDomainRules([]DomainRuleDB{
			{Host: "app.example.com", Internal: "pass", External: "auth"},
			{Host: "private.local", Internal: "pass", External: "reject"},
		})
		rules := GetDomainRules()
		if len(rules) != 2 {
			t.Fatalf("GetDomainRules() returned %d items, want 2", len(rules))
		}
		if rules[0].Host != "app.example.com" {
			t.Errorf("rules[0].Host = %q, want 'app.example.com'", rules[0].Host)
		}
		if rules[0].Internal != "pass" {
			t.Errorf("rules[0].Internal = %q, want 'pass'", rules[0].Internal)
		}
		if rules[0].External != "auth" {
			t.Errorf("rules[0].External = %q, want 'auth'", rules[0].External)
		}
		if rules[1].Host != "private.local" {
			t.Errorf("rules[1].Host = %q, want 'private.local'", rules[1].Host)
		}

		// 替换
		SetDomainRules([]DomainRuleDB{
			{Host: "new.example.com", Internal: "auth", External: "pass"},
		})
		rules = GetDomainRules()
		if len(rules) != 1 {
			t.Fatalf("GetDomainRules() after replace returned %d items, want 1", len(rules))
		}
		if rules[0].Host != "new.example.com" {
			t.Errorf("rules[0].Host = %q, want 'new.example.com'", rules[0].Host)
		}

		// 清空
		SetDomainRules([]DomainRuleDB{})
		if rules := GetDomainRules(); rules != nil {
			t.Errorf("GetDomainRules() after clear = %v, want nil", rules)
		}
	})

	t.Run("ConfigPersistence", func(t *testing.T) {
		// 写入一些配置
		ConfigSet("jwt_secret", "secret123")
		ConfigSet("admin_password", "hashed_admin")
		ConfigSet("auth_password", "hashed_auth")
		ConfigSet("default_internal_policy", "pass")
		ConfigSet("default_external_policy", "auth")
		SetInternalNets([]string{"10.0.0.0/8"})
		SetDomainRules([]DomainRuleDB{
			{Host: "test.local", Internal: "pass", External: "reject"},
		})

		// 关闭 DB
		DB.Close()

		// 重新打开
		if err := InitDB(dbPath); err != nil {
			t.Fatalf("InitDB reopen failed: %v", err)
		}

		// 验证所有数据都还在
		if v := ConfigGet("jwt_secret"); v != "secret123" {
			t.Errorf("jwt_secret after reopen = %q, want 'secret123'", v)
		}
		if v := ConfigGet("admin_password"); v != "hashed_admin" {
			t.Errorf("admin_password after reopen = %q, want 'hashed_admin'", v)
		}
		if v := ConfigGet("default_internal_policy"); v != "pass" {
			t.Errorf("default_internal_policy after reopen = %q, want 'pass'", v)
		}
		nets := GetInternalNets()
		if len(nets) != 1 || nets[0] != "10.0.0.0/8" {
			t.Errorf("internal_nets after reopen = %v, want [10.0.0.0/8]", nets)
		}
		rules := GetDomainRules()
		if len(rules) != 1 || rules[0].Host != "test.local" {
			t.Errorf("domain_rules after reopen = %v, want [test.local]", rules)
		}
	})

	// 清理
	os.Remove(dbPath)
}
