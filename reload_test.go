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
