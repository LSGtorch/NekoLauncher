//go:build !windows

package bindings

// applyAcrylicRuntime 非 Windows 平台暂无运行时切换窗口背景模糊的能力，
// 恒返回 false 让调用方走"写配置 + 重启进程"的旧路径。
func applyAcrylicRuntime(enabled bool) bool { return false }

// fixupAcrylicBackdrop 非 Windows 平台无 DWM，无需启动修正。
func fixupAcrylicBackdrop() {}

// watchWindowActivation — 已移除（见 acrylic_windows.go 的说明：实机对照测试
// 证明它并非"最小化后透明异常"的成因，且重设 margins -1 会画出原生标题栏按钮）。

// SetBackdropFrameEnabled 非 Windows 平台无 DWM 玻璃框架，空实现，
// 只为让 main.go 的调用在两端一致。
func SetBackdropFrameEnabled(bool) {}
