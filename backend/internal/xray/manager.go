package xray

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"xpanel/internal/config"
	"xpanel/internal/database"
	"xpanel/internal/models"
)

type Manager struct {
	binaryPath string
	configPath string
	assetsPath string
	process    *exec.Cmd
	running    bool
	version    string
	mu         sync.RWMutex
}

// XrayConfig Xray 配置结构
type XrayConfig struct {
	Log       *LogConfig      `json:"log,omitempty"`
	API       *APIConfig      `json:"api,omitempty"`
	Stats     *StatsConfig    `json:"stats,omitempty"`
	Policy    *PolicyConfig   `json:"policy,omitempty"`
	Inbounds  []InboundConfig `json:"inbounds"`
	Outbounds []OutboundConfig `json:"outbounds"`
	Routing   *RoutingConfig  `json:"routing,omitempty"`
}

type LogConfig struct {
	Access   string `json:"access,omitempty"`
	Error    string `json:"error,omitempty"`
	Loglevel string `json:"loglevel"`
}

type APIConfig struct {
	Tag      string   `json:"tag"`
	Services []string `json:"services"`
}

type StatsConfig struct{}

type PolicyConfig struct {
	System *SystemPolicy `json:"system,omitempty"`
}

type SystemPolicy struct {
	StatsInboundUplink   bool `json:"statsInboundUplink"`
	StatsInboundDownlink bool `json:"statsInboundDownlink"`
}

type InboundConfig struct {
	Tag            string          `json:"tag"`
	Port           int             `json:"port"`
	Listen         string          `json:"listen"`
	Protocol       string          `json:"protocol"`
	Settings       json.RawMessage `json:"settings"`
	StreamSettings json.RawMessage `json:"streamSettings,omitempty"`
	Sniffing       json.RawMessage `json:"sniffing,omitempty"`
}

type OutboundConfig struct {
	Tag            string          `json:"tag"`
	Protocol       string          `json:"protocol"`
	Settings       json.RawMessage `json:"settings,omitempty"`
	StreamSettings json.RawMessage `json:"streamSettings,omitempty"`
	ProxySettings  json.RawMessage `json:"proxySettings,omitempty"`
	Mux            json.RawMessage `json:"mux,omitempty"`
}

type RoutingConfig struct {
	DomainStrategy string        `json:"domainStrategy"`
	Rules          []RoutingRule `json:"rules"`
}

type RoutingRule struct {
	Type        string   `json:"type"`
	InboundTag  []string `json:"inboundTag,omitempty"`
	OutboundTag string   `json:"outboundTag"`
	IP          []string `json:"ip,omitempty"`
	Domain      []string `json:"domain,omitempty"`
}

func NewManager(cfg *config.XrayConfig) *Manager {
	m := &Manager{
		binaryPath: cfg.BinaryPath,
		configPath: cfg.ConfigPath,
		assetsPath: cfg.AssetsPath,
	}

	// 获取版本
	m.version = m.fetchVersion()

	return m
}

// fetchVersion 获取 Xray 版本
func (m *Manager) fetchVersion() string {
	cmd := exec.Command(m.binaryPath, "version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		parts := strings.Fields(lines[0])
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return "unknown"
}

// IsRunning 检查 Xray 是否运行
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// GetVersion 获取版本
func (m *Manager) GetVersion() string {
	return m.version
}

// Start 启动 Xray
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return errors.New("Xray 已在运行")
	}

	// 生成配置
	if err := m.generateConfig(); err != nil {
		return fmt.Errorf("生成配置失败: %v", err)
	}

	// 验证配置
	if err := m.testConfig(); err != nil {
		return fmt.Errorf("配置验证失败: %v", err)
	}

	// 启动进程
	m.process = exec.Command(m.binaryPath, "run", "-c", m.configPath)
	if m.assetsPath != "" {
		m.process.Env = append(os.Environ(), "XRAY_LOCATION_ASSET="+m.assetsPath)
	}

	if err := m.process.Start(); err != nil {
		return fmt.Errorf("启动失败: %v", err)
	}

	m.running = true

	// 监控进程
	go m.monitor()

	return nil
}

