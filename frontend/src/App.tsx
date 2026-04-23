import {useState, useEffect, useCallback, useRef} from 'react';
import './App.css';
import {EventsOn} from "../wailsjs/runtime";
import {
    GetConfig,
    SetInterval,
    SetShortcut,
    SetNotificationDuration,
    SetActivateOnTimer,
    SetMessage,
    SetLoopMode,
    StartTimer,
    PauseTimer,
    ResetTimer,
    GetRemainingTime,
} from "../wailsjs/go/main/App";

interface ShortcutConfig {
    resetKey: string;
    startKey: string;
    closeNotifyKey: string;
    ctrl: boolean;
    shift: boolean;
    alt: boolean;
    global: boolean;
}

interface Config {
    interval: number;
    isRunning: boolean;
    remaining: number;
    shortcut: ShortcutConfig;
    notificationDuration: number;
    activateOnTimer: boolean;
    message: string;
    loopMode: boolean;
}

const defaultConfig: Config = {
    interval: 30,
    isRunning: false,
    remaining: 30 * 60,
    shortcut: {
        resetKey: 'r',
        startKey: 's',
        closeNotifyKey: 'Escape',
        ctrl: true,
        shift: true,
        alt: false,
        global: true
    },
    notificationDuration: 0,
    activateOnTimer: true,
    message: '您已经坐了很久了，起来活动一下吧！',
    loopMode: false,
};

// 预设文案
const MESSAGE_PRESETS = [
    { label: '默认', value: '您已经坐了很久了，起来活动一下吧！' },
    { label: '温馨', value: '该休息一下啦，喝杯水，走动走动~' },
    { label: '搞怪', value: '喂！屁股都要长蘑菇了！快起来！' },
];

