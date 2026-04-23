# SitLong 图标设计说明

## 图标预览

SVG 源文件：`build/icon.svg`

## 设计理念

### 视觉元素
- **圆形徽章**：符合 macOS/Windows 桌面图标设计规范
- **时钟**：体现定时提醒的核心功能，指针带有微动画
- **起立伸展的人物剪影**：直观表达"久坐提醒，起身活动"的主题
- **动态弧线**：增强视觉活力和现代感

### 配色方案
- **主色渐变**：科技蓝 (#3B82F6) → 健康绿 (#10B981)
- **象征意义**：
  - 蓝色：科技、专业、信任
  - 绿色：健康、活力、自然

### 设计特点
- ✅ 简洁现代，符合当前桌面应用设计趋势
- ✅ 圆形背景适配各平台图标遮罩
- ✅ 清晰可辨识，小尺寸下依然醒目
- ✅ 语义明确，用户一眼就能理解应用功能

## 生成的文件

| 文件 | 用途 | 尺寸 |
|------|------|------|
| `build/icon.svg` | 矢量源文件 | 可缩放 |
| `build/appicon.png` | 主图标（Wails 源） | 1024x1024 |
| `build/darwin/icon.icns` | macOS 应用图标 | 多尺寸集合 |
| `build/windows/icon.ico` | Windows 应用图标 | 256x256 |

## 构建说明

Wails 会自动使用以下标准路径的图标：

```
build/
├── appicon.png          ← 主图标源文件
├── darwin/
│   └── icon.icns        ← macOS 图标（自动使用）
└── windows/
    └── icon.ico         ← Windows 图标（自动使用）
```

### 构建命令

```bash
# 开发模式
wails dev

# 构建当前平台
wails build

# 构建 macOS
wails build -platform darwin/amd64
wails build -platform darwin/arm64
wails build -platform darwin/universal

# 构建 Windows（需要在 Windows 或交叉编译环境）
wails build -platform windows/amd64
wails build -platform windows/arm64
```

构建完成后：
- macOS：`build/bin/sitlong.app`（图标已嵌入）
- Windows：`build/bin/sitlong.exe`（图标已嵌入）

## 自定义修改

如需修改图标：

1. 编辑 `build/icon.svg` 源文件
2. 重新生成 PNG：`rsvg-convert -w 1024 -h 1024 icon.svg -o appicon.png`
3. 重新生成 macOS icns：
   ```bash
   mkdir -p appicon.iconset
   for size in 16 32 64 128 256 512; do
     sips -z $size $size appicon.png --out appicon.iconset/icon_${size}x${size}.png
     sips -z $((size*2)) $((size*2)) appicon.png --out appicon.iconset/icon_${size}x${size}@2x.png
   done
   iconutil -c icns appicon.iconset -o darwin/icon.icns
   ```
4. 重新生成 Windows ico：
   ```bash
   magick appicon.png -resize 256x256 windows/icon.ico
   ```

## 配色代码

主渐变：
- 起点：`#3B82F6`（蓝色）
- 终点：`#10B981`（绿色）

时钟渐变：
- 起点：`#60A5FA`（浅蓝）
- 终点：`#34D399`（浅绿）

---

设计时间：2026-04-22
设计工具：SVG 矢量图形
兼容平台：macOS、Windows、Linux
