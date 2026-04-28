package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// 配置文件名
const configFileName = "sitlong-config.json"

// ShortcutConfig 快捷键配置
type ShortcutConfig struct {
	ResetKey       string `json:"resetKey"`       // 重置键
	StartKey       string `json:"startKey"`       // 开始键
	CloseNotifyKey string `json:"closeNotifyKey"` // 关闭通知键
	Ctrl           bool   `json:"ctrl"`           // 是否使用Ctrl
	Shift          bool   `json:"shift"`          // 是否使用Shift
	Alt            bool   `json:"alt"`            // 是否使用Alt
	Global         bool   `json:"global"`         // 是否全局热键
}

// 默认快捷键配置
func defaultShortcutConfig() ShortcutConfig {
	return ShortcutConfig{
		ResetKey:       "r",
		StartKey:       "s",
		CloseNotifyKey: "Escape",
		Ctrl:           true,
		Shift:          true,
		Alt:            false,
		Global:         true,
	}
}

// Settings 应用设置
type Settings struct {
	Interval               int              `json:"interval"`               // 提醒间隔（分钟）
	Shortcut              ShortcutConfig   `json:"shortcut"`              // 快捷键配置
	NotificationDuration int              `json:"notificationDuration"`   // 通知持续时间（秒），0表示持久
	ActivateOnTimer       bool             `json:"activateOnTimer"`       // 时间到时激活窗口
	Message               string           `json:"message"`               // 提醒文案
	LoopMode              bool             `json:"loopMode"`              // 循环模式
}

// SitLongConfig 前端配置结构
type SitLongConfig struct {
	Interval              int              `json:"interval"`
	IsRunning            bool             `json:"isRunning"`
	Remaining            int              `json:"remaining"`
	Shortcut             ShortcutConfig    `json:"shortcut"`
	NotificationDuration int              `json:"notificationDuration"`
	ActivateOnTimer      bool             `json:"activateOnTimer"`
	Message               string           `json:"message"`
	LoopMode             bool             `json:"loopMode"`
}

// App struct
type App struct {
	ctx          context.Context
	timer       *time.Timer
	mu           sync.Mutex
	settings     Settings
	remaining    time.Duration
	isRunning    bool
	lastTickTime time.Time
	configDir    string
	appMenu     *menu.Menu
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		settings: Settings{
			Interval:             30,
			Shortcut:             defaultShortcutConfig(),
			NotificationDuration: 0, // 默认持久显示
			ActivateOnTimer:      true, // 默认激活窗口
			Message:              "您已经坐了很久了，起来活动一下吧！", // 默认提醒文案
			LoopMode:             false, // 默认关闭循环模式
		},
		remaining:    30 * time.Minute,
		isRunning:    false,
	}
}

// buildResetAccelerator 根据快捷键配置构建加速器（重置）
func (a *App) buildResetAccelerator() string {
	var mods []string
	sc := a.settings.Shortcut

	if sc.Ctrl {
		mods = append(mods, "ctrl")
	}
	if sc.Shift {
		mods = append(mods, "shift")
	}
	if sc.Alt {
		mods = append(mods, "alt")
	}

	key := strings.ToLower(sc.ResetKey)
	if len(mods) > 0 {
		return strings.Join(mods, "+") + "+" + key
	}
	return key
}

// buildStartAccelerator 根据快捷键配置构建加速器（开始）
func (a *App) buildStartAccelerator() string {
	var mods []string
	sc := a.settings.Shortcut

	if sc.Ctrl {
		mods = append(mods, "ctrl")
	}
	if sc.Shift {
		mods = append(mods, "shift")
	}
	if sc.Alt {
		mods = append(mods, "alt")
	}

	key := strings.ToLower(sc.StartKey)
	if len(mods) > 0 {
		return strings.Join(mods, "+") + "+" + key
	}
	return key
}

