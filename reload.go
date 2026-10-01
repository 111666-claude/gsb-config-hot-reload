// Package hotreload 是配置热更：版本号单调、发布时快照拷贝、可回滚、发布幂等。
package hotreload

// Config 是一份配置快照。
type Config struct {
	Version int
	Values  map[string]int
}

// Store 是配置仓库。
type Store struct {
	current    *Config
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

// Publish 发布一份配置，返回是否生效与原因。
// 只有严格大于历史最高版本的发布才生效；相等返回 duplicate，更小返回 stale。
// 生效的配置是发布那一刻的拷贝，之后调用方改自己的 map 不影响线上。
func (s *Store) Publish(version int, values map[string]int) (bool, string) {
	if s.published > 0 {
		if version == s.maxVersion {
			return false, "duplicate"
		}
		if version < s.maxVersion {
			return false, "stale"
		}
	}
	snapshot := &Config{Version: version, Values: cloneValues(values)}
	s.current = snapshot
	s.history = append(s.history, snapshot)
	s.maxVersion = version
	s.published++
	return true, "ok"
}

// Current 是当前配置的拷贝，内部之后的改动不影响已返回的这份。
func (s *Store) Current() *Config {
	if s.current == nil {
		return nil
	}
	return &Config{Version: s.current.Version, Values: cloneValues(s.current.Values)}
}

// Rollback 回滚到上一版并返回 true；没有上一版返回 false。
// 回滚不改变历史最高版本，旧版本发布仍被拒绝。
func (s *Store) Rollback() bool {
	if len(s.history) < 2 {
		return false
	}
	s.history = s.history[:len(s.history)-1]
	s.current = s.history[len(s.history)-1]
	return true
}

// Published 是发布成功次数，O(1) 读取计数，不全扫历史。
func (s *Store) Published() int { return s.published }

// Scanned 是累计扫描的历史条数（规模观测）。
func (s *Store) Scanned() int { return s.scanned }