// Stop 停止 Xray
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running || m.process == nil {
		return nil
	}

	if err := m.process.Process.Signal(syscall.SIGTERM); err != nil {
		// 强制杀死
		m.process.Process.Kill()
	}

	m.running = false
	return nil
}

// Restart 重启 Xray
func (m *Manager) Restart() error {
	if err := m.Stop(); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)
	return m.Start()
}

// Reload 热重载配置
func (m *Manager) Reload() error {
	m.mu.Lock()

	// 生成新配置
	if err := m.generateConfig(); err != nil {
		m.mu.Unlock()
		return fmt.Errorf("生成配置失败: %v", err)
	}

	// 验证配置
	if err := m.testConfig(); err != nil {
		m.mu.Unlock()
		return fmt.Errorf("配置验证失败: %v", err)
	}

	if !m.running || m.process == nil {
		m.mu.Unlock()
		return nil
	}

	// 发送 SIGHUP 信号热重载
	if err := m.process.Process.Signal(syscall.SIGHUP); err != nil {
		// 如果热重载失败，尝试重启（先释放锁避免死锁）
		m.mu.Unlock()
		return m.Restart()
	}

	m.mu.Unlock()
	return nil
}

// testConfig 验证配置
func (m *Manager) testConfig() error {
	cmd := exec.Command(m.binaryPath, "-test", "-c", m.configPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(output))
	}
	return nil
}

// generateConfig 生成 Xray 配置
func (m *Manager) generateConfig() error {
	xrayConfig := &XrayConfig{
		Log: &LogConfig{
			Loglevel: "warning",
		},
		API: &APIConfig{
			Tag:      "api",
			Services: []string{"HandlerService", "StatsService"},
		},
		Stats: &StatsConfig{},
		Policy: &PolicyConfig{
			System: &SystemPolicy{
				StatsInboundUplink:   true,
				StatsInboundDownlink: true,
			},
		},
		Routing: &RoutingConfig{
			DomainStrategy: "AsIs",
			Rules: []RoutingRule{
				{
					Type:        "field",
					InboundTag:  []string{"api"},
					OutboundTag: "api",
				},
			},
		},
	}

	// 添加 API 入站
	apiInbound := InboundConfig{
		Tag:      "api",
		Port:     10085,
		Listen:   "127.0.0.1",
		Protocol: "dokodemo-door",
		Settings: json.RawMessage(`{"address": "127.0.0.1"}`),
	}
	xrayConfig.Inbounds = append(xrayConfig.Inbounds, apiInbound)

	// 从数据库加载入站配置
	var inbounds []models.Inbound
	database.DB.Where("enable = ?", true).Find(&inbounds)

	for _, ib := range inbounds {
		inboundConfig := InboundConfig{
			Tag:      ib.Tag,
			Port:     ib.Port,
			Listen:   ib.Listen,
			Protocol: ib.Protocol,
		}

		// 获取此入站的所有客户端
		clients, _ := m.getInboundClients(ib.ID)
		settings := m.buildInboundSettings(ib.Protocol, clients)
		inboundConfig.Settings = settings

		if ib.StreamSettings != "" && ib.StreamSettings != "null" {
			inboundConfig.StreamSettings = json.RawMessage(ib.StreamSettings)
		}
		if ib.Sniffing != "" && ib.Sniffing != "null" {
			inboundConfig.Sniffing = json.RawMessage(ib.Sniffing)
		}

		xrayConfig.Inbounds = append(xrayConfig.Inbounds, inboundConfig)
	}

	// 从数据库加载出站配置
	var outbounds []models.Outbound
	database.DB.Where("enable = ?", true).Find(&outbounds)

	for _, ob := range outbounds {
		outboundConfig := OutboundConfig{
			Tag:      ob.Tag,
			Protocol: ob.Protocol,
		}

		if ob.Settings != "" && ob.Settings != "null" {
			outboundConfig.Settings = json.RawMessage(ob.Settings)
		}
		if ob.StreamSettings != "" && ob.StreamSettings != "null" {
			outboundConfig.StreamSettings = json.RawMessage(ob.StreamSettings)
		}

		xrayConfig.Outbounds = append(xrayConfig.Outbounds, outboundConfig)
	}

	// 如果没有出站，添加默认的
	if len(xrayConfig.Outbounds) == 0 {
		xrayConfig.Outbounds = []OutboundConfig{
			{Tag: "direct", Protocol: "freedom", Settings: json.RawMessage(`{}`)},
			{Tag: "blocked", Protocol: "blackhole", Settings: json.RawMessage(`{}`)},
		}
	}

	// 写入文件
	configDir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(xrayConfig, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.configPath, data, 0600)
}

