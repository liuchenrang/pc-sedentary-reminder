# 久坐提醒 (SitLong)

使用 Wails2 + React + TypeScript + Go 开发的跨平台久坐提醒桌面应用。

## 功能特性

- ⏱️ **定时提醒**: 支持自定义提醒时间间隔 (1-180分钟)
- 🔔 **系统通知**: 使用操作系统的原生通知方式，**持久显示直到手动关闭**
- ⌨️ **自定义快捷键**: 支持自定义重置和关闭通知的快捷键
- 🎨 **美观界面**: 使用 React + CSS 现代的界面设计，**自适应窗口无滚动条**
- 🔄 **实时倒计时**: 动态显示剩余时间
- ▶️/⏸️ **控制功能**: 支持开始、暂停、重置计时器
- 💾 **配置持久化**: 自动保存设置到配置文件

## 技术栈

| 技术 | 用途 |
|------|------|
| Wails2 | 跨平台桌面应用框架 |
| Go | 后端业务逻辑、系统通知 |
| React 18 | 前端 UI 框架 |
| TypeScript | 类型安全 |
| Vite | 前端构建工具 |
| beeep | 跨平台系统通知 |

## 快速开始

### 运行开发模式

```bash
wails dev
```

### 构建生产版本

```bash
wails build
```

构建完成后，应用位于 `build/bin/sitlong.app` (macOS) 或 `build/bin/sitlong` (Windows/Linux)。

## 使用说明

### 基本操作

1. **设置提醒间隔**: 在"时间设置"标签页输入时间（分钟），点击"保存"
2. **开始计时**: 点击"开始"按钮
3. **暂停计时**: 点击"暂停"按钮
4. **重置计时**: 点击"重置"按钮或使用快捷键
5. **到达时间**: 系统会弹出通知提醒，**不会自动消失**，需要手动关闭

### 自定义快捷键

点击"快捷键"标签页，可以设置：

| 功能 | 说明 |
|------|------|
| 重置键 | 触发重置计时的按键 |
| 关闭通知键 | 关闭提醒弹窗的按键 |
| 修饰键 | Ctrl / Shift / Alt 组合键 |

默认快捷键：**Ctrl+Shift+R** 重置计时

### 关闭提醒通知

提醒弹出后 **不会自动消失**，关闭方式：
- 点击"我知道了"按钮
- 按下设置的"关闭通知键"（默认 Escape）

## 界面特点

### 自适应布局

- ✨ 自动适配窗口大小，**无滚动条**
- ✨ 元素使用 `clamp()` 响应式缩放
- ✨ 小屏幕自动隐藏帮助区域，确保核心功能可见
- ✨ 标签页切换管理设置，节省空间

### 通知弹窗

- 🔔 持久化显示（不会自动消失）
- 🎉 动画效果和图标
- ⌨️ 支持快捷键快速关闭
- 👻 半透明背景遮罩

## 配置文件

配置自动保存到系统配置目录：

- **macOS**: `~/Library/Application Support/SitLong/sitlong-config.json`
- **Windows**: `%APPDATA%\SitLong\sitlong-config.json`
- **Linux**: `~/.config/sitlong/sitlong-config.json`

配置示例：
```json
{
  "interval": 30,
  "notificationDuration": 0,
  "shortcut": {
    "resetKey": "r",
    "closeNotifyKey": "Escape",
    "ctrl": true,
    "shift": true,
    "alt": false
  }
}
```

## 文件结构

```
sitlong/
├── app.go                 # Go 后端逻辑（定时器、通知、配置存储）
├── main.go                # 应用入口
├── wails.json             # Wails 配置文件
├── go.mod / go.sum        # Go 依赖
└── frontend/
    ├── src/
    │   ├── App.tsx        # React 主组件（标签页、快捷键设置）
    │   ├── App.css        # 自适应样式（flex布局、clamp）
    │   └── main.tsx       # 前端入口
    ├── package.json       # 前端依赖
    └── wailsjs/           # 自动生成的 Wails 绑定
```

## 开发说明

### 添加新功能

1. 在 `app.go` 中添加新的导出方法
2. 运行 `wails generate module` 生成 JS 绑定
3. 在 `frontend/src/App.tsx` 中调用新方法
4. 使用 `frontend` 目录下的 npm 命令开发前端

### 系统通知

应用使用 `github.com/gen2brain/beeep` 库实现跨平台系统通知：
- `beeep.Alert()`: 持久通知（不会自动消失）
- `beeep.Notify()`: 临时通知

### 配置持久化

配置保存在系统标准配置目录，确保：
- 应用重启后设置保留
- 不同系统遵循各自规范
- 独立配置文件，不污染代码

### 布局适配技术

- **Flex布局**: 主容器使用 `display: flex`
- **clamp() 函数**: 元素大小随视口变化
- **overflow: hidden**: 禁止整体滚动
- **媒体查询**: 针对小屏幕特殊处理

## 平台说明

### macOS
- ✅ 需要授权通知权限
- ✅ 配置保存在 `~/Library/Application Support/SitLong/`

### Windows
- ✅ 支持 Windows 10/11
- ✅ 配置保存在 `%APPDATA%\SitLong\`

### Linux
- ✅ 需要安装通知服务 (如 notify-osd)
- ✅ 配置保存在 `~/.config/sitlong/`

## License

MIT License
