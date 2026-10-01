package hotreload

import (
	"math/rand"
	"reflect"
	"testing"
)

func TestFirstPublishSucceeds(t *testing.T) {
	ok, _ := NewStore().Publish(1, map[string]int{"tick": 30})
	if !ok {
		t.Fatal("第一份配置应该发布成功")
	}
}

func TestCurrentVersionAfterPublish(t *testing.T) {
	store := NewStore()
	store.Publish(3, map[string]int{"tick": 30})
	if store.Current().Version != 3 {
		t.Fatalf("当前版本不对：%d", store.Current().Version)
	}
}

func TestCurrentValuesAfterPublish(t *testing.T) {
	store := NewStore()
	store.Publish(1, map[string]int{"tick": 30})
	if store.Current().Values["tick"] != 30 {
		t.Fatalf("配置值不对：%v", store.Current().Values)
	}
}

func TestPublishedCountsOnce(t *testing.T) {
	store := NewStore()
	store.Publish(1, map[string]int{"tick": 30})
	if store.Published() != 1 {
		t.Fatalf("发布次数不对：%d", store.Published())
	}
}

func TestPublishRejectsDuplicateAndStale(t *testing.T) {
	store := NewStore()
	store.Publish(2, map[string]int{"tick": 30})
	if ok, reason := store.Publish(2, map[string]int{"tick": 10}); ok || reason != "duplicate" {
		t.Fatalf("重复版本应拒绝：ok=%v reason=%s", ok, reason)
	}
	if ok, reason := store.Publish(1, map[string]int{"tick": 10}); ok || reason != "stale" {
		t.Fatalf("旧版本应拒绝：ok=%v reason=%s", ok, reason)
	}
	if store.Current().Version != 2 {
		t.Fatalf("当前版本不应被旧版覆盖：%d", store.Current().Version)
	}
	if store.Published() != 1 {
		t.Fatalf("被拒绝的发布不应计数：%d", store.Published())
	}
}

func TestPublishSnapshotsValues(t *testing.T) {
	store := NewStore()
	values := map[string]int{"tick": 30}
	store.Publish(1, values)
	values["tick"] = 99
	if got := store.Current().Values["tick"]; got != 30 {
		t.Fatalf("发布后被调用方改动影响：%d", got)
	}
}

func TestCurrentReturnsIndependentCopy(t *testing.T) {
	store := NewStore()
	store.Publish(1, map[string]int{"tick": 30})
	store.Current().Values["tick"] = 99
	if got := store.Current().Values["tick"]; got != 30 {
		t.Fatalf("改 Current 返回值不应影响仓库：%d", got)
	}
}

func TestRollbackRestoresPreviousVersion(t *testing.T) {
	store := NewStore()
	store.Publish(1, map[string]int{"tick": 20})
	store.Publish(2, map[string]int{"tick": 30})
	if !store.Rollback() {
		t.Fatal("有上一版应该回滚成功")
	}
	current := store.Current()
	if current.Version != 1 || current.Values["tick"] != 20 {
		t.Fatalf("回滚后配置不对：%+v", current)
	}
}

func TestRollbackWithoutPreviousFails(t *testing.T) {
	store := NewStore()
	if store.Rollback() {
		t.Fatal("空仓库不该回滚成功")
	}
	store.Publish(1, map[string]int{"tick": 20})
	if store.Rollback() {
		t.Fatal("只有一版不该回滚成功")
	}
}

func TestRollbackStillRejectsOldVersions(t *testing.T) {
	store := NewStore()
	store.Publish(1, map[string]int{"tick": 20})
	store.Publish(2, map[string]int{"tick": 30})
	store.Rollback()
	if ok, reason := store.Publish(2, map[string]int{"tick": 40}); ok || reason != "duplicate" {
		t.Fatalf("回滚后重发最高版本应拒绝：ok=%v reason=%s", ok, reason)
	}
	if ok, reason := store.Publish(1, map[string]int{"tick": 40}); ok || reason != "stale" {
		t.Fatalf("回滚后更旧版本应拒绝：ok=%v reason=%s", ok, reason)
	}
	if ok, _ := store.Publish(3, map[string]int{"tick": 40}); !ok {
		t.Fatal("回滚后新版本应生效")
	}
}

