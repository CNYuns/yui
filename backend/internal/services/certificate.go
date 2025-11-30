package services

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"time"

	"y-ui/internal/database"
	"y-ui/internal/models"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/http01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
)

type CertificateService struct{}

func NewCertificateService() *CertificateService {
	return &CertificateService{}
}

type ACMEUser struct {
	Email        string
	Registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *ACMEUser) GetEmail() string {
	return u.Email
}

func (u *ACMEUser) GetRegistration() *registration.Resource {
	return u.Registration
}

func (u *ACMEUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

type RequestCertRequest struct {
	Domain string `json:"domain" binding:"required"`
	Email  string `json:"email" binding:"required,email"`
}

// List 获取证书列表
func (s *CertificateService) List(page, pageSize int) ([]models.Certificate, int64, error) {
	var certs []models.Certificate
	var total int64

	db := database.DB.Model(&models.Certificate{})
	db.Count(&total)

	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Find(&certs).Error; err != nil {
		return nil, 0, err
	}

	return certs, total, nil
}

// Get 获取单个证书
func (s *CertificateService) Get(id uint) (*models.Certificate, error) {
	var cert models.Certificate
	if err := database.DB.First(&cert, id).Error; err != nil {
		return nil, errors.New("证书不存在")
	}
	return &cert, nil
}

// Request 申请证书
func (s *CertificateService) Request(req *RequestCertRequest, certDir string) (*models.Certificate, error) {
	// 检查是否已存在
	var count int64
	database.DB.Model(&models.Certificate{}).Where("domain = ?", req.Domain).Count(&count)
	if count > 0 {
		return nil, errors.New("该域名证书已存在")
	}

	// 生成私钥
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	user := &ACMEUser{
		Email: req.Email,
		key:   privateKey,
	}

	config := lego.NewConfig(user)
	config.Certificate.KeyType = certcrypto.RSA2048

	client, err := lego.NewClient(config)
	if err != nil {
		return nil, err
	}

	// 使用 HTTP-01 挑战
	err = client.Challenge.SetHTTP01Provider(http01.NewProviderServer("", "80"))
	if err != nil {
		return nil, err
	}

	// 注册用户
	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return nil, err
	}
	user.Registration = reg

	// 申请证书
	request := certificate.ObtainRequest{
		Domains: []string{req.Domain},
		Bundle:  true,
	}

	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		// 保存错误状态
		cert := models.Certificate{
			Domain:    req.Domain,
			Email:     req.Email,
			Status:    "error",
			LastError: err.Error(),
		}
		database.DB.Create(&cert)
		return nil, err
	}

	// 保存证书文件
	domainDir := filepath.Join(certDir, req.Domain)
	if err := os.MkdirAll(domainDir, 0755); err != nil {
		return nil, err
	}

	certPath := filepath.Join(domainDir, "fullchain.pem")
	keyPath := filepath.Join(domainDir, "privkey.pem")

	if err := os.WriteFile(certPath, certificates.Certificate, 0600); err != nil {
		return nil, err
	}
	if err := os.WriteFile(keyPath, certificates.PrivateKey, 0600); err != nil {
		return nil, err
	}

	// 保存到数据库
	cert := models.Certificate{
		Domain:    req.Domain,
		Email:     req.Email,
		CertPath:  certPath,
		KeyPath:   keyPath,
		ExpireAt:  time.Now().Add(90 * 24 * time.Hour), // Let's Encrypt 证书有效期90天
		AutoRenew: true,
		Provider:  "letsencrypt",
		Status:    "active",
	}

	if err := database.DB.Create(&cert).Error; err != nil {
		return nil, err
	}

	return &cert, nil
}

// Renew 续签证书
func (s *CertificateService) Renew(id uint) error {
	cert, err := s.Get(id)
	if err != nil {
		return err
	}

	// 生成私钥
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	user := &ACMEUser{
		Email: cert.Email,
		key:   privateKey,
	}

	config := lego.NewConfig(user)
	config.Certificate.KeyType = certcrypto.RSA2048

	client, err := lego.NewClient(config)
	if err != nil {
		return err
	}

	err = client.Challenge.SetHTTP01Provider(http01.NewProviderServer("", "80"))
	if err != nil {
		return err
	}

	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return err
	}
	user.Registration = reg

	// 续签
	request := certificate.ObtainRequest{
		Domains: []string{cert.Domain},
		Bundle:  true,
	}

	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		database.DB.Model(cert).Updates(map[string]interface{}{
			"status":     "error",
			"last_error": err.Error(),
		})
		return err
	}

	// 更新证书文件
	if err := os.WriteFile(cert.CertPath, certificates.Certificate, 0600); err != nil {
		return err
	}
	if err := os.WriteFile(cert.KeyPath, certificates.PrivateKey, 0600); err != nil {
		return err
	}

	// 更新数据库
	return database.DB.Model(cert).Updates(map[string]interface{}{
		"expire_at":  time.Now().Add(90 * 24 * time.Hour),
		"status":     "active",
		"last_error": "",
	}).Error
}

// Delete 删除证书
func (s *CertificateService) Delete(id uint) error {
	cert, err := s.Get(id)
	if err != nil {
		return err
	}

	// 删除文件（忽略不存在的错误）
	if cert.CertPath != "" {
		if err := os.Remove(cert.CertPath); err != nil && !os.IsNotExist(err) {
			// 记录错误但继续删除
		}
	}
	if cert.KeyPath != "" {
		if err := os.Remove(cert.KeyPath); err != nil && !os.IsNotExist(err) {
			// 记录错误但继续删除
		}
	}
	// 尝试删除目录（如果为空）
	if cert.CertPath != "" {
		dir := filepath.Dir(cert.CertPath)
		os.Remove(dir) // 忽略错误，目录可能非空
	}

	return database.DB.Delete(cert).Error
}

// GetExpiringCertificates 获取即将过期的证书
func (s *CertificateService) GetExpiringCertificates(days int) ([]models.Certificate, error) {
	var certs []models.Certificate
	expireTime := time.Now().AddDate(0, 0, days)
	err := database.DB.Where("expire_at < ? AND status = ?", expireTime, "active").Find(&certs).Error
	return certs, err
}

// UpdateAutoRenew 更新自动续���设置
func (s *CertificateService) UpdateAutoRenew(id uint, autoRenew bool) error {
	return database.DB.Model(&models.Certificate{}).Where("id = ?", id).Update("auto_renew", autoRenew).Error
}
