// Package hotreload 是配置热更：版本号单调、发布时快照拷贝、可回滚、发布幂等。
// 缺陷：原地改共享 map（读到半份配置）、不校验版本（旧版本覆盖新版本）、回滚不支持、
// 每次查询全扫历史。
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

// Publish 发布一份配置，返回是否生效与原因。
// 缺陷：直接改共享的 map（不是原子替换，调用方改自己那份也会影响线上）、
// 不校验版本号（旧版本照样覆盖）、重复发布同一版本也重复计数。
func (s *Store) Publish(version int, values map[string]int) (bool, string) {
	if s.current == nil {
		s.current = &Config{Version: version, Values: values}
	} else {
		s.current.Values = values
		s.current.Version = version
	}
	s.history = append(s.history, s.current)
	s.maxVersion = version
	s.published++
	return true, "ok"
}

// Current 是当前配置。缺陷：把内部对象直接给出去了。
func (s *Store) Current() *Config { return s.current }

// Rollback 回滚上一版。缺陷：没有实现。
func (s *Store) Rollback() bool { return false }

// Published 是发布成功次数。缺陷：每次全扫历史。
func (s *Store) Published() int {
	s.scanned += len(s.history)
	return s.published
}

// Scanned 是累计扫描的历史条数（规模观测）。
func (s *Store) Scanned() int { return s.scanned }