// updateMenu 更新菜单（包括全局热键）
func (a *App) updateMenu() {
	if a.ctx == nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	appMenu := menu.NewMenu()
	operationMenu := appMenu.AddSubmenu("操作")

	// 重置计时
	if a.settings.Shortcut.Global {
		accelerator := a.buildResetAccelerator()
		accel := keys.Key(accelerator)
		operationMenu.AddText(fmt.Sprintf("重置计时 %s", accelerator), accel, func(_ *menu.CallbackData) {
			a.ResetTimer()
		})
	} else {
		operationMenu.AddText("重置计时", nil, func(_ *menu.CallbackData) {
			a.ResetTimer()
		})
	}

	// 开始计时
	if a.settings.Shortcut.Global {
		accelerator := a.buildStartAccelerator()
		accel := keys.Key(accelerator)
		operationMenu.AddText(fmt.Sprintf("开始计时 %s", accelerator), accel, func(_ *menu.CallbackData) {
			a.StartTimer()
		})
	} else {
		operationMenu.AddText("开始计时", nil, func(_ *menu.CallbackData) {
			a.StartTimer()
		})
	}

	operationMenu.AddText("暂停计时", keys.Key("p"), func(_ *menu.CallbackData) {
		a.PauseTimer()
	})

	// 帮助菜单
	helpMenu := appMenu.AddSubmenu("帮助")

	helpMenu.AddText("关于软件", nil, func(_ *menu.CallbackData) {
		wailsRuntime.EventsEmit(a.ctx, "show-about")
	})

	helpMenu.AddText("软件介绍", nil, func(_ *menu.CallbackData) {
		wailsRuntime.BrowserOpenURL(a.ctx, "https://www.haogongjua.cn/cms/sitlong/")
	})

	helpMenu.AddText("在线工具", nil, func(_ *menu.CallbackData) {
		wailsRuntime.BrowserOpenURL(a.ctx, "https://www.haogongjua.cn/")
	})

	a.appMenu = appMenu
	wailsRuntime.MenuSetApplicationMenu(a.ctx, appMenu)
}

// GetVersion 获取软件版本号
func (a *App) GetVersion() string {
	return AppVersion
}

// startup is called when the app starts. The context is saved
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// 获取配置目录
	var err error
	a.configDir, err = a.getConfigDir()
	if err != nil {
		wailsRuntime.LogError(ctx, fmt.Sprintf("无法获取配置目录: %v", err))
		a.configDir = "."
	}

	wailsRuntime.LogInfo(ctx, fmt.Sprintf("配置目录: %s", a.configDir))
	wailsRuntime.LogInfo(ctx, fmt.Sprintf("完整配置文件路径: %s", a.getConfigPath()))

	// 加载配置
	a.loadSettings()

	// 更新菜单（包括全局热键）
	a.updateMenu()

	wailsRuntime.LogInfo(ctx, "应用启动完成")
}

// getConfigDir 获取配置目录
func (a *App) getConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	var configDir string
	switch runtime.GOOS {
	case "darwin":
		configDir = filepath.Join(home, "Library", "Application Support", "SitLong")
	case "windows":
		configDir = filepath.Join(os.Getenv("APPDATA"), "SitLong")
	default: // linux
		configDir = filepath.Join(home, ".config", "sitlong")
	}

	// 确保目录存在
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return configDir, nil
}

// getConfigPath 获取完整配置文件路径
func (a *App) getConfigPath() string {
	return filepath.Join(a.configDir, configFileName)
}

// loadSettings 从文件加载设置
func (a *App) loadSettings() {
	configPath := a.getConfigPath()
	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, fmt.Sprintf("正在加载配置文件: %s", configPath))
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			if a.ctx != nil {
				wailsRuntime.LogInfo(a.ctx, "配置文件不存在，使用默认配置并创建文件")
			}
			// 创建默认配置文件（此时 ctx 可能为 nil）
			a.saveSettings()
		} else {
			if a.ctx != nil {
				wailsRuntime.LogError(a.ctx, fmt.Sprintf("读取配置文件失败: %v", err))
			}
		}
		return
	}

	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, fmt.Sprintf("配置文件内容: %s", string(data)))
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		if a.ctx != nil {
			wailsRuntime.LogError(a.ctx, fmt.Sprintf("解析配置失败: %v", err))
		}
		return
	}

	a.mu.Lock()
	a.settings = settings
	a.remaining = time.Duration(settings.Interval) * time.Minute
	a.mu.Unlock()

	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, fmt.Sprintf("配置加载成功: interval=%d, shortcut=%v",
			settings.Interval, settings.Shortcut))
	}
}

