package hotreload

import "testing"

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

func TestDuplicateVersionRejected(t *testing.T) {
	store := NewStore()
	store.Publish(1, map[string]int{"tick": 30})
	ok, reason := store.Publish(1, map[string]int{"tick": 40})
	if ok || reason != "duplicate" {
		t.Fatalf("重复发布应被拒绝：ok=%v reason=%s", ok, reason)
	}
	if store.Current().Values["tick"] != 30 {
		t.Fatalf("重复发布不得覆盖线上配置：%v", store.Current().Values)
	}
}

func TestStaleVersionRejected(t *testing.T) {
	store := NewStore()
	store.Publish(2, map[string]int{"tick": 30})
	ok, reason := store.Publish(1, map[string]int{"tick": 10})
	if ok || reason != "stale" {
		t.Fatalf("旧版本发布应被拒绝：ok=%v reason=%s", ok, reason)
	}
	if store.Current().Version != 2 || store.Current().Values["tick"] != 30 {
		t.Fatalf("旧版本发布不得覆盖线上配置：%+v", store.Current())
	}
}

func TestStrictlyGreaterVersionTakesEffect(t *testing.T) {
	store := NewStore()
	store.Publish(5, map[string]int{"tick": 30})
	if ok, _ := store.Publish(6, map[string]int{"tick": 40}); !ok {
		t.Fatal("更大版本应该发布成功")
	}
	if store.Current().Version != 6 || store.Current().Values["tick"] != 40 {
		t.Fatalf("新版本未生效：%+v", store.Current())
	}
}

func TestPublishSnapshotsCallerMap(t *testing.T) {
	store := NewStore()
	values := map[string]int{"tick": 30}
	store.Publish(1, values)
	values["tick"] = 99
	if store.Current().Values["tick"] != 30 {
		t.Fatalf("调用方改 map 不应影响线上：%v", store.Current().Values)
	}
}

func TestCurrentReturnedConfigIsIsolated(t *testing.T) {
	store := NewStore()
	store.Publish(1, map[string]int{"tick": 30})
	first := store.Current()
	first.Values["tick"] = 99
	first.Version = 99
	if store.Current().Version != 1 || store.Current().Values["tick"] != 30 {
		t.Fatalf("Current 返回的配置被内部改动牵连：%+v", store.Current())
	}
	store.Publish(2, map[string]int{"tick": 40})
	if first.Version != 99 || first.Values["tick"] != 99 {
		t.Fatalf("内部新版本不应改动已返回的配置：%+v", first)
	}
}

func TestRollback(t *testing.T) {
	store := NewStore()
	store.Publish(1, map[string]int{"tick": 20})
	store.Publish(2, map[string]int{"tick": 30})
	if !store.Rollback() {
		t.Fatal("有上一版时回滚应返回 true")
	}
	current := store.Current()
	if current.Version != 1 || current.Values["tick"] != 20 {
		t.Fatalf("回滚后配置不对：%+v", current)
	}
}

func TestRollbackWithoutHistoryReturnsFalse(t *testing.T) {
	if NewStore().Rollback() {
		t.Fatal("没有上一版应返回 false")
	}
	store := NewStore()
	store.Publish(1, map[string]int{"tick": 30})
	if store.Rollback() {
		t.Fatal("只有一版时回滚应返回 false")
	}
}

func TestRejectStaleAfterRollback(t *testing.T) {
	store := NewStore()
	store.Publish(1, map[string]int{"tick": 20})
	store.Publish(2, map[string]int{"tick": 30})
	store.Rollback()
	if ok, reason := store.Publish(2, map[string]int{"tick": 40}); ok || reason != "duplicate" {
		t.Fatalf("回滚后重复历史版本应拒绝：ok=%v reason=%s", ok, reason)
	}
	if ok, reason := store.Publish(1, map[string]int{"tick": 40}); ok || reason != "stale" {
		t.Fatalf("回滚后旧版本发布应拒绝：ok=%v reason=%s", ok, reason)
	}
	if ok, _ := store.Publish(3, map[string]int{"tick": 40}); !ok {
		t.Fatal("回滚后更大版本应发布成功")
	}
	if store.Current().Version != 3 {
		t.Fatalf("新版本未生效：%d", store.Current().Version)
	}
}

func TestRepeatedSequenceDeterministic(t *testing.T) {
	run := func() (int, int, int) {
		store := NewStore()
		store.Publish(1, map[string]int{"tick": 1})
		store.Publish(1, map[string]int{"tick": 2})
		store.Publish(0, map[string]int{"tick": 3})
		store.Publish(2, map[string]int{"tick": 4})
		store.Rollback()
		store.Publish(3, map[string]int{"tick": 5})
		return store.Published(), store.Current().Version, store.Current().Values["tick"]
	}
	a1, a2, a3 := run()
	b1, b2, b3 := run()
	if a1 != b1 || a2 != b2 || a3 != b3 {
		t.Fatalf("同一序列结果不一致：(%d,%d,%d) vs (%d,%d,%d)", a1, a2, a3, b1, b2, b3)
	}
}

