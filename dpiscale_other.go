//go:build !windows

package main

// systemDPIScale 非 Windows 平台不做额外换算。
//
// 本文档只针对 Windows 的"系统级 DPI 感知 + 物理像素窗口尺寸"错配；
// Linux/macOS 下 wails 的窗口尺寸本身按逻辑像素处理，UI 不会因此被挤窄，
// 额外乘系数反而会放大窗口。详见 dpiscale_windows.go 的说明。
func systemDPIScale() float64 { return 1 }