// saveSettings 保存设置到文件（调用方必须持有锁，且 ctx 可能为 nil）
func (a *App) saveSettings() error {
	settings := a.settings

	configPath := a.getConfigPath()
	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, fmt.Sprintf("正在保存配置到: %s", configPath))
		wailsRuntime.LogInfo(a.ctx, fmt.Sprintf("配置内容: interval=%d, shortcut_reset=%s",
			settings.Interval, settings.Shortcut.ResetKey))
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		if a.ctx != nil {
			wailsRuntime.LogError(a.ctx, fmt.Sprintf("序列化配置失败: %v", err))
		}
		return err
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		if a.ctx != nil {
			wailsRuntime.LogError(a.ctx, fmt.Sprintf("写入配置文件失败: %v", err))
		}
		return err
	}

	// 验证写入
	verifyData, err := os.ReadFile(configPath)
	if err != nil {
		if a.ctx != nil {
			wailsRuntime.LogError(a.ctx, fmt.Sprintf("验证读取失败: %v", err))
		}
	} else {
		if a.ctx != nil {
			wailsRuntime.LogInfo(a.ctx, fmt.Sprintf("配置保存成功，文件大小: %d bytes", len(verifyData)))
		}
	}

	return nil
}

// GetConfig 获取当前配置
func (a *App) GetConfig() SitLongConfig {
	a.mu.Lock()
	defer a.mu.Unlock()

	config := SitLongConfig{
		Interval:              a.settings.Interval,
		IsRunning:            a.isRunning,
		Remaining:            int(a.remaining.Seconds()),
		Shortcut:             a.settings.Shortcut,
		NotificationDuration: a.settings.NotificationDuration,
		ActivateOnTimer:      a.settings.ActivateOnTimer,
		Message:              a.settings.Message,
		LoopMode:             a.settings.LoopMode,
	}

	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, fmt.Sprintf("GetConfig: interval=%d, isRunning=%v, remaining=%d",
			config.Interval, config.IsRunning, config.Remaining))
	}

	return config
}

// SetInterval 设置提醒间隔
func (a *App) SetInterval(minutes int) error {
	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, fmt.Sprintf("=== SetInterval 被调用: minutes=%d ===", minutes))
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if minutes < 1 {
		if a.ctx != nil {
			wailsRuntime.LogError(a.ctx, "SetInterval: 间隔时间不能小于1分钟")
		}
		return fmt.Errorf("间隔时间不能小于1分钟")
	}

	oldInterval := a.settings.Interval
	oldRemaining := a.remaining

	a.settings.Interval = minutes
	if !a.isRunning {
		a.remaining = time.Duration(minutes) * time.Minute
		if a.ctx != nil {
			wailsRuntime.LogInfo(a.ctx, fmt.Sprintf("SetInterval: 未运行中，更新remaining %v -> %v",
				oldRemaining, a.remaining))
		}
	} else {
		if a.ctx != nil {
			wailsRuntime.LogInfo(a.ctx, fmt.Sprintf("SetInterval: 运行中，只更新interval %d -> %d, remaining保持不变=%v",
				oldInterval, minutes, a.remaining))
		}
	}

	err := a.saveSettings()
	if err != nil {
		if a.ctx != nil {
			wailsRuntime.LogError(a.ctx, fmt.Sprintf("SetInterval: 保存失败 %v", err))
		}
		return err
	}

	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, fmt.Sprintf("SetInterval: 保存成功"))
	}
	return nil
}

// SetShortcut 设置快捷键
func (a *App) SetShortcut(config ShortcutConfig) error {
	a.mu.Lock()
	a.settings.Shortcut = config
	err := a.saveSettings()
	a.mu.Unlock()

	// 更新菜单（全局热键可能改变）
	a.updateMenu()

	return err
}

// SetNotificationDuration 设置通知持续时间
func (a *App) SetNotificationDuration(seconds int) error {
	a.mu.Lock()
	a.settings.NotificationDuration = seconds
	err := a.saveSettings()
	a.mu.Unlock()

	return err
}

// SetActivateOnTimer 设置时间到时是否激活窗口
func (a *App) SetActivateOnTimer(activate bool) error {
	a.mu.Lock()
	a.settings.ActivateOnTimer = activate
	err := a.saveSettings()
	a.mu.Unlock()

	return err
}

