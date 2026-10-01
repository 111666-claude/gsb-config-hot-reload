// Package hotreload 是配置热更：版本号单调、发布时快照拷贝、可回滚、发布幂等。
package hotreload

import "sync"

// Config 是一份配置快照。
type Config struct {
	Version int
	Values  map[string]int
}

// Store 是配置仓库。history 是生效过的快照栈，栈顶即当前版本；
// maxVersion 是历史最高版本，回滚不降低它，保证旧版本永远被拒绝。
type Store struct {
	mu         sync.RWMutex
	history    []*Config
	maxVersion int
	published  int
	scanned    int
}

// NewStore 建仓库。
func NewStore() *Store { return &Store{} }

func cloneValues(values map[string]int) map[string]int {
	cloned := make(map[string]int, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

// Publish 发布一份配置。只有严格大于历史最高版本才生效；
// 相等返回 (false, "duplicate")，更小返回 (false, "stale")。
// 生效的是发布那一刻的拷贝，之后调用方改自己的 map 不影响线上。
func (s *Store) Publish(version int, values map[string]int) (bool, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.history) > 0 {
		switch {
		case version == s.maxVersion:
			return false, "duplicate"
		case version < s.maxVersion:
			return false, "stale"
		}
	}
	s.history = append(s.history, &Config{Version: version, Values: cloneValues(values)})
	s.maxVersion = version
	s.published++
	return true, "ok"
}

// Current 返回当前配置的独立拷贝：内部改动不影响返回值，
// 调用方改返回值也不影响仓库。没有配置时返回 nil。
func (s *Store) Current() *Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.history) == 0 {
		return nil
	}
	top := s.history[len(s.history)-1]
	return &Config{Version: top.Version, Values: cloneValues(top.Values)}
}

// Rollback 切回上一版并返回 true；没有上一版返回 false。
// 回滚不降低历史最高版本，之后仍拒绝旧版本发布。
func (s *Store) Rollback() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.history) < 2 {
		return false
	}
	s.history = s.history[:len(s.history)-1]
	return true
}

// Published 是发布成功次数。O(1) 计数，不扫历史。
func (s *Store) Published() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.published
}

// Scanned 是累计扫描的历史条数（规模观测）。
func (s *Store) Scanned() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.scanned
}
