package test

import (
	"errors"
	flow "github.com/Bilibotter/light-flow"
	plugins "github.com/Bilibotter/light-flow-plugins"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"sync/atomic"
	"testing"
)

func TestRecover(t *testing.T) {
	flow.SetEncryptor(flow.NewAES256Encryptor([]byte("secret")))
	suc := int64(0)
	count := int64(0)
	db0, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failureed to open database: %v", err)
	}
	if err = plugins.NewSuspendPlugin(db0).InjectSuspend(); err != nil {
		t.Logf("Error injecting persistence: %v", err)
		return
	}
	wf := flow.RegisterFlow("TestRecover")
	wf.EnableRecover()
	proc := wf.Process("TestRecover")
	proc.NameStep(func(ctx flow.Step) (any, error) {
		t.Logf("Step[%s] start", ctx.Name())
		if atomic.LoadInt64(&suc) == 0 {
			ctx.Set("hello", "world")
			atomic.StoreInt64(&suc, 1)
			t.Logf("Step[%s] failed", ctx.Name())
			atomic.AddInt64(&count, 1)
			return nil, errors.New("execute error")
		}
		atomic.AddInt64(&count, 1)
		t.Logf("Step[%s] succeed", ctx.Name())
		return nil, nil
	}, "1")
	proc.NameStep(func(ctx flow.Step) (any, error) {
		t.Logf("Step[%s] start", ctx.Name())
		if key, exist := ctx.Get("hello"); !exist {
			t.Errorf("Step[%s] failed, key not found", ctx.Name())
			return nil, errors.New("key not found")
		} else if key != "world" {
			t.Errorf("Step[%s] failed, key not match, expected: world, actual: %s", ctx.Name(), key)
		}
		atomic.AddInt64(&count, 1)
		t.Logf("Step[%s] succeed", ctx.Name())
		return nil, nil
	}, "2", "1")
	ff := flow.DoneFlow("TestRecover", nil)
	if atomic.LoadInt64(&count) != 1 {
		t.Errorf("TestRecover failed, count: %d, expected: 1", count)
	}
	if ff, err = ff.Recover(); err != nil {
		t.Errorf("Failed to recover flow: %s", err.Error())
	} else if !ff.Success() {
		t.Errorf("Flow should succeed, but failed")
	}
	if atomic.LoadInt64(&count) != 3 {
		t.Errorf("TestRecover failed, count: %d, expected: 3", count)
	}
}