// SetMessage 设置提醒文案
func (a *App) SetMessage(message string) error {
	a.mu.Lock()
	a.settings.Message = message
	err := a.saveSettings()
	a.mu.Unlock()

	return err
}

// SetLoopMode 设置循环模式
func (a *App) SetLoopMode(loopMode bool) error {
	a.mu.Lock()
	a.settings.LoopMode = loopMode
	err := a.saveSettings()
	a.mu.Unlock()

	return err
}

// beforeClose 关闭前清理
func (a *App) beforeClose(ctx context.Context) bool {
	a.mu.Lock()
	if a.timer != nil {
		a.timer.Stop()
	}
	a.mu.Unlock()
	return false // 允许关闭
}

// StartTimer 开始计时
func (a *App) StartTimer() error {
	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, "=== StartTimer 被调用 ===")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, fmt.Sprintf("当前状态: isRunning=%v, remaining=%v, interval=%d",
			a.isRunning, a.remaining, a.settings.Interval))
	}

	if a.isRunning {
		if a.ctx != nil {
			wailsRuntime.LogInfo(a.ctx, "StartTimer: 已经在运行中，直接返回")
		}
		return nil
	}

	if a.remaining <= 0 {
		oldRemaining := a.remaining
		a.remaining = time.Duration(a.settings.Interval) * time.Minute
		if a.ctx != nil {
			wailsRuntime.LogInfo(a.ctx, fmt.Sprintf("StartTimer: remaining(%v)<=0，重置为 %v (interval=%d分钟)",
				oldRemaining, a.remaining, a.settings.Interval))
		}
	}

	a.isRunning = true
	a.lastTickTime = time.Now()

	a.timer = time.AfterFunc(a.remaining, a.onTimerComplete)

	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, fmt.Sprintf("计时器已启动: remaining=%v, lastTickTime=%v",
			a.remaining, a.lastTickTime))
	}
	return nil
}

// PauseTimer 暂停计时
func (a *App) PauseTimer() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.isRunning {
		return nil
	}

	elapsed := time.Since(a.lastTickTime)
	a.remaining -= elapsed

	if a.timer != nil {
		a.timer.Stop()
	}

	a.isRunning = false

	return nil
}

// ResetTimer 重置计时器
func (a *App) ResetTimer() error {
	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, "=== ResetTimer 被调用 ===")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, fmt.Sprintf("重置前状态: isRunning=%v, remaining=%v, interval=%d",
			a.isRunning, a.remaining, a.settings.Interval))
	}

	if a.timer != nil {
		stopped := a.timer.Stop()
		if a.ctx != nil {
			wailsRuntime.LogInfo(a.ctx, fmt.Sprintf("旧计时器停止: %v", stopped))
		}
	}

	oldRemaining := a.remaining
	a.remaining = time.Duration(a.settings.Interval) * time.Minute
	a.isRunning = false

	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, fmt.Sprintf("重置 remaining: %v -> %v (interval=%d分钟)",
			oldRemaining, a.remaining, a.settings.Interval))
	}

	a.isRunning = true
	a.lastTickTime = time.Now()
	a.timer = time.AfterFunc(a.remaining, a.onTimerComplete)

	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, fmt.Sprintf("重置完成: isRunning=true, remaining=%v, timer已启动",
			a.remaining))
	}

	wailsRuntime.EventsEmit(a.ctx, "timer-reset", a.remaining.Seconds())
	return nil
}

// GetRemainingTime 获取剩余时间
func (a *App) GetRemainingTime() int {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.isRunning {
		elapsed := time.Since(a.lastTickTime)
		return int((a.remaining - elapsed).Seconds())
	}

	return int(a.remaining.Seconds())
}