// getInboundClients 获取入站的客户端列表
func (m *Manager) getInboundClients(inboundID uint) ([]models.Client, error) {
	var clients []models.Client
	err := database.DB.
		Joins("JOIN inbound_clients ON inbound_clients.client_id = clients.id").
		Where("inbound_clients.inbound_id = ? AND clients.enable = ?", inboundID, true).
		Find(&clients).Error
	return clients, err
}

// buildInboundSettings 构建入站设置
func (m *Manager) buildInboundSettings(protocol string, clients []models.Client) json.RawMessage {
	switch protocol {
	case "vmess":
		return m.buildVMessSettings(clients)
	case "vless":
		return m.buildVLESSSettings(clients)
	case "trojan":
		return m.buildTrojanSettings(clients)
	case "shadowsocks":
		return m.buildShadowsocksSettings(clients)
	default:
		return json.RawMessage(`{}`)
	}
}

func (m *Manager) buildVMessSettings(clients []models.Client) json.RawMessage {
	type vmessClient struct {
		ID      string `json:"id"`
		Email   string `json:"email,omitempty"`
		AlterId int    `json:"alterId"`
	}

	vmessClients := make([]vmessClient, 0, len(clients))
	for _, c := range clients {
		vmessClients = append(vmessClients, vmessClient{
			ID:      c.UUID,
			Email:   c.Email,
			AlterId: 0,
		})
	}

	settings := map[string]interface{}{
		"clients": vmessClients,
	}

	data, _ := json.Marshal(settings)
	return data
}

func (m *Manager) buildVLESSSettings(clients []models.Client) json.RawMessage {
	type vlessClient struct {
		ID    string `json:"id"`
		Email string `json:"email,omitempty"`
		Flow  string `json:"flow,omitempty"`
	}

	vlessClients := make([]vlessClient, 0, len(clients))
	for _, c := range clients {
		vlessClients = append(vlessClients, vlessClient{
			ID:    c.UUID,
			Email: c.Email,
		})
	}

	settings := map[string]interface{}{
		"clients":    vlessClients,
		"decryption": "none",
	}

	data, _ := json.Marshal(settings)
	return data
}

func (m *Manager) buildTrojanSettings(clients []models.Client) json.RawMessage {
	type trojanClient struct {
		Password string `json:"password"`
		Email    string `json:"email,omitempty"`
	}

	trojanClients := make([]trojanClient, 0, len(clients))
	for _, c := range clients {
		trojanClients = append(trojanClients, trojanClient{
			Password: c.UUID,
			Email:    c.Email,
		})
	}

	settings := map[string]interface{}{
		"clients": trojanClients,
	}

	data, _ := json.Marshal(settings)
	return data
}

func (m *Manager) buildShadowsocksSettings(clients []models.Client) json.RawMessage {
	// Shadowsocks 多用户模式 - 必须有客户端
	if len(clients) == 0 {
		// 无客户端时使用占位配置，但实际上这个入站不会有可用用户
		return json.RawMessage(`{"method": "aes-256-gcm", "password": "no-client-configured", "network": "tcp,udp"}`)
	}

	settings := map[string]interface{}{
		"method":   "aes-256-gcm",
		"password": clients[0].UUID,
		"network":  "tcp,udp",
	}

	data, _ := json.Marshal(settings)
	return data
}

// GetCurrentConfig 获取当前配置
func (m *Manager) GetCurrentConfig() (interface{}, error) {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return nil, err
	}

	var config interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return config, nil
}

// monitor 监控进程
func (m *Manager) monitor() {
	if m.process == nil {
		return
	}

	m.process.Wait()

	m.mu.Lock()
	m.running = false
	m.mu.Unlock()
}
