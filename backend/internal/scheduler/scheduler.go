package scheduler

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"y-ui/internal/database"
	"y-ui/internal/middleware"
	"y-ui/internal/models"
	"y-ui/internal/services"
	"y-ui/internal/xray"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type Scheduler struct {
	cron           *cron.Cron
	xrayManager    *xray.Manager
	statsClient    *xray.StatsClient
	clientService  *services.ClientService
	trafficService *services.TrafficService
	auditService   *services.AuditService
	certService    *services.CertificateService
	inboundService *services.InboundService
	mu             sync.Mutex
	dbPath         string
	backupDir      string
}

func NewScheduler(xrayManager *xray.Manager, dbPath string) *Scheduler {
	// 默认备份目录
	backupDir := filepath.Join(filepath.Dir(dbPath), "backups")

	return &Scheduler{
		cron:           cron.New(cron.WithSeconds()),
		xrayManager:    xrayManager,
		statsClient:    xray.NewStatsClient("127.0.0.1:10085"),
		clientService:  services.NewClientService(),
		trafficService: services.NewTrafficService(),
		auditService:   services.NewAuditService(),
		certService:    services.NewCertificateService(),
		inboundService: services.NewInboundService(),
		dbPath:         dbPath,
		backupDir:      backupDir,
	}
}

// Start 启动调度器
func (s *Scheduler) Start() {
	// 每5分钟收集��量
	s.cron.AddFunc("0 */5 * * * *", s.collectTraffic)

	// 每30分钟检查配额
	s.cron.AddFunc("0 */30 * * * *", s.checkQuota)

	// 每天凌晨2点续签证书
	s.cron.AddFunc("0 0 2 * * *", s.renewCertificates)

	// 每天凌晨3点备份数据库
	s.cron.AddFunc("0 0 3 * * *", s.backupDatabase)

	// 每周日凌晨4点清理旧数据
	s.cron.AddFunc("0 0 4 * * 0", s.cleanup)

	s.cron.Start()
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.cron.Stop()
}

// collectTraffic 收集流量数据
func (s *Scheduler) collectTraffic() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查 Xray 是否运行
	if s.xrayManager == nil || !s.xrayManager.IsRunning() {
		return
	}

	// 获取所有入站流量
	inboundTraffic, err := s.statsClient.GetAllInboundTraffic()
	if err != nil {
		// 记录错误日志而非静默忽略
		if middleware.Logger != nil {
			middleware.Logger.Debug("获取入站流量失败", zap.Error(err))
		}
		return
	}

	// 获取所有用户流量
	userTraffic, err := s.statsClient.GetAllUserTraffic()
	if err != nil {
		if middleware.Logger != nil {
			middleware.Logger.Debug("获取用户流量失败", zap.Error(err))
		}
		return
	}

	// 获取入站到ID的映射
	inbounds, _, err := s.inboundService.List(1, 1000)
	if err != nil {
		if middleware.Logger != nil {
			middleware.Logger.Error("获取入站列表失败", zap.Error(err))
		}
		return
	}
	inboundMap := make(map[string]uint)
	for _, ib := range inbounds {
		inboundMap[ib.Tag] = ib.ID
	}

	// 获取客户端邮箱到ID的映射
	var clients []models.Client
	if err := database.DB.Find(&clients).Error; err != nil {
		if middleware.Logger != nil {
			middleware.Logger.Error("获取客户端列表失败", zap.Error(err))
		}
		return
	}
	clientMap := make(map[string]uint)
	for _, c := range clients {
		if c.Email != "" {
			clientMap[c.Email] = c.ID
		}
	}

	// 记录入站流量
	for tag, traffic := range inboundTraffic {
		if traffic.Upload == 0 && traffic.Download == 0 {
			continue
		}
		if inboundID, ok := inboundMap[tag]; ok {
			// 记录到日流量统计
			if err := s.trafficService.RecordTraffic(0, inboundID, traffic.Upload, traffic.Download); err != nil {
				if middleware.Logger != nil {
					middleware.Logger.Error("记录入站流量失败", zap.String("tag", tag), zap.Error(err))
				}
			}
		}
	}

	// 记录用户流量并更新已用量
	for email, traffic := range userTraffic {
		if traffic.Upload == 0 && traffic.Download == 0 {
			continue
		}
		if clientID, ok := clientMap[email]; ok {
			// 更新客户端已用流量
			totalBytes := traffic.Upload + traffic.Download
			if err := s.clientService.UpdateTraffic(clientID, totalBytes); err != nil {
				if middleware.Logger != nil {
					middleware.Logger.Error("更新客户端流量失败", zap.String("email", email), zap.Error(err))
				}
			}
		}
	}
}

