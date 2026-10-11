//go:build windows

package main

import "syscall"

// systemDPIScale 返回系统 DPI 缩放倍数（96 DPI = 1.0，例如 200% 缩放返回 2）。
//
// 为什么需要它：wails 在 Windows 上只调用了 SetProcessDPIAware（系统级 DPI 感知，
// 见 wails 的 pkg/application/init_windows.go），因此 options.App 里给的窗口宽高
// 是按**物理像素**生效的。而整套 UI 是按 CSS 像素设计的——在 200% 缩放的屏幕上，
// 760px 的窗口只等于 380 CSS 像素，远窄于 UI 需要的宽度（MinWidth 720 就是那条
// 设计线），内容被挤得"特别大"。
//
// 把窗口尺寸乘上这个倍数，视口就始终等于设计所需的 CSS 宽度，UI 按原样排布。
//
// 只取系统 DPI 而不是逐显示器 DPI：SetProcessDPIAware 声明的是系统级感知，
// 窗口尺寸的单位口径与它一致，跟随它才不会出现口径错配。
func systemDPIScale() float64 {
	dpi, _, _ := syscall.NewLazyDLL("user32.dll").
		NewProc("GetDpiForSystem").Call()

	// 老系统（Win10 1607 之前）没有 GetDpiForSystem，返回 0：不缩放，
	// 行为与引入本函数之前完全一致。
	if dpi == 0 {
		return 1
	}

	scale := float64(dpi) / 96.0
	if scale < 1 {
		// 小于 96 DPI 的罕见情形不做反向缩小：那会让窗口小到不可用
		return 1
	}

	return scale
}