// TestOneMinute 专门测试1分钟间隔
func (a *App) TestOneMinute() string {
	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, "=== TestOneMinute 开始 ===")
	}

	// 先停止任何运行中的计时器
	a.mu.Lock()
	if a.timer != nil {
		a.timer.Stop()
	}
	a.isRunning = false
	a.remaining = 0
	a.mu.Unlock()

	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, "TestOneMinute: 已清理状态")
	}

	// 设置间隔为1分钟
	err := a.SetInterval(1)
	if err != nil {
		return fmt.Sprintf("SetInterval(1) 失败: %v", err)
	}

	// 获取设置后的状态
	a.mu.Lock()
	remaining := a.remaining
	interval := a.settings.Interval
	isRunning := a.isRunning
	a.mu.Unlock()

	result := fmt.Sprintf("设置1分钟后:\n")
	result += fmt.Sprintf("  interval=%d分钟\n", interval)
	result += fmt.Sprintf("  remaining=%v (%d秒)\n", remaining, int(remaining.Seconds()))
	result += fmt.Sprintf("  isRunning=%v\n\n", isRunning)

	// 启动计时器
	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, "TestOneMinute: 准备启动计时器")
	}
	err = a.StartTimer()
	if err != nil {
		return result + fmt.Sprintf("StartTimer 失败: %v", err)
	}

	// 启动后状态
	a.mu.Lock()
	remaining = a.remaining
	isRunning = a.isRunning
	lastTick := a.lastTickTime
	a.mu.Unlock()

	result += fmt.Sprintf("StartTimer后:\n")
	result += fmt.Sprintf("  isRunning=%v\n", isRunning)
	result += fmt.Sprintf("  remaining=%v (%d秒)\n", remaining, int(remaining.Seconds()))
	result += fmt.Sprintf("  lastTickTime=%v\n", lastTick)
	result += fmt.Sprintf("  time.Now=%v\n\n", time.Now())

	// 等待500ms后检查
	time.Sleep(500 * time.Millisecond)

	remainingTime := a.GetRemainingTime()
	a.mu.Lock()
	isRunning = a.isRunning
	a.mu.Unlock()

	result += fmt.Sprintf("500ms后:\n")
	result += fmt.Sprintf("  isRunning=%v\n", isRunning)
	result += fmt.Sprintf("  GetRemainingTime=%d秒\n", remainingTime)

	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, fmt.Sprintf("=== TestOneMinute 完成: %s ===", result))
	}
	return result
}

func (a *App) TestTimer() string {
	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, "=== 开始测试 ===")
	}

	// 测试 1: 初始状态
	config := a.GetConfig()
	result := fmt.Sprintf("初始状态: interval=%d, isRunning=%v, remaining=%d\n",
		config.Interval, config.IsRunning, config.Remaining)

	// 测试 2: 设置间隔为1分钟
	err := a.SetInterval(1)
	if err != nil {
		result += fmt.Sprintf("SetInterval(1) 失败: %v\n", err)
		return result
	}

	config = a.GetConfig()
	result += fmt.Sprintf("设置间隔后: interval=%d, remaining=%d\n", config.Interval, config.Remaining)

	// 测试 3: 启动计时器
	err = a.StartTimer()
	if err != nil {
		result += fmt.Sprintf("StartTimer 失败: %v\n", err)
		return result
	}

	time.Sleep(100 * time.Millisecond) // 等待100ms

	remaining := a.GetRemainingTime()
	config = a.GetConfig()
	result += fmt.Sprintf("启动后: isRunning=%v, remaining=%d (GetRemainingTime返回=%d)\n",
		config.IsRunning, config.Remaining, remaining)

	// 测试 4: 重置计时器
	err = a.ResetTimer()
	if err != nil {
		result += fmt.Sprintf("ResetTimer 失败: %v\n", err)
		return result
	}

	time.Sleep(100 * time.Millisecond)

	remaining = a.GetRemainingTime()
	config = a.GetConfig()
	result += fmt.Sprintf("重置后: isRunning=%v, remaining=%d (GetRemainingTime返回=%d)\n",
		config.IsRunning, config.Remaining, remaining)

	// 测试 5: 暂停计时器
	err = a.PauseTimer()
	if err != nil {
		result += fmt.Sprintf("PauseTimer 失败: %v\n", err)
	}

	config = a.GetConfig()
	result += fmt.Sprintf("暂停后: isRunning=%v, remaining=%d\n", config.IsRunning, config.Remaining)

	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, "=== 测试完成 ===")
	}
	return result
}

