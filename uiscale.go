package main

import (
	"strconv"
	"strings"

	"nekolauncher/internal/config"
)

// 界面缩放的配置键。值为 "auto"（默认）或百分数（如 "85"）。
const uiScaleKey = "launcherUiScale"

// 手动档的取值范围与自动档下限（百分数）
const (
	uiScaleMinPercent = 50
	uiScaleMaxPercent = 150
	uiScaleAutoFloor  = 60
)

// resolveUiZoom 解析"界面缩放"设置，返回交给 WebView2 的缩放倍数（1 = 100%）。
//
// 为什么用 WebView2 的 ZoomFactor 而不是 CSS zoom：后者只把元素缩小画出来，
// 不改变布局视口——position: fixed 的遮罩、100vh 的容器都仍按原视口计算，
// 缩完就盖不满窗口（实测：界面不完整、遮罩缺一块）。ZoomFactor 是真正的页面
// 缩放，对渲染引擎等同于"窗口变大了"，fixed/vh 全部自然生效。
//
// 代价：wails 只在创建 webview 时读 ZoomFactor，所以**改动下次启动生效**。
func resolveUiZoom(windowWidth int, dpiScale float64) float64 {
	raw := strings.TrimSpace(config.GetValue(uiScaleKey))
	if raw == "" || raw == "auto" {
		return autoUiZoom(windowWidth, dpiScale)
	}

	percent, err := strconv.Atoi(raw)
	if err != nil || percent <= 0 {
		// 值损坏时回落自动档，而不是让界面卡在某个奇怪的倍率上
		return autoUiZoom(windowWidth, dpiScale)
	}

	return clampUiZoom(float64(percent) / 100)
}

// autoUiZoom 自动档：按窗口的 **CSS 像素**宽度决定倍数，视口窄于设计宽度就整体调小。
//
// windowWidth 是 wails 用的物理像素（见 dpiscale_windows.go），除以 DPI 才是
// 渲染时的 CSS 宽度。自动档只缩小、不放大——放大交给手动档，免得在大屏上把界面
// 莫名撑大。
func autoUiZoom(windowWidth int, dpiScale float64) float64 {
	cssWidth := float64(windowWidth)
	if dpiScale > 0 {
		cssWidth /= dpiScale
	}

	scale := cssWidth / float64(uiMinWidth)
	floor := float64(uiScaleAutoFloor) / 100

	if scale > 1 {
		return 1
	}
	if scale < floor {
		return floor
	}

	return scale
}

// clampUiZoom 把倍数收敛到手动档允许的范围
func clampUiZoom(zoom float64) float64 {
	minZoom := float64(uiScaleMinPercent) / 100
	maxZoom := float64(uiScaleMaxPercent) / 100

	if zoom < minZoom {
		return minZoom
	}
	if zoom > maxZoom {
		return maxZoom
	}

	return zoom
}
