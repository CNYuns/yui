package xray

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// StatsClient Xray 统计 API 客户端
type StatsClient struct {
	apiAddr string
	client  *http.Client
}

// TrafficData 流量数据
type TrafficData struct {
	Tag      string `json:"tag"`
	Upload   int64  `json:"upload"`
	Download int64  `json:"download"`
}

// UserTrafficData 用户流量数据
type UserTrafficData struct {
	Email    string `json:"email"`
	Upload   int64  `json:"upload"`
	Download int64  `json:"download"`
}

// NewStatsClient 创建统计客户端
func NewStatsClient(apiAddr string) *StatsClient {
	return &StatsClient{
		apiAddr: apiAddr,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetInboundTraffic 获取入站流量（通过命令行方式）
func (s *StatsClient) GetInboundTraffic(tag string) (*TrafficData, error) {
	data := &TrafficData{Tag: tag}

	// 获取上行流量
	uplink, err := s.queryStat(fmt.Sprintf("inbound>>>%s>>>traffic>>>uplink", tag))
	if err == nil {
		data.Upload = uplink
	}

	// 获取下行流量
	downlink, err := s.queryStat(fmt.Sprintf("inbound>>>%s>>>traffic>>>downlink", tag))
	if err == nil {
		data.Download = downlink
	}

	return data, nil
}

// GetUserTraffic 获取用户流量
func (s *StatsClient) GetUserTraffic(email string) (*UserTrafficData, error) {
	data := &UserTrafficData{Email: email}

	// 获取上行流量
	uplink, err := s.queryStat(fmt.Sprintf("user>>>%s>>>traffic>>>uplink", email))
	if err == nil {
		data.Upload = uplink
	}

	// 获取下行流量
	downlink, err := s.queryStat(fmt.Sprintf("user>>>%s>>>traffic>>>downlink", email))
	if err == nil {
		data.Download = downlink
	}

	return data, nil
}

// GetAllInboundTraffic 获取所有入站流量
func (s *StatsClient) GetAllInboundTraffic() (map[string]*TrafficData, error) {
	result := make(map[string]*TrafficData)

	// 查询所有统计
	stats, err := s.queryAllStats()
	if err != nil {
		return result, err
	}

	// 解析入站���量
	for name, value := range stats {
		parts := strings.Split(name, ">>>")
		if len(parts) >= 4 && parts[0] == "inbound" {
			tag := parts[1]
			direction := parts[3]

			if _, exists := result[tag]; !exists {
				result[tag] = &TrafficData{Tag: tag}
			}

			if direction == "uplink" {
				result[tag].Upload = value
			} else if direction == "downlink" {
				result[tag].Download = value
			}
		}
	}

	return result, nil
}

// GetAllUserTraffic 获取所有用户流量
func (s *StatsClient) GetAllUserTraffic() (map[string]*UserTrafficData, error) {
	result := make(map[string]*UserTrafficData)

	// 查询所有统计
	stats, err := s.queryAllStats()
	if err != nil {
		return result, err
	}

	// 解析用户流量
	for name, value := range stats {
		parts := strings.Split(name, ">>>")
		if len(parts) >= 4 && parts[0] == "user" {
			email := parts[1]
			direction := parts[3]

			if _, exists := result[email]; !exists {
				result[email] = &UserTrafficData{Email: email}
			}

			if direction == "uplink" {
				result[email].Upload = value
			} else if direction == "downlink" {
				result[email].Download = value
			}
		}
	}

	return result, nil
}

// queryStat 查询单个统计项
func (s *StatsClient) queryStat(name string) (int64, error) {
	url := fmt.Sprintf("http://%s/stats/query?name=%s&reset=true", s.apiAddr, name)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result struct {
		Stat struct {
			Value int64 `json:"value"`
		} `json:"stat"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	return result.Stat.Value, nil
}

// queryAllStats 查询所有统计
func (s *StatsClient) queryAllStats() (map[string]int64, error) {
	url := fmt.Sprintf("http://%s/stats/query?reset=true", s.apiAddr)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Stat []struct {
			Name  string `json:"name"`
			Value int64  `json:"value"`
		} `json:"stat"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	stats := make(map[string]int64)
	for _, s := range result.Stat {
		stats[s.Name] = s.Value
	}

	return stats, nil
}

// ResetStats 重置统计
func (s *StatsClient) ResetStats() error {
	url := fmt.Sprintf("http://%s/stats/query?reset=true", s.apiAddr)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
