# SitLong - 久坐提醒桌面应用

> 每隔一段时间提醒你站起来活动，告别久坐危害。

## 🎯 产品介绍

SitLong 是一款简洁高效的久坐提醒桌面应用，专为长时间办公人群设计。它能在你连续久坐一段时间后自动弹出提醒，催促你起身活动、放松身体。

### 核心功能

- ⏱ **自定义计时**：设置提醒间隔（分钟）
- 🔁 **循环模式**：提醒后自动开始下一轮，无需手动操作
- 📢 **自定义文案**：内置多套提醒文案，也可自定义
- ⌨ **全局快捷键**：支持自定义开始/重置热键，随时随地控制
- 🖥 **窗口激活**：计时结束时自动激活应用窗口，确保你看到提醒

## 🖥 技术架构

### 技术栈

| 层级 | 技术 |
|------|------|
| 桌面框架 | [Wails](https://wails.io/) |
| 后端 | Go |
| 前端 | React + TypeScript |
| 构建工具 | Vite |
| 本地存储 | JSON 配置文件 |

### 核心设计

#### 1. Wails 前后端分离架构

```
┌─────────────────────────────────────┐
│           Frontend (React)          │
│  ┌───────────┐  ┌────────────────┐  │
│  │   UI层    │  │  状态管理      │  │
│  └─────┬─────┘  └───────┬────────┘  │
│        └────────┬───────┘           │
│                 │ Wails Bindings   │
├─────────────────┼───────────────────┤
│                 │                   │
│  ┌──────────────┴────────────────┐│
│  │        Backend (Go)             ││
│  │  ┌─────────┐  ┌─────────────┐  ││
│  │  │ Timer   │  │  Settings    │  ││
│  │  │ Engine  │  │  Manager     │  ││
│  │  └────┬────┘  └──────┬──────┘  ││
│  │       └───────┬──────┘          ││
│  │       ┌───────┴───────┐          ││
│  │  ┌────┴────┐  ┌───────┴────┐   ││
│  │  │ Window  │  │  Menu/     │   ││
│  │  │ Manager │  │  Shortcuts │   ││
│  └───────────┴───────────────────┘ │
└─────────────────────────────────────┘
```

**关键特性：**
- 前端通过 `wailsjs` 包直接调用 Go 方法，无需 REST API
- 后端事件（timer-completed）实时推送到前端
- 单二进制程序，分发简单

#### 2. 配置管理

```go
type SitLongConfig struct {
    Interval             int            // 间隔（分钟）
    IsRunning            bool           // 计时器状态
    Remaining           int            // 剩余秒数
    Shortcut            ShortcutConfig // 快捷键配置
    NotificationDuration int            // 通知持续时间
    ActivateOnTimer     bool           // 计时结束激活窗口
    Message             string         // 提醒文案
    LoopMode            bool           // 循环模式
}

type ShortcutConfig struct {
    ResetKey       string // 重置键字母
    StartKey       string // 开始键字母
    CloseNotifyKey string // 关闭通知键
    Ctrl           bool   // 是否使用 Ctrl
    Shift          bool   // 是否使用 Shift
    Alt            bool   // 是否使用 Alt
    Global         bool   // 是否全局热键
}
```

配置文件存储在系统配置目录（跨平台兼容）:
- **Windows**: `%APPDATA%/sitlong/`
- **macOS**: `~/Library/Application Support/sitlong/`
- **Linux**: `~/.config/sitlong/`

#### 3. 计时器引擎

后端 Go 协程驱动的高精度计时器：

```go
func (a *App) startTimer() {
    a.mu.Lock()
    if a.isRunning {
        a.mu.Unlock()
        return
    }
    a.isRunning = true
    remaining := a.settings.Interval * 60
    a.mu.Unlock()

    go func() {
        ticker := time.NewTicker(1 * time.Second)
        defer ticker.Stop()

        for {
            select {
            case <-ticker.C:
                a.mu.Lock()
                remaining--
                a.remaining = remaining
                isRunning := a.isRunning
                a.mu.Unlock()

                if !isRunning || remaining <= 0 {
                    if remaining <= 0 {
                        a.onTimerComplete()
                    }
                    return
                }
            }
        }
    }()
}
```

#### 4. 全局快捷键

利用 Wails 菜单系统的加速器功能：

```go
// 构建加速器字符串
func (a *App) buildStartAccelerator() string {
    var mods []string
    if a.settings.Shortcut.Ctrl {
        mods = append(mods, "ctrl")
    }
    if a.settings.Shortcut.Shift {
        mods = append(mods, "shift")
    }
    if a.settings.Shortcut.Alt {
        mods = append(mods, "alt")
    }
    key := strings.ToLower(a.settings.Shortcut.StartKey)
    return strings.Join(mods, "+") + "+" + key
}
```

#### 5. 前后端事件通信

```typescript
// 前端监听后端事件
EventsOn('timer-completed', () => {
    setNotification(true);
});

EventsOn('timer-reset', (data: number) => {
    setConfig(prev => ({...prev, remaining: data, isRunning: true}));
});
```

```go
// 后端发送事件
wailsRuntime.EventsEmit(a.ctx, "timer-completed")
wailsRuntime.EventsEmit(a.ctx, "timer-reset", remaining)
```

#### 6. 循环模式

循环模式下，计时结束自动开始下一轮：

```go
func (a *App) onTimerComplete() {
    a.mu.Lock()
    isLoopMode := a.settings.LoopMode
    a.mu.Unlock()

    // 显示通知...

    if isLoopMode {
        // 延迟1秒后自动开始下一轮
        go func() {
            time.Sleep(1 * time.Second)
            a.mu.Lock()
            a.remaining = a.settings.Interval * 60
            a.isRunning = true
            a.mu.Unlock()
            a.startTimerGoroutine()
            wailsRuntime.EventsEmit(a.ctx, "timer-reset", a.remaining)
        }()
    }
}
```

### 项目结构

```
sitlong/
├── app.go              # Go 后端主逻辑
├── main.go             # 程序入口
├── go.mod
├── wails.json          # Wails 配置
├── frontend/
│   ├── src/
│   │   ├── App.tsx     # React 主组件
│   │   ├── App.css     # 样式
│   │   ├── main.tsx
│   │   └── vite-env.d.ts
│   ├── index.html
│   ├── package.json
│   ├── vite.config.ts
│   └── wailsjs/
│       └── go/
│           ├── main/
│           │   ├── App.js      # Wails 生成的 JS 绑定
│           │   └── App.d.ts    # TypeScript 类型声明
│           └── models.ts       # 数据模型
└── README.md
```

## 🔧 构建与运行

```bash
# 开发模式（热重载）
wails dev

# 生产构建
wails build

# macOS 构建
wails build -platform darwin/universal

# Windows 构建
wails build -platform windows/amd64
```

## ✨ 特色亮点

1. **轻量高效**：单二进制文件，无需 Electron 的 Node.js 运行时
2. **跨平台**：同一代码，支持 Windows、macOS、Linux
3. **响应迅速**：Go 后端处理计时，前端仅负责 UI
4. **原生体验**：使用系统原生菜单和窗口
5. **低资源占用**：内存占用 < 50MB，CPU 几乎为零

## 📝 总结

SitLong 展示了如何使用 Wails 构建一个轻量、跨平台、功能完整的桌面应用。通过 Go 实现核心逻辑保证了高性能和低资源占用，而 React/TypeScript 前端则提供了现代化的开发体验和良好的可维护性。

对于需要构建类似工具类应用的开发者，Wails 是一个值得考虑的选择。