// checkQuota 检查用户配额
func (s *Scheduler) checkQuota() {
	s.mu.Lock()
	defer s.mu.Unlock()

	needReload := false

	// 检查过期的客户端
	expiredClients, err := s.clientService.GetExpiredClients()
	if err != nil {
		if middleware.Logger != nil {
			middleware.Logger.Error("获取过期客户端失败", zap.Error(err))
		}
		expiredClients = []models.Client{}
	}
	for _, client := range expiredClients {
		if err := s.clientService.DisableClient(client.ID); err != nil {
			if middleware.Logger != nil {
				middleware.Logger.Error("禁用过期客户端失败", zap.Uint("client_id", client.ID), zap.Error(err))
			}
		} else {
			needReload = true
			if middleware.Logger != nil {
				middleware.Logger.Info("已禁用过期客户端", zap.Uint("client_id", client.ID), zap.String("email", client.Email))
			}
		}
	}

	// 检查超量的客户端
	overQuotaClients, err := s.clientService.GetOverQuotaClients()
	if err != nil {
		if middleware.Logger != nil {
			middleware.Logger.Error("获取超量客户端失败", zap.Error(err))
		}
		overQuotaClients = []models.Client{}
	}
	for _, client := range overQuotaClients {
		if err := s.clientService.DisableClient(client.ID); err != nil {
			if middleware.Logger != nil {
				middleware.Logger.Error("禁用超量客户端失败", zap.Uint("client_id", client.ID), zap.Error(err))
			}
		} else {
			needReload = true
			if middleware.Logger != nil {
				middleware.Logger.Info("已禁用超量客户端", zap.Uint("client_id", client.ID), zap.String("email", client.Email))
			}
		}
	}

	// 如果有客户端被禁用，重载 Xray
	if needReload && s.xrayManager != nil {
		if err := s.xrayManager.Reload(); err != nil {
			if middleware.Logger != nil {
				middleware.Logger.Error("重载 Xray 失败", zap.Error(err))
			}
		}
	}
}

// renewCertificates 续签证书
func (s *Scheduler) renewCertificates() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 获取需要续签的证书（30天内过期）
	certs, err := s.certService.GetExpiringCertificates(30)
	if err != nil {
		if middleware.Logger != nil {
			middleware.Logger.Error("获取待续签证书失败", zap.Error(err))
		}
		return
	}
	for _, cert := range certs {
		if cert.AutoRenew {
			if err := s.certService.Renew(cert.ID); err != nil {
				if middleware.Logger != nil {
					middleware.Logger.Error("续签证书失败", zap.String("domain", cert.Domain), zap.Error(err))
				}
			} else {
				if middleware.Logger != nil {
					middleware.Logger.Info("证书续签成功", zap.String("domain", cert.Domain))
				}
			}
		}
	}
}

// backupDatabase 备份数据库
func (s *Scheduler) backupDatabase() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查数据库文件是否存在
	if _, err := os.Stat(s.dbPath); os.IsNotExist(err) {
		if middleware.Logger != nil {
			middleware.Logger.Warn("数据库文件不存在，跳过备份", zap.String("path", s.dbPath))
		}
		return
	}

	// 创建备份目录
	if err := os.MkdirAll(s.backupDir, 0755); err != nil {
		if middleware.Logger != nil {
			middleware.Logger.Error("创建备份目录失败", zap.String("dir", s.backupDir), zap.Error(err))
		}
		return
	}

	// 生成备份文件名（带时间戳）
	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(s.backupDir, fmt.Sprintf("y-ui_%s.db", timestamp))

	// 复制数据库文件
	if err := copyFile(s.dbPath, backupFile); err != nil {
		if middleware.Logger != nil {
			middleware.Logger.Error("备份数据库失败", zap.String("src", s.dbPath), zap.String("dst", backupFile), zap.Error(err))
		}
		return
	}

	// 设置备份文件权限
	if err := os.Chmod(backupFile, 0600); err != nil {
		if middleware.Logger != nil {
			middleware.Logger.Warn("设置备份文件权限失败", zap.String("file", backupFile), zap.Error(err))
		}
	}

	if middleware.Logger != nil {
		middleware.Logger.Info("数据库备份成功", zap.String("file", backupFile))
	}

	// 清理旧备份（保留最近7天）
	s.cleanOldBackups(7)
}

// cleanOldBackups 清理旧备份
func (s *Scheduler) cleanOldBackups(keepDays int) {
	cutoff := time.Now().AddDate(0, 0, -keepDays)

	entries, err := os.ReadDir(s.backupDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// 删除超过保留期的备份
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(s.backupDir, entry.Name()))
		}
	}
}

// cleanup 清理旧数据
func (s *Scheduler) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 清理90天前的流量统计
	if err := s.trafficService.CleanOldStats(90); err != nil {
		if middleware.Logger != nil {
			middleware.Logger.Error("清理旧流量统计失败", zap.Error(err))
		}
	} else {
		if middleware.Logger != nil {
			middleware.Logger.Info("已清理90天前的流量统计")
		}
	}

	// 清理180天前的审计日志
	if err := s.auditService.CleanOldLogs(180); err != nil {
		if middleware.Logger != nil {
			middleware.Logger.Error("清理旧审计日志失败", zap.Error(err))
		}
	} else {
		if middleware.Logger != nil {
			middleware.Logger.Info("已清理180天前的审计日志")
		}
	}
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
