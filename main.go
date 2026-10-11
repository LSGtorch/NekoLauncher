package main

import (
	"context"
	"embed"
	"math"
	"time"

	"nekolauncher/internal/bindings"
	"nekolauncher/internal/config"
	"nekolauncher/internal/logs"
	"nekolauncher/internal/solo"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

// UI 的设计尺寸（CSS 像素）。窗口视口不该窄于这个宽度，否则内容会被挤变形。
const (
	uiMinWidth  = 720
	uiMinHeight = 480
)

// scaleByDPI 把 CSS 像素的设计尺寸换算成 wails 需要的物理像素。
// dpiScale 为 1（非 Windows 或 96 DPI）时原样返回。
func scaleByDPI(value int, dpiScale float64) int {
	if dpiScale <= 1 {
		return value
	}

	return int(math.Round(float64(value) * dpiScale))
}

func main() {
	api := bindings.New()
	// 恢复上次保存的窗口尺寸（未保存过时用默认值 760×480）
	windowWidth, windowHeight := config.LoadLauncherWindowSize()
	// wails 在 Windows 上只声明了"系统级 DPI 感知"（SetProcessDPIAware），options.App
	// 里的尺寸因此按**物理像素**生效；而 UI 是按 CSS 像素设计的（uiMinWidth 就是那条
	// 设计线）。不换算的话，200% 缩放的屏幕上 720px 的最小窗口只等于 360 CSS 像素，
	// 内容被挤得"特别大"。
	//
	// 只把**最小尺寸**按 DPI 放大，再把尺寸往上夹到这条线——夹取是幂等的。
	// 不能对 LoadLauncherWindowSize 的结果直接乘：它存的是
	// wailsruntime.WindowGetSize 给的物理像素（见 config.SaveGlobalWindowSize），
	// 乘一次就会 1520 → 3040 逐次翻倍。初次安装时默认值 760×480 也会被这条线
	// 抬到设计宽度，效果与"把默认值乘 DPI"一致。
	dpiScale := systemDPIScale()
	minWidth := scaleByDPI(uiMinWidth, dpiScale)
	minHeight := scaleByDPI(uiMinHeight, dpiScale)
	if windowWidth < minWidth {
		windowWidth = minWidth
	}
	if windowHeight < minHeight {
		windowHeight = minHeight
	}
	backdrop := windows.None
	if config.AcrylicBackdropEnabled() {
		backdrop = windows.Acrylic // 该功能(亚克力背景)仅Windows有效。
	}
	bgR, bgG, bgB := startupBackgroundColour()
	// Linux 合成策略必须在 wails.Run 之前定好（WebKit 设置只在创建 webview 时生效）。
	// 不填 options.Linux 时 wails 会强制 Never —— 全部帧走软件合成，毛玻璃、
	// 滚动和场景壁纸一起掉帧，这就是 Linux 端"卡顿"的根因。
	linuxGpuPolicy := linux.WebviewGpuPolicyOnDemand
	if !config.LinuxGpuAccelerationEnabled() {
		linuxGpuPolicy = linux.WebviewGpuPolicyNever
	}
	// WebView2 的 GPU 开关同样只在创建时生效，关掉后界面走软件渲染（明显变卡），
	// 只留给 GPU 驱动有问题、开着就花屏/闪退的用户（设置页可切，需重启生效）。
	windowsGpuDisabled := !config.WindowsGpuAccelerationEnabled()

	err := wails.Run(&options.App{
		Title:     "NekoLauncher",
		Frameless: true,
		Width:     windowWidth,
		Height:    windowHeight,
		MinWidth:  minWidth,
		MinHeight: minHeight,
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: bindings.NewAssetFallbackHandler(),
		},
		BackgroundColour: &options.RGBA{R: bgR, G: bgG, B: bgB, A: 0},
		// 隐藏启动（治启动闪屏的关键）：窗口先不显示，等前端完成主题应用、
		// 配置 hydrate（窗口透明度/背景模式等）并渲染出首屏后再调
		// window.runtime.Show() 显示（见 frontend/src/layouts/index.tsx 的
		// WindowReveal）。此前窗口是"可见但空白"——用户会先看到透明/亚克力
		// 空窗，再看到 UI 弹出、透明度突变，全程都在闪。
		StartHidden: true,
		// 文件拖放：WebView2 的 HTML5 拖放拿不到磁盘路径（File.path 恒为空），
		// 必须走 Wails 原生 OnFileDrop 通道；EnableFileDrop 打开该通道，
		// DisableWebViewDrop 阻止 WebView 默认"打开拖入文件"的导航行为。
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: true,
		},
		OnStartup: func(ctx context.Context) {
			// NekoSolo 安装标记必须在实例扫描（api.Startup）之前消费：
			// 它会接管游戏目录 / 捆绑 Java / 选中实例，首启扫描要看到这些配置
			if err := solo.ApplyStartupDefaults(); err != nil {
				logs.Write("WARN", err.Error())
			}
			// 把 Wails runtime ctx 注入各 API（事件桥接依赖），并执行启动初始化
			api.Startup(ctx)
			// 常驻托盘：单击显示主窗口，菜单可显示/退出
			bindings.StartTray(ctx)
			// 兜底：前端若因异常始终没能调 Show（脚本崩溃/插件阻塞），到点强制
			// 显示窗口，避免应用"启动了但什么都看不到"。窗口已显示时再调一次
			// WindowShow 无副作用。
			time.AfterFunc(5*time.Second, func() {
				wailsruntime.WindowShow(ctx)
			})
		},
		OnBeforeClose: func(ctx context.Context) bool {
			// 按「关闭按钮行为」设置分发：询问（前端弹窗）/ 托盘 / 直接退出
			return bindings.HandleBeforeClose(ctx)
		},
		OnShutdown: func(context.Context) {
			// API关闭，同时关闭托盘
			api.Shutdown()
			bindings.StopTray()
		},
		Bind: []interface{}{
			api.Config, api.Launcher, api.Download, api.Account,
			api.Instance, api.World, api.Content, api.Modpack,
			api.Music, api.Monitor, api.Server, api.ServerHost,
			api.Online, api.System, api.Plugin, api.Update,
		},
		Linux: &linux.Options{
			WebviewGpuPolicy: linuxGpuPolicy,
		},
		Windows: &windows.Options{
			WindowIsTranslucent:  true,
			WebviewIsTransparent: true,
			WebviewGpuIsDisabled: windowsGpuDisabled,
			BackdropType:         backdrop,
			// 关闭 Wails 的默认窗口框架装饰：不关的话它会在每次 WM_ACTIVATE
			// （窗口获得/失去焦点的每次点击、Alt-Tab）调 DwmExtendFrameIntoClientArea
			// 重置 DWM 框架，与透明/亚克力 backdrop 材质冲突——正是"窗口操作就闪
			// 一下"的来源。圆角等外观由 bindings/acrylic_windows.go 的
			// fixupAcrylicBackdrop 自行补回（见该文件注释）。
			DisableFramelessWindowDecorations: true,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