function App() {
    const [config, setConfig] = useState<Config>(defaultConfig);
    const [inputInterval, setInputInterval] = useState(30);
    const [notification, setNotification] = useState(false);
    const [activeTab, setActiveTab] = useState<'timer' | 'message' | 'shortcut'>('timer');
    const [isProcessing, setIsProcessing] = useState(false);
    const [localMessage, setLocalMessage] = useState('');

    const operationLock = useRef(false);
    const [localShortcut, setLocalShortcut] = useState<ShortcutConfig>(defaultConfig.shortcut);

    // 计算快捷键显示文本
    const getShortcutDisplay = (sc: ShortcutConfig): string => {
        const parts: string[] = [];
        if (sc.ctrl) parts.push('⌃');
        if (sc.shift) parts.push('⇧');
        if (sc.alt) parts.push('⌥');
        return parts.join('') + sc.startKey.toUpperCase() + ' 开始 / ' + sc.resetKey.toUpperCase() + ' 重置';
    };

    const formatTime = (seconds: number): string => {
        const mins = Math.floor(Math.abs(seconds) / 60);
        const secs = Math.abs(seconds) % 60;
        return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
    };

    const loadConfig = useCallback(async () => {
        try {
            const cfg = await GetConfig();
            setConfig(cfg);
            setInputInterval(cfg.interval);
            setLocalShortcut(cfg.shortcut);
            setLocalMessage(cfg.message || '');
        } catch (e) {
            console.error('加载配置失败:', e);
        }
    }, []);

    const updateRemaining = useCallback(async () => {
        try {
            const remaining = await GetRemainingTime();
            setConfig(prev => ({...prev, remaining}));
        } catch (e) {
            console.error('获取剩余时间失败:', e);
        }
    }, []);

    useEffect(() => {
        loadConfig();
    }, [loadConfig]);

    useEffect(() => {
        if (!config.isRunning) return;
        const interval = setInterval(updateRemaining, 1000);
        return () => clearInterval(interval);
    }, [config.isRunning, updateRemaining]);

    useEffect(() => {
        EventsOn('timer-completed', () => {
            console.log('收到 timer-completed 事件');
            setNotification(true);
            setConfig(prev => ({...prev, isRunning: false}));
            setIsProcessing(false);
        });

        EventsOn('timer-reset', (data: number) => {
            console.log('收到 timer-reset 事件:', data);
            setConfig(prev => ({
                ...prev,
                remaining: data,
                isRunning: true
            }));
            setIsProcessing(false);
        });
    }, []);

    // 循环模式下，通知10秒后自动关闭
    useEffect(() => {
        if (notification && config.loopMode) {
            console.log('循环模式：10秒后自动关闭通知');
            const timer = setTimeout(() => {
                console.log('自动关闭通知');
                setNotification(false);
            }, 10000);
            return () => clearTimeout(timer);
        }
    }, [notification, config.loopMode]);

    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            const key = e.key;
            const shortcut = config.shortcut;

            // 修饰键匹配：配置为true时必须有该键按下，配置为false时必须没有按下
            const matchCtrl = (!shortcut.ctrl && !e.ctrlKey) || (shortcut.ctrl && e.ctrlKey);
            const matchShift = (!shortcut.shift && !e.shiftKey) || (shortcut.shift && e.shiftKey);
            const matchAlt = (!shortcut.alt && !e.altKey) || (shortcut.alt && e.altKey);

            const matchStart = (
                key.toLowerCase() === shortcut.startKey.toLowerCase() &&
                matchCtrl && matchShift && matchAlt
            );
            const matchReset = (
                key.toLowerCase() === shortcut.resetKey.toLowerCase() &&
                matchCtrl && matchShift && matchAlt
            );

            if (matchStart) {
                e.preventDefault();
                handleStart();
                return;
            }

            if (matchReset) {
                e.preventDefault();
                handleReset();
                return;
            }

            if (notification && key === shortcut.closeNotifyKey) {
                e.preventDefault();
                setNotification(false);
                return;
            }

            if (notification && key === 'Escape') {
                e.preventDefault();
                setNotification(false);
            }
        };

        window.addEventListener('keydown', handleKeyDown);
        return () => window.removeEventListener('keydown', handleKeyDown);
    }, [config.shortcut, notification]);

    const withLock = async (operation: () => Promise<void>, operationName: string) => {
        if (operationLock.current) {
            console.log(`[${operationName}] 操作被忽略: 正在处理中`);
            return;
        }

        operationLock.current = true;
        setIsProcessing(true);
        console.log(`[${operationName}] 开始执行`);

        try {
            await operation();
            console.log(`[${operationName}] 执行成功`);
        } catch (e: any) {
            console.error(`[${operationName}] 执行失败:`, e);
            alert(`${operationName} 失败: ${e?.message || String(e)}`);
        } finally {
            setIsProcessing(false);
            operationLock.current = false;
        }
    };

    const handleStart = async () => {
        await withLock(async () => {
            await StartTimer();
            await loadConfig();
        }, '开始计时');
    };

    const handlePause = async () => {
        await withLock(async () => {
            await PauseTimer();
            await loadConfig();
        }, '暂停计时');
    };

    const handleReset = async () => {
        await withLock(async () => {
            await ResetTimer();
        }, '重置计时');
    };

    const handleSaveInterval = async () => {
        await withLock(async () => {
            if (inputInterval < 1) {
                alert('间隔时间不能小于1分钟');
                return;
            }
            await SetInterval(inputInterval);
            await loadConfig();
        }, '保存设置');
    };

    const saveShortcut = async () => {
        await withLock(async () => {
            await SetShortcut(localShortcut);
            await loadConfig();
        }, '保存快捷键');
    };

    const handleActivateOnTimerChange = async (checked: boolean) => {
        await withLock(async () => {
            await SetActivateOnTimer(checked);
            setConfig(prev => ({...prev, activateOnTimer: checked}));
        }, '保存激活窗口设置');
    };

    const handleLoopModeChange = async (checked: boolean) => {
        await withLock(async () => {
            await SetLoopMode(checked);
            setConfig(prev => ({...prev, loopMode: checked}));
        }, '保存循环模式设置');
    };

    const handleSaveMessage = async () => {
        await withLock(async () => {
            if (!localMessage.trim()) {
                alert('提醒文案不能为空');
                return;
            }
            await SetMessage(localMessage.trim());
            await loadConfig();
        }, '保存提醒文案');
    };

    const handlePresetSelect = (preset: typeof MESSAGE_PRESETS[0]) => {
        setLocalMessage(preset.value);
    };

    return (
        <div id="App">
            {isProcessing && (
                <div className="processing-overlay">
                    <div className="processing-spinner"/>
                </div>
            )}

            {notification && (
                <div className="notification-overlay">
                    <div className="notification-box">
                        <div className="notification-close-hint">
                            按 {config.shortcut.closeNotifyKey} 关闭
                        </div>
                        <div className="notification-icon">&#9200;</div>
                        <h2>时间到！</h2>
                        <p>{config.message || '您已经坐了很久了，起来活动一下吧！'}</p>
                        <button
                            className="notification-btn"
                            onClick={() => setNotification(false)}
                        >
                            我知道了
                        </button>
                    </div>
                </div>
            )}

            <div className="header">
                <h1>久坐提醒</h1>
                <p>定时提醒，保持健康</p>
            </div>

            <div className="timer-container">
                <div className={`timer-display ${config.isRunning ? 'running' : ''}`}>
                    {formatTime(config.remaining)}
                </div>
                <div className="timer-status">
                    {config.isRunning ? '计时中' : '已暂停'}
                </div>
            </div>

            <div className="controls">
                {config.isRunning ? (
                    <button className="btn btn-pause" onClick={handlePause} disabled={isProcessing}>
                        {isProcessing ? '...' : '暂停'}
                    </button>
                ) : (
                    <button className="btn btn-start" onClick={handleStart} disabled={isProcessing}>
                        {isProcessing ? '...' : '开始'}
                    </button>
                )}
                <button className="btn btn-reset" onClick={handleReset} disabled={isProcessing}>
                    {isProcessing ? '...' : '重置'}
                </button>
            </div>

            <div className="settings">
                <div className="settings-tabs">
                    <button
                        className={`tab-btn ${activeTab === 'timer' ? 'active' : ''}`}
                        onClick={() => setActiveTab('timer')}
                    >
                        时间设置
                    </button>
                    <button
                        className={`tab-btn ${activeTab === 'message' ? 'active' : ''}`}
                        onClick={() => setActiveTab('message')}
                    >
                        提醒文案
                    </button>
                    <button
                        className={`tab-btn ${activeTab === 'shortcut' ? 'active' : ''}`}
                        onClick={() => setActiveTab('shortcut')}
                    >
                        快捷键
                    </button>
                </div>

                {activeTab === 'timer' ? (
                    <div className="settings-content">
                    <>
                        <div className="setting-row">
                            <label className="setting-label">提醒间隔</label>
                            <input
                                type="number"
                                min="1"
                                max="180"
                                value={inputInterval}
                                onChange={(e) => setInputInterval(parseInt(e.target.value) || 1)}
                                className="input-interval"
                                disabled={isProcessing}
                            />
                            <span className="unit">分钟</span>
                        </div>

                        <div className="setting-row">
                            <label className="setting-label">弹窗时顺便弹出界面</label>
                            <label className="switch">
                                <input
                                    type="checkbox"
                                    checked={config.activateOnTimer}
                                    onChange={(e) => handleActivateOnTimerChange(e.target.checked)}
                                    disabled={isProcessing}
                                />
                                <span className="slider"/>
                            </label>
                        </div>

                        <div className="setting-row">
                            <label className="setting-label">循环模式</label>
                            <label className="switch">
                                <input
                                    type="checkbox"
                                    checked={config.loopMode}
                                    onChange={(e) => handleLoopModeChange(e.target.checked)}
                                    disabled={isProcessing}
                                />
                                <span className="slider"/>
                            </label>
                        </div>

                        <div className="setting-row save-row">
                            <button className="btn btn-save" onClick={handleSaveInterval} disabled={isProcessing}>
                                {isProcessing ? '...' : '保存'}
                            </button>
                        </div>
                    </>
                    </div>
                ) : (
                    <div className="settings-content" style={{display:"none"}} />
                )}

                {activeTab === 'message' ? (
                    <div className="settings-content">
                    <>
                        <div className="setting-row message-row">
                            <label className="setting-label">提醒文案</label>
                            <textarea
                                value={localMessage}
                                onChange={(e) => setLocalMessage(e.target.value)}
                                className="input-message"
                                placeholder="输入自定义提醒文案..."
                                disabled={isProcessing}
                                rows={3}
                            />
                        </div>

                        <div className="setting-row">
                            <label className="setting-label">预设文案</label>
                            <div className="preset-buttons">
                                {MESSAGE_PRESETS.map((preset, idx) => (
                                    <button
                                        key={idx}
                                     
                                        className={`btn  btn-preset ${localMessage === preset.value ? 'active' : ''}`}
                                        onClick={() => handlePresetSelect(preset)}
                                        disabled={isProcessing}
                                    >
                                        {preset.label}
                                    </button>
                                ))}
                            </div>
                        </div>

                        <div className="setting-row save-row">
                            <button className="btn btn-save" onClick={handleSaveMessage} disabled={isProcessing}>
                                {isProcessing ? '...' : '保存文案'}
                            </button>
                        </div>
                    </>
                    </div>
                ) : (
                    <div className="settings-content" style={{display:"none"}} />
                )}

                {activeTab === 'shortcut' ? (
                    <div className="settings-content">
                    <>
                        <div className="setting-row">
                            <label className="setting-label">开始键</label>
                            <input
                                type="text"
                                value={localShortcut.startKey}
                                onChange={(e) => setLocalShortcut(prev => ({
                                    ...prev, startKey: e.target.value
                                }))}
                                className="input-shortcut"
                                maxLength={1}
                                placeholder="如: s"
                                disabled={isProcessing}
                            />
                            <label className="setting-label" style={{marginLeft: '20px'}}>重置键</label>
                            <input
                                type="text"
                                value={localShortcut.resetKey}
                                onChange={(e) => setLocalShortcut(prev => ({
                                    ...prev, resetKey: e.target.value
                                }))}
                                className="input-shortcut"
                                maxLength={1}
                                placeholder="如: r"
                                disabled={isProcessing}
                            />
                        </div>

                        <div className="setting-row">
                            <label className="setting-label">修饰键</label>
                            <div className="checkbox-group">
                                <label className="checkbox-label">
                                    <input
                                        type="checkbox"
                                        checked={localShortcut.ctrl}
                                        onChange={(e) => setLocalShortcut(prev => ({
                                            ...prev, ctrl: e.target.checked
                                        }))}
                                        disabled={isProcessing}
                                    />
                                    Ctrl
                                </label>
                                <label className="checkbox-label">
                                    <input
                                        type="checkbox"
                                        checked={localShortcut.shift}
                                        onChange={(e) => setLocalShortcut(prev => ({
                                            ...prev, shift: e.target.checked
                                        }))}
                                        disabled={isProcessing}
                                    />
                                    Shift
                                </label>
                                <label className="checkbox-label">
                                    <input
                                        type="checkbox"
                                        checked={localShortcut.alt}
                                        onChange={(e) => setLocalShortcut(prev => ({
                                            ...prev, alt: e.target.checked
                                        }))}
                                        disabled={isProcessing}
                                    />
                                    Alt
                                </label>
                            </div>
                        </div>

                        <div className="setting-row">
                            <label className="setting-label">全局生效</label>
                            <label className="switch">
                                <input
                                    type="checkbox"
                                    checked={localShortcut.global}
                                    onChange={(e) => setLocalShortcut(prev => ({
                                        ...prev, global: e.target.checked
                                    }))}
                                    disabled={isProcessing}
                                />
                                <span className="slider"/>
                            </label>
                            <span className="hint-text"><span className="shortcut-preview">
                                当前: <span className="key">{getShortcutDisplay(localShortcut)}</span>
                            </span></span>
                        </div>

                        <div className="setting-row save-row">
                            
                            <button className="btn btn-save" onClick={saveShortcut} disabled={isProcessing}>
                                {isProcessing ? '...' : '保存快捷键'}
                            </button>
                        </div>
                    </>
                    </div>
                ) : (
                    <div className="settings-content" style={{display:"none"}} />
                )}
            </div>

            <div className="help">
                <ul>
                    <li>设置提醒间隔后点击「开始」</li>
                    <li>
                        <span className="key">{getShortcutDisplay(config.shortcut)}</span>
                        {config.shortcut.global && <span className="hint-text">（全局）</span>}
                    </li>
                    <li>点击「快捷键」标签自定义热键</li>
                </ul>
            </div>
        </div>
    );
}

export default App
