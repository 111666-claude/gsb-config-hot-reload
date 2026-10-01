# config-hot-reload

配置热更：版本号单调、发布时快照拷贝、可回滚、发布幂等，只用 Go 标准库。

```
go test ./...
go run ./cmd/config-hot-reload --sample stale
go run ./cmd/config-hot-reload --sample atomic
go run ./cmd/config-hot-reload --sample rollback
```

## 口径（README 为准）

- **版本单调**：只有严格大于当前版本的发布才生效；相等（重复发布）或更小返回
  `(false, "duplicate")`/`(false, "stale")`。
- **发布即快照**：生效的配置是发布那一刻的拷贝，调用方之后修改自己的 map 不影响线上。
- **可回滚**：`Rollback` 切回上一版并返回 `true`；没有上一版返回 `false`；
  回滚之后仍以历史最高版本为准拒绝旧版本发布。
- **查询代价**：`Published` 不许每次全扫历史，`Scanned` 不随历史条数线性增长。
- **规模**：单服 10 万次发布、每秒 10 万次查询，单次摊还 O(1)，内存 O(历史条数)。

## 输出契约（不改格式）

```
ok=false reason=stale version=2
tick=30
ok=true version=1
scanned<=3000
```

## 目录

```
reload.go               配置发布与回滚
cmd/config-hot-reload   命令行入口
reload_test.go          go test 用例
```
