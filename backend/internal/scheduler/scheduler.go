package scheduler

import (
	"sync"

	"y-ui/internal/models"
	"y-ui/internal/services"
	"y-ui/internal/xray"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron          *cron.Cron
	xrayManager   *xray.Manager
	clientService *services.ClientService
	trafficService *services.TrafficService
	auditService  *services.AuditService
	certService   *services.CertificateService
	mu            sync.Mutex
}

func NewScheduler(xrayManager *xray.Manager) *Scheduler {
	return &Scheduler{
		cron:          cron.New(cron.WithSeconds()),
		xrayManager:   xrayManager,
		clientService: services.NewClientService(),
		trafficService: services.NewTrafficService(),
		auditService:  services.NewAuditService(),
		certService:   services.NewCertificateService(),
	}
}

// Start 启动调度器
func (s *Scheduler) Start() {
	// 每5分钟收集流量
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

	// TODO: 从 Xray API 获取流量数据并更新到数据库
	// 这需要连接到 Xray 的 stats API
}

// checkQuota 检查用户配额
func (s *Scheduler) checkQuota() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查过期的客户端
	expiredClients, err := s.clientService.GetExpiredClients()
	if err != nil {
		// 记录错误但继续执行
		expiredClients = []models.Client{}
	}
	for _, client := range expiredClients {
		s.clientService.DisableClient(client.ID)
	}

	// 检查超量的客户端
	overQuotaClients, err := s.clientService.GetOverQuotaClients()
	if err != nil {
		// 记录错误但继续执行
		overQuotaClients = []models.Client{}
	}
	for _, client := range overQuotaClients {
		s.clientService.DisableClient(client.ID)
	}

	// 如果有客户端被禁用，重载 Xray
	if len(expiredClients) > 0 || len(overQuotaClients) > 0 {
		if s.xrayManager != nil {
			s.xrayManager.Reload()
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
		// 记录错误但继续执行
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

	// 数据库备份逻辑
	// 对于 SQLite，可以直接复制文件
	// TODO: 实现备份逻辑
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
