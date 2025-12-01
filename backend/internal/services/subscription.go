package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"y-ui/internal/database"
	"y-ui/internal/models"
)

type SubscriptionService struct{}

func NewSubscriptionService() *SubscriptionService {
	return &SubscriptionService{}
}

// GenerateClientLinks 生成客户端的所有订阅链接
func (s *SubscriptionService) GenerateClientLinks(clientID uint, serverAddr string) ([]map[string]interface{}, error) {
	var client models.Client
	if err := database.DB.First(&client, clientID).Error; err != nil {
		return nil, err
	}

	// 获取该客户端关联的所有入站
	var inboundClients []models.InboundClient
	if err := database.DB.Where("client_id = ?", clientID).Find(&inboundClients).Error; err != nil {
		return nil, err
	}

	var links []map[string]interface{}
	for _, ic := range inboundClients {
		var inbound models.Inbound
		if err := database.DB.First(&inbound, ic.InboundID).Error; err != nil {
			continue
		}
		if !inbound.Enable {
			continue
		}

		link, err := s.generateLink(&client, &inbound, serverAddr)
		if err != nil {
			continue
		}

		links = append(links, map[string]interface{}{
			"protocol": inbound.Protocol,
			"tag":      inbound.Tag,
			"port":     inbound.Port,
			"link":     link,
			"remark":   inbound.Remark,
		})
	}

	return links, nil
}

// GenerateSubscription 生成 Base64 编码的订阅内容
func (s *SubscriptionService) GenerateSubscription(clientID uint, serverAddr string) (string, error) {
	links, err := s.GenerateClientLinks(clientID, serverAddr)
	if err != nil {
		return "", err
	}

	var allLinks []string
	for _, l := range links {
		if link, ok := l["link"].(string); ok {
			allLinks = append(allLinks, link)
		}
	}

	content := strings.Join(allLinks, "\n")
	return base64.StdEncoding.EncodeToString([]byte(content)), nil
}

// generateLink 根据协议生成单个链接
func (s *SubscriptionService) generateLink(client *models.Client, inbound *models.Inbound, serverAddr string) (string, error) {
	var streamSettings map[string]interface{}
	if inbound.StreamSettings != "" {
		json.Unmarshal([]byte(inbound.StreamSettings), &streamSettings)
	}

	network := "tcp"
	security := "none"
	if streamSettings != nil {
		if n, ok := streamSettings["network"].(string); ok {
			network = n
		}
		if sec, ok := streamSettings["security"].(string); ok {
			security = sec
		}
	}

	remark := inbound.Remark
	if remark == "" {
		remark = inbound.Tag
	}

	switch inbound.Protocol {
	case "vmess":
		return s.generateVMessLink(client, inbound, serverAddr, network, security, streamSettings, remark)
	case "vless":
		return s.generateVLESSLink(client, inbound, serverAddr, network, security, streamSettings, remark)
	case "trojan":
		return s.generateTrojanLink(client, inbound, serverAddr, network, security, streamSettings, remark)
	case "shadowsocks":
		return s.generateSSLink(client, inbound, serverAddr, remark)
	default:
		return "", fmt.Errorf("unsupported protocol: %s", inbound.Protocol)
	}
}

// generateVMessLink 生成 VMess 链接
func (s *SubscriptionService) generateVMessLink(client *models.Client, inbound *models.Inbound, serverAddr, network, security string, streamSettings map[string]interface{}, remark string) (string, error) {
	vmessConfig := map[string]interface{}{
		"v":    "2",
		"ps":   remark,
		"add":  serverAddr,
		"port": inbound.Port,
		"id":   client.UUID,
		"aid":  0,
		"scy":  "auto",
		"net":  network,
		"type": "none",
		"host": "",
		"path": "",
		"tls":  "",
		"sni":  "",
	}

	if security == "tls" {
		vmessConfig["tls"] = "tls"
		if tlsSettings, ok := streamSettings["tlsSettings"].(map[string]interface{}); ok {
			if sni, ok := tlsSettings["serverName"].(string); ok {
				vmessConfig["sni"] = sni
			}
		}
	}

	// 处理传输配置
	if network == "ws" {
		if wsSettings, ok := streamSettings["wsSettings"].(map[string]interface{}); ok {
			if path, ok := wsSettings["path"].(string); ok {
				vmessConfig["path"] = path
			}
			if headers, ok := wsSettings["headers"].(map[string]interface{}); ok {
				if host, ok := headers["Host"].(string); ok {
					vmessConfig["host"] = host
				}
			}
		}
	} else if network == "grpc" {
		if grpcSettings, ok := streamSettings["grpcSettings"].(map[string]interface{}); ok {
			if serviceName, ok := grpcSettings["serviceName"].(string); ok {
				vmessConfig["path"] = serviceName
			}
		}
	} else if network == "h2" {
		if httpSettings, ok := streamSettings["httpSettings"].(map[string]interface{}); ok {
			if path, ok := httpSettings["path"].(string); ok {
				vmessConfig["path"] = path
			}
			if hosts, ok := httpSettings["host"].([]interface{}); ok && len(hosts) > 0 {
				if host, ok := hosts[0].(string); ok {
					vmessConfig["host"] = host
				}
			}
		}
	}

	jsonData, _ := json.Marshal(vmessConfig)
	return "vmess://" + base64.StdEncoding.EncodeToString(jsonData), nil
}

