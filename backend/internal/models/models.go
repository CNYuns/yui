package models

import (
	"time"

	"gorm.io/gorm"
)

// User 管理员账号
type User struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	Username   string         `gorm:"uniqueIndex;size:32" json:"username"`
	Email      string         `gorm:"size:255" json:"email,omitempty"` // 可选
	Password   string         `gorm:"size:255" json:"-"`
	Role       string         `gorm:"size:50;default:viewer" json:"role"` // admin, operator, viewer
	Nickname   string         `gorm:"size:100" json:"nickname"`
	TwoFA      bool           `gorm:"default:false" json:"two_fa"`
	TOTPSecret string         `gorm:"size:255" json:"-"`
	LastLogin  *time.Time     `json:"last_login,omitempty"`
	Status     int            `gorm:"default:1" json:"status"` // 1: active, 0: disabled
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// Client 业务用户
type Client struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	UUID        string         `gorm:"uniqueIndex;size:36" json:"uuid"`
	Email       string         `gorm:"size:255" json:"email"`
	Remark      string         `gorm:"size:500" json:"remark"`
	Enable      bool           `gorm:"default:true" json:"enable"`
	TotalGB     int64          `gorm:"default:0" json:"total_gb"`       // 总流量限制 (bytes)
	UsedGB      int64          `gorm:"default:0" json:"used_gb"`        // 已使用流量 (bytes)
	ExpireAt    *time.Time     `json:"expire_at,omitempty"`
	CreatedByID uint           `json:"created_by_id"`
	CreatedBy   *User          `gorm:"foreignKey:CreatedByID" json:"created_by,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// Inbound 入站配置
type Inbound struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Tag           string         `gorm:"uniqueIndex;size:100" json:"tag"`
	Protocol      string         `gorm:"size:50" json:"protocol"` // vmess, vless, trojan, shadowsocks
	Port          int            `gorm:"index" json:"port"`
	Listen        string         `gorm:"size:100;default:0.0.0.0" json:"listen"`
	Settings      string         `gorm:"type:text" json:"settings"`        // JSON 配置
	StreamSettings string        `gorm:"type:text" json:"stream_settings"` // 传输配置
	Sniffing      string         `gorm:"type:text" json:"sniffing"`
	Enable        bool           `gorm:"default:true" json:"enable"`
	Remark        string         `gorm:"size:500" json:"remark"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// Outbound 出站配置
type Outbound struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	Tag            string         `gorm:"uniqueIndex;size:100" json:"tag"`
	Protocol       string         `gorm:"size:50" json:"protocol"` // freedom, blackhole, vmess, etc.
	Settings       string         `gorm:"type:text" json:"settings"`
	StreamSettings string         `gorm:"type:text" json:"stream_settings"`
	ProxySettings  string         `gorm:"type:text" json:"proxy_settings"`
	Mux            string         `gorm:"type:text" json:"mux"`
	Enable         bool           `gorm:"default:true" json:"enable"`
	Remark         string         `gorm:"size:500" json:"remark"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// InboundClient 入站与用户关联表
type InboundClient struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	InboundID uint      `gorm:"index" json:"inbound_id"`
	ClientID  uint      `gorm:"index" json:"client_id"`
	Inbound   *Inbound  `gorm:"foreignKey:InboundID" json:"inbound,omitempty"`
	Client    *Client   `gorm:"foreignKey:ClientID" json:"client,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Certificate TLS 证书
type Certificate struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	Domain     string         `gorm:"uniqueIndex;size:255" json:"domain"`
	Email      string         `gorm:"size:255" json:"email"`
	CertPath   string         `gorm:"size:500" json:"cert_path"`
	KeyPath    string         `gorm:"size:500" json:"key_path"`
	ExpireAt   time.Time      `json:"expire_at"`
	AutoRenew  bool           `gorm:"default:true" json:"auto_renew"`
	Provider   string         `gorm:"size:50;default:letsencrypt" json:"provider"`
	Status     string         `gorm:"size:50;default:pending" json:"status"` // pending, active, expired, error
	LastError  string         `gorm:"type:text" json:"last_error,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// TrafficStats 流量统计
type TrafficStats struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ClientID  uint      `gorm:"index:idx_traffic_client_date" json:"client_id"`
	Client    *Client   `gorm:"foreignKey:ClientID" json:"client,omitempty"`
	InboundID uint      `gorm:"index:idx_traffic_inbound_date" json:"inbound_id"`
	Inbound   *Inbound  `gorm:"foreignKey:InboundID" json:"inbound,omitempty"`
	Upload    int64     `gorm:"default:0" json:"upload"`   // bytes
	Download  int64     `gorm:"default:0" json:"download"` // bytes
	Date      string    `gorm:"size:10;index:idx_traffic_client_date;index:idx_traffic_inbound_date;index:idx_traffic_date" json:"date"` // YYYY-MM-DD
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AuditLog 审计日志
type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Action    string    `gorm:"size:100" json:"action"`
	Resource  string    `gorm:"size:100" json:"resource"`
	ResourceID uint     `json:"resource_id"`
	Detail    string    `gorm:"type:text" json:"detail"`
	IP        string    `gorm:"size:50" json:"ip"`
	UserAgent string    `gorm:"size:500" json:"user_agent"`
	Status    string    `gorm:"size:50" json:"status"` // success, failed
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

// TableName 自定义表名
func (User) TableName() string           { return "users" }
func (Client) TableName() string         { return "clients" }
func (Inbound) TableName() string        { return "inbounds" }
func (Outbound) TableName() string       { return "outbounds" }
func (InboundClient) TableName() string  { return "inbound_clients" }
func (Certificate) TableName() string    { return "certificates" }
func (TrafficStats) TableName() string   { return "traffic_stats" }
func (AuditLog) TableName() string       { return "audit_logs" }
