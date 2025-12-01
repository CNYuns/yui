package scheduler

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"y-ui/internal/database"
	"y-ui/internal/models"
	"y-ui/internal/services"
	"y-ui/internal/xray"

	"github.com/robfig/cron/v3"
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
		// Xray Stats API 可能未就绪，静默忽略
		return
	}

	// 获取所有用户流量
	userTraffic, err := s.statsClient.GetAllUserTraffic()
	if err != nil {
		// 静默忽略
		return
	}

	// 获取入站到ID的映射
	inbounds, _, _ := s.inboundService.List(1, 1000)
	inboundMap := make(map[string]uint)
	for _, ib := range inbounds {
		inboundMap[ib.Tag] = ib.ID
	}

	// 获取客户端邮箱到ID的映射
	var clients []models.Client
	database.DB.Find(&clients)
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
			s.trafficService.RecordTraffic(0, inboundID, traffic.Upload, traffic.Download)
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
			s.clientService.UpdateTraffic(clientID, totalBytes)

			// 记录到日流量统计（需要找到对应的入站）
			// 这里简化处理，只更新客户端总流量
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
		expiredClients = []models.Client{}
	}
	for _, client := range expiredClients {
		s.clientService.DisableClient(client.ID)
		needReload = true
	}

	// 检查超量的客户端
	overQuotaClients, err := s.clientService.GetOverQuotaClients()
	if err != nil {
		overQuotaClients = []models.Client{}
	}
	for _, client := range overQuotaClients {
		s.clientService.DisableClient(client.ID)
		needReload = true
	}

	// 如果有客户端被禁用，重载 Xray
	if needReload && s.xrayManager != nil {
		s.xrayManager.Reload()
	}
}

// renewCertificates 续签证书
func (s *Scheduler) renewCertificates() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 获取需要续签的证书（30天内过期）
	certs, err := s.certService.GetExpiringCertificates(30)
	if err != nil {
		certs = []models.Certificate{}
	}
	for _, cert := range certs {
		if cert.AutoRenew {
			s.certService.Renew(cert.ID)
		}
	}
}

// backupDatabase 备份数据库
func (s *Scheduler) backupDatabase() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查数据库文件是否存在
	if _, err := os.Stat(s.dbPath); os.IsNotExist(err) {
		return
	}

	// 创建备份目录
	if err := os.MkdirAll(s.backupDir, 0755); err != nil {
		return
	}

	// 生成备份文件名（带时间戳）
	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(s.backupDir, fmt.Sprintf("y-ui_%s.db", timestamp))

	// 复制数据库文件
	if err := copyFile(s.dbPath, backupFile); err != nil {
		return
	}

	// 设置备份文件权限
	os.Chmod(backupFile, 0600)

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
	s.trafficService.CleanOldStats(90)

	// 清理180天前的审计日志
	s.auditService.CleanOldLogs(180)
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