func TestPublishedDoesNotScanHistory(t *testing.T) {
	store := NewStore()
	for version := 1; version <= 3000; version++ {
		store.Publish(version, map[string]int{"tick": version})
	}
	for query := 0; query < 3000; query++ {
		if store.Published() != 3000 {
			t.Fatalf("发布次数不对：%d", store.Published())
		}
	}
	if store.Scanned() > 3000 {
		t.Fatalf("扫描条数随历史线性增长：%d", store.Scanned())
	}
}

// refSnapshot 是参考实现的快照，刻意用最直白的写法对照 Store 的行为。
type refSnapshot struct {
	version int
	values  map[string]int
}

type reference struct {
	history    []refSnapshot
	maxVersion int
}

func (r *reference) publish(version int, values map[string]int) (bool, string) {
	if len(r.history) > 0 {
		if version == r.maxVersion {
			return false, "duplicate"
		}
		if version < r.maxVersion {
			return false, "stale"
		}
	}
	copied := make(map[string]int, len(values))
	for key, value := range values {
		copied[key] = value
	}
	r.history = append(r.history, refSnapshot{version: version, values: copied})
	r.maxVersion = version
	return true, "ok"
}

func (r *reference) rollback() bool {
	if len(r.history) < 2 {
		return false
	}
	r.history = r.history[:len(r.history)-1]
	return true
}

func TestDifferentialAgainstReference(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for trial := 0; trial < 200; trial++ {
		store := NewStore()
		ref := &reference{}
		for step := 0; step < 1+rng.Intn(50); step++ {
			switch rng.Intn(4) {
			case 0, 1, 2: // 发布：新 / 重复 / 旧版本
				version := rng.Intn(ref.maxVersion + 3)
				values := map[string]int{"tick": rng.Intn(100)}
				ok, reason := store.Publish(version, values)
				refOK, refReason := ref.publish(version, values)
				values["tick"] = -1 // 发布后再改调用方的 map，验证快照隔离
				if ok != refOK || reason != refReason {
					t.Fatalf("trial=%d step=%d 发布 v%d 结果不一致：(%v,%s) vs (%v,%s)",
						trial, step, version, ok, reason, refOK, refReason)
				}
			case 3: // 回滚
				if ok, refOK := store.Rollback(), ref.rollback(); ok != refOK {
					t.Fatalf("trial=%d step=%d 回滚结果不一致：%v vs %v", trial, step, ok, refOK)
				}
			}
			current := store.Current()
			if len(ref.history) == 0 {
				if current != nil {
					t.Fatalf("trial=%d 空仓库 Current 应为 nil：%+v", trial, current)
				}
				continue
			}
			top := ref.history[len(ref.history)-1]
			if current == nil || current.Version != top.version || !reflect.DeepEqual(current.Values, top.values) {
				t.Fatalf("trial=%d 当前配置不一致：%+v vs %+v", trial, current, top)
			}
		}
	}
}

func TestDeterministicSequences(t *testing.T) {
	run := func() []bool {
		store := NewStore()
		results := []bool{}
		for version := 1; version <= 100; version++ {
			ok, _ := store.Publish(version, map[string]int{"tick": version})
			results = append(results, ok)
			ok, _ = store.Publish(version, map[string]int{"tick": version})
			results = append(results, ok)
			ok, _ = store.Publish(version-1, map[string]int{"tick": version})
			results = append(results, ok)
			results = append(results, store.Rollback())
		}
		return results
	}
	first, second := run(), run()
	if !reflect.DeepEqual(first, second) {
		t.Fatal("同一串发布序列重复执行结果不同")
	}
}