func TestPublishedQueryDoesNotScanHistory(t *testing.T) {
	store := NewStore()
	for index := 1; index <= 3000; index++ {
		store.Publish(index, map[string]int{"tick": index})
	}
	for query := 0; query < 3000; query++ {
		if store.Published() != 3000 {
			t.Fatal("Published 应返回成功发布次数")
		}
	}
	if scanned := store.Scanned(); scanned > 3000 {
		t.Fatalf("Scanned 不应随历史条数线性增长：%d", scanned)
	}
}

// refStore 是「版本校验 + 快照拷贝 + 回滚」的参考实现。
type refStore struct {
	history    []refConfig
	maxVersion int
	published  int
}

type refConfig struct {
	version int
	values  map[string]int
}

func (r *refStore) publish(version int, values map[string]int) (bool, string) {
	if r.published > 0 {
		switch {
		case version == r.maxVersion:
			return false, "duplicate"
		case version < r.maxVersion:
			return false, "stale"
		}
	}
	snapshot := map[string]int{}
	for key, value := range values {
		snapshot[key] = value
	}
	r.history = append(r.history, refConfig{version, snapshot})
	r.maxVersion = version
	r.published++
	return true, "ok"
}

func (r *refStore) current() (int, map[string]int) {
	if len(r.history) == 0 {
		return 0, nil
	}
	top := r.history[len(r.history)-1]
	cloned := map[string]int{}
	for key, value := range top.values {
		cloned[key] = value
	}
	return top.version, cloned
}

func (r *refStore) rollback() bool {
	if len(r.history) < 2 {
		return false
	}
	r.history = r.history[:len(r.history)-1]
	return true
}

func TestDifferentialAgainstReference(t *testing.T) {
	const rounds = 200
	for round := 0; round < rounds; round++ {
		seed := int64(round*1000003 + 17)
		rng := func(n int64) int64 {
			seed = seed*6364136223846793005 + 1442695040888963407
			return ((seed>>33)%n + n) % n
		}

		store := NewStore()
		ref := &refStore{}
		nextVersion := 1
		compare := func(stage string) {
			t.Helper()
			gotVersion := 0
			var gotValues map[string]int
			if current := store.Current(); current != nil {
				gotVersion = current.Version
				gotValues = current.Values
			}
			wantVersion, wantValues := ref.current()
			if gotVersion != wantVersion || store.Published() != ref.published {
				t.Fatalf("round %d %s：版本/次数不一致 got=(%d,%d) want=(%d,%d)",
					round, stage, gotVersion, store.Published(), wantVersion, ref.published)
			}
			if len(gotValues) != len(wantValues) {
				t.Fatalf("round %d %s：配置条目数不一致 %v vs %v", round, stage, gotValues, wantValues)
			}
			for key, value := range wantValues {
				if gotValues[key] != value {
					t.Fatalf("round %d %s：配置内容不一致 got=%v want=%v", round, stage, gotValues, wantValues)
				}
			}
		}

		for step := 0; step < 30; step++ {
			values := map[string]int{"tick": int(rng(1000))}
			switch rng(4) {
			case 0: // 发布新版本
				gotOK, gotReason := store.Publish(nextVersion, values)
				wantOK, wantReason := ref.publish(nextVersion, values)
				nextVersion++
				if gotOK != wantOK || gotReason != wantReason {
					t.Fatalf("round %d step %d：发布结果不一致 got=(%v,%s) want=(%v,%s)",
						round, step, gotOK, gotReason, wantOK, wantReason)
				}
			case 1: // 重复发布当前最高版本
				target := ref.maxVersion
				gotOK, gotReason := store.Publish(target, values)
				wantOK, wantReason := ref.publish(target, values)
				if gotOK != wantOK || gotReason != wantReason {
					t.Fatalf("round %d step %d：重复发布结果不一致 got=(%v,%s) want=(%v,%s)",
						round, step, gotOK, gotReason, wantOK, wantReason)
				}
			case 2: // 发布旧版本
				target := 1
				if ref.maxVersion > 1 {
					target = int(rng(int64(ref.maxVersion)))
				}
				gotOK, gotReason := store.Publish(target, values)
				wantOK, wantReason := ref.publish(target, values)
				if gotOK != wantOK || gotReason != wantReason {
					t.Fatalf("round %d step %d：旧版本发布结果不一致 got=(%v,%s) want=(%v,%s)",
						round, step, gotOK, gotReason, wantOK, wantReason)
				}
			case 3: // 回滚
				got := store.Rollback()
				if want := ref.rollback(); got != want {
					t.Fatalf("round %d step %d：回滚结果不一致 got=%v want=%v", round, step, got, want)
				}
			}
			compare("step")
		}
	}
}

func TestScaleHundredThousandPublishes(t *testing.T) {
	store := NewStore()
	const publishes = 100000
	for index := 1; index <= publishes; index++ {
		ok, reason := store.Publish(index, map[string]int{"tick": index})
		if !ok || reason != "ok" {
			t.Fatalf("第 %d 次发布失败：%v %s", index, ok, reason)
		}
	}
	for query := 0; query < publishes; query++ {
		if store.Published() != publishes {
			t.Fatal("Published 应始终返回发布总数")
		}
	}
	if store.Scanned() != 0 {
		t.Fatalf("查询不得扫描历史，Scanned=%d", store.Scanned())
	}
	if store.Current().Version != publishes || store.Current().Values["tick"] != publishes {
		t.Fatalf("当前配置不对：%+v", store.Current())
	}
}