// generateVLESSLink 生成 VLESS 链接
func (s *SubscriptionService) generateVLESSLink(client *models.Client, inbound *models.Inbound, serverAddr, network, security string, streamSettings map[string]interface{}, remark string) (string, error) {
	params := url.Values{}
	params.Set("type", network)
	params.Set("security", security)

	if security == "tls" {
		if tlsSettings, ok := streamSettings["tlsSettings"].(map[string]interface{}); ok {
			if sni, ok := tlsSettings["serverName"].(string); ok && sni != "" {
				params.Set("sni", sni)
			}
			if alpn, ok := tlsSettings["alpn"].([]interface{}); ok {
				var alpnStrs []string
				for _, a := range alpn {
					if aStr, ok := a.(string); ok {
						alpnStrs = append(alpnStrs, aStr)
					}
				}
				if len(alpnStrs) > 0 {
					params.Set("alpn", strings.Join(alpnStrs, ","))
				}
			}
		}
	} else if security == "reality" {
		if realitySettings, ok := streamSettings["realitySettings"].(map[string]interface{}); ok {
			if sni, ok := realitySettings["serverNames"].([]interface{}); ok && len(sni) > 0 {
				if sniStr, ok := sni[0].(string); ok {
					params.Set("sni", sniStr)
				}
			}
			if pbk, ok := realitySettings["publicKey"].(string); ok {
				params.Set("pbk", pbk)
			}
			if fp, ok := realitySettings["fingerprint"].(string); ok {
				params.Set("fp", fp)
			}
		}
	}

	// 处理传输配置
	if network == "ws" {
		if wsSettings, ok := streamSettings["wsSettings"].(map[string]interface{}); ok {
			if path, ok := wsSettings["path"].(string); ok {
				params.Set("path", path)
			}
			if headers, ok := wsSettings["headers"].(map[string]interface{}); ok {
				if host, ok := headers["Host"].(string); ok {
					params.Set("host", host)
				}
			}
		}
	} else if network == "grpc" {
		if grpcSettings, ok := streamSettings["grpcSettings"].(map[string]interface{}); ok {
			if serviceName, ok := grpcSettings["serviceName"].(string); ok {
				params.Set("serviceName", serviceName)
			}
		}
		params.Set("mode", "gun")
	}

	link := fmt.Sprintf("vless://%s@%s:%d?%s#%s",
		client.UUID,
		serverAddr,
		inbound.Port,
		params.Encode(),
		url.QueryEscape(remark),
	)

	return link, nil
}

// generateTrojanLink 生成 Trojan 链接
func (s *SubscriptionService) generateTrojanLink(client *models.Client, inbound *models.Inbound, serverAddr, network, security string, streamSettings map[string]interface{}, remark string) (string, error) {
	params := url.Values{}
	params.Set("type", network)
	params.Set("security", security)

	if security == "tls" {
		if tlsSettings, ok := streamSettings["tlsSettings"].(map[string]interface{}); ok {
			if sni, ok := tlsSettings["serverName"].(string); ok && sni != "" {
				params.Set("sni", sni)
			}
		}
	}

	// 处理传输配置
	if network == "ws" {
		if wsSettings, ok := streamSettings["wsSettings"].(map[string]interface{}); ok {
			if path, ok := wsSettings["path"].(string); ok {
				params.Set("path", path)
			}
		}
	} else if network == "grpc" {
		if grpcSettings, ok := streamSettings["grpcSettings"].(map[string]interface{}); ok {
			if serviceName, ok := grpcSettings["serviceName"].(string); ok {
				params.Set("serviceName", serviceName)
			}
		}
		params.Set("mode", "gun")
	}

	// Trojan 使用 UUID 作为密码
	link := fmt.Sprintf("trojan://%s@%s:%d?%s#%s",
		client.UUID,
		serverAddr,
		inbound.Port,
		params.Encode(),
		url.QueryEscape(remark),
	)

	return link, nil
}

// generateSSLink 生成 Shadowsocks 链接
func (s *SubscriptionService) generateSSLink(client *models.Client, inbound *models.Inbound, serverAddr string, remark string) (string, error) {
	// SS 需要从 inbound settings 中获取加密方式
	var settings map[string]interface{}
	if inbound.Settings != "" {
		json.Unmarshal([]byte(inbound.Settings), &settings)
	}

	method := "aes-256-gcm"
	if settings != nil {
		if m, ok := settings["method"].(string); ok {
			method = m
		}
	}

	// SS 链接格式: ss://base64(method:password)@server:port#remark
	userInfo := base64.URLEncoding.EncodeToString([]byte(method + ":" + client.UUID))
	link := fmt.Sprintf("ss://%s@%s:%d#%s",
		userInfo,
		serverAddr,
		inbound.Port,
		url.QueryEscape(remark),
	)

	return link, nil
}