// onTimerComplete 定时器完成回调
func (a *App) onTimerComplete() {
	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, "=== onTimerComplete 被调用 ===")
	}

	a.mu.Lock()
	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, fmt.Sprintf("完成前状态: isRunning=%v, remaining=%v", a.isRunning, a.remaining))
	}

	a.isRunning = false
	a.remaining = 0
	notificationDuration := a.settings.NotificationDuration
	activateOnTimer := a.settings.ActivateOnTimer
	message := a.settings.Message
	loopMode := a.settings.LoopMode
	interval := a.settings.Interval
	if message == "" {
		message = "您已经坐了很久了，起来活动一下吧！"
	}
	a.mu.Unlock()

	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, fmt.Sprintf("计时器完成: isRunning=false, remaining=0, loopMode=%v", loopMode))
	}

	// 发送系统通知
	iconPath := ""

	if notificationDuration == 0 && !loopMode {
		// 持久通知 - 使用Alert（非循环模式）
		err := beeep.Alert("久坐提醒", message, iconPath)
		if err != nil {
			if a.ctx != nil {
				wailsRuntime.LogError(a.ctx, fmt.Sprintf("通知发送失败: %v", err))
			}
		}
	} else {
		// 非持久通知（循环模式下强制使用）
		err := beeep.Notify("久坐提醒", message, iconPath)
		if err != nil {
			if a.ctx != nil {
				wailsRuntime.LogError(a.ctx, fmt.Sprintf("通知发送失败: %v", err))
			}
		}
	}

	// 激活窗口（如果开启了该设置）
	if activateOnTimer && a.ctx != nil {
		wailsRuntime.WindowSetAlwaysOnTop(a.ctx, true)
		time.Sleep(200 * time.Millisecond)
		wailsRuntime.WindowSetAlwaysOnTop(a.ctx, false)
		time.Sleep(100 * time.Millisecond)
		// 再次设置确保在前面
		wailsRuntime.WindowSetAlwaysOnTop(a.ctx, true)
		time.Sleep(200 * time.Millisecond)
		wailsRuntime.WindowSetAlwaysOnTop(a.ctx, false)
	}

	// 发送计时完成事件
	wailsRuntime.EventsEmit(a.ctx, "timer-completed", true)

	// 循环模式：10秒后自动开始下一轮
	if loopMode {
		if a.ctx != nil {
			wailsRuntime.LogInfo(a.ctx, "循环模式开启，10秒后将自动开始下一轮")
		}
		go func() {
			time.Sleep(10 * time.Second)
			a.mu.Lock()
			a.remaining = time.Duration(interval) * time.Minute
			a.isRunning = true
			a.lastTickTime = time.Now()
			a.timer = time.AfterFunc(a.remaining, a.onTimerComplete)
			a.mu.Unlock()
			
			if a.ctx != nil {
				wailsRuntime.LogInfo(a.ctx, fmt.Sprintf("循环模式：自动开始下一轮，间隔=%d分钟", interval))
			}
			wailsRuntime.EventsEmit(a.ctx, "timer-reset", float64(interval*60))
		}()
	}
}

// ForceReset 强制重置所有状态
func (a *App) ForceReset() string {
	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, "=== ForceReset 被调用 ===")
	}

	a.mu.Lock()

	// 停止现有计时器
	if a.timer != nil {
		a.timer.Stop()
		a.timer = nil
	}

	oldIsRunning := a.isRunning
	oldRemaining := a.remaining
	oldInterval := a.settings.Interval

	// 强制重置所有状态
	a.isRunning = false
	a.remaining = time.Duration(a.settings.Interval) * time.Minute

	if a.ctx != nil {
		wailsRuntime.LogInfo(a.ctx, fmt.Sprintf("ForceReset: isRunning %v->false, remaining %v->%v, interval=%d",
			oldIsRunning, oldRemaining, a.remaining, oldInterval))
	}

	a.mu.Unlock()

	// 发送事件通知前端
	wailsRuntime.EventsEmit(a.ctx, "timer-reset", a.remaining.Seconds())

	result := fmt.Sprintf("强制重置完成:\n")
	result += fmt.Sprintf("  旧状态: isRunning=%v, remaining=%v\n", oldIsRunning, oldRemaining)
	result += fmt.Sprintf("  新状态: isRunning=false, remaining=%v (%d秒)\n", a.remaining, int(a.remaining.Seconds()))
	result += fmt.Sprintf("  间隔设置: %d分钟\n", oldInterval)
	result += fmt.Sprintf("  请现在点击\"开始\"按钮测试")

	return result
}
