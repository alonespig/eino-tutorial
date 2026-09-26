# Go 编码规范

## 错误处理
- 禁止使用 `_` 忽略 error，确实不需要处理时必须写注释说明原因。
- 包装错误使用 `fmt.Errorf("xxx: %w", err)`，便于上层 errors.Is / errors.As 判断。

## 并发
- 所有 goroutine 必须能被 context 取消，禁止“野 goroutine”。
- 共享 map 必须加锁或使用 sync.Map。

## 日志
- 统一使用 slog，禁止使用 fmt.Println 打印业务日志。
- 日志中禁止输出手机号、身份证号、密码等敏感信息。
