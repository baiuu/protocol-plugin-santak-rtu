// internal/platform/platform.go
package platform

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ThingsPanel/tp-protocol-sdk-go/client"
	"github.com/ThingsPanel/tp-protocol-sdk-go/types"
	"github.com/sirupsen/logrus"
)

// PlatformClient 平台客户端
type PlatformClient struct {
	sdkClient   *client.Client
	logger      *logrus.Logger
	deviceCache map[string]*types.Device
	cacheMutex  sync.RWMutex
}

// Config 平台配置
type Config struct {
	BaseURL      string
	MQTTBroker   string
	MQTTUsername string
	MQTTPassword string
}

// NewPlatformClient 创建平台客户端
func NewPlatformClient(config Config, logger *logrus.Logger) (*PlatformClient, error) {
	sdkConfig := client.ClientConfig{
		BaseURL:      config.BaseURL,
		MQTTBroker:   config.MQTTBroker,
		MQTTUsername: config.MQTTUsername,
		MQTTPassword: config.MQTTPassword,
		MQTTClientID: fmt.Sprintf("SANTAK-RTU-%d", time.Now().Unix()),
	}

	sdkClient, err := client.NewClient(sdkConfig)
	if err != nil {
		return nil, err
	}

	if err := sdkClient.Connect(); err != nil {
		return nil, err
	}

	return &PlatformClient{
		sdkClient:   sdkClient,
		logger:      logger,
		deviceCache: make(map[string]*types.Device),
	}, nil
}

// GetDevice 获取设备信息(带缓存)
func (p *PlatformClient) GetDevice(deviceNumber string) (*types.Device, error) {
	// 先查缓存
	p.cacheMutex.RLock()
	if device, ok := p.deviceCache[deviceNumber]; ok {
		p.cacheMutex.RUnlock()
		return device, nil
	}
	p.cacheMutex.RUnlock()

	// 缓存未命中,从平台获取
	req := &client.DeviceConfigRequest{
		DeviceNumber: deviceNumber,
	}

	resp, err := p.sdkClient.Device().GetDeviceConfig(context.Background(), req)
	if err != nil {
		return nil, err
	}

	// 更新缓存
	p.cacheMutex.Lock()
	p.deviceCache[deviceNumber] = &resp.Data
	p.cacheMutex.Unlock()

	return &resp.Data, nil
}

// GetDeviceByVoucher 获取设备信息(带缓存)
func (p *PlatformClient) GetDeviceByVoucher(Voucher string) (*types.Device, error) {
	// 先查缓存
	p.cacheMutex.RLock()
	if device, ok := p.deviceCache[Voucher]; ok {
		p.cacheMutex.RUnlock()
		return device, nil
	}
	p.cacheMutex.RUnlock()

	// 缓存未命中,从平台获取
	req := &client.DeviceConfigRequest{
		Voucher: Voucher,
	}

	resp, err := p.sdkClient.Device().GetDeviceConfig(context.Background(), req)
	if err != nil {
		return nil, err
	}

	// 更新缓存
	p.cacheMutex.Lock()
	p.deviceCache[Voucher] = &resp.Data
	p.cacheMutex.Unlock()

	return &resp.Data, nil
}

// 获取服务接入点列表
func (p *PlatformClient) GetServiceAccessPoints() ([]types.ServiceAccessRsp, error) {
	req := &client.ServiceAccessRequest{
		ServiceIdentifier: "SANTAK-RTU",
	}
	resp, err := p.sdkClient.Service().GetServiceAccessList(context.Background(), req)
	if err != nil {
		return nil, err
	}
	if resp.Code != 200 {
		return nil, fmt.Errorf("获取服务接入点列表失败: code=%d, message=%s", resp.Code, resp.Message)
	}

	return resp.Data, nil
}

// ClearDeviceCache 清理指定设备的缓存
func (p *PlatformClient) ClearDeviceCache(deviceNumber string) {
	p.cacheMutex.Lock()
	delete(p.deviceCache, deviceNumber)
	p.cacheMutex.Unlock()
	p.logger.WithField("device_number", deviceNumber).Debug("设备缓存已清理")
}

// ClearDeviceCache 清理指定设备的缓存
func (p *PlatformClient) ClearDeviceCacheByVoucher(Voucher string) {
	p.cacheMutex.Lock()
	delete(p.deviceCache, Voucher)
	p.cacheMutex.Unlock()
	p.logger.WithField("voucher", Voucher).Debug("设备缓存已清理")
}

// GetDeviceByID 通过设备ID查找设备
func (p *PlatformClient) GetDeviceByID(deviceID string) (*types.Device, error) {
	var foundDevice *types.Device
	p.cacheMutex.RLock()
	for _, device := range p.deviceCache {
		if device.ID == deviceID {
			foundDevice = device
			break
		}
	}
	p.cacheMutex.RUnlock()
	if foundDevice != nil {
		return foundDevice, nil
	}
	return nil, fmt.Errorf("device not found")
}

// SendTelemetry 发送遥测数据
func (p *PlatformClient) SendTelemetry(deviceID string, values map[string]interface{}) error {
	// 1. 先将 values 转换为 JSON
	valuesJSON, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("序列化values失败: %v", err)
	}

	// 2. 将 JSON 进行 base64 编码
	valuesBase64 := base64.StdEncoding.EncodeToString(valuesJSON)

	// 3. 构造最终消息
	msg := map[string]interface{}{
		"device_id": deviceID,
		"values":    valuesBase64, // base64 编码的字符串
	}

	// 4. 将整个消息转换为 JSON
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	// 5. 发送消息
	if err := p.sdkClient.MQTT().Publish("devices/telemetry", 1, string(payload)); err != nil {
		return fmt.Errorf("发送消息失败: %v", err)
	}

	p.logger.WithFields(logrus.Fields{
		"device_id": deviceID,
	}).Debug("遥测数据发送成功", string(valuesJSON))

	return nil
}

// Close 关闭客户端
func (p *PlatformClient) Close() {
	if p.sdkClient != nil {
		p.sdkClient.Close()
	}
}

func (p *PlatformClient) SendDeviceStatus(deviceID string, msg interface{}) error {
	logrus.WithField("deviceID", deviceID).Debugf("发送设备状态: %s", msg)

	return p.sdkClient.MQTT().Publish("devices/status/"+deviceID, 1, msg)
}

// SendHeartbeat 发送插件心跳
func (p *PlatformClient) SendHeartbeat(ctx context.Context, serviceIdentifier string) error {
	req := &client.HeartbeatRequest{
		ServiceIdentifier: serviceIdentifier,
	}

	resp, err := p.sdkClient.Service().SendHeartbeat(ctx, req)
	if err != nil {
		return fmt.Errorf("发送心跳失败: %v", err)
	}

	if resp.Code != 200 {
		return fmt.Errorf("心跳响应异常: code=%d, message=%s", resp.Code, resp.Message)
	}

	return nil
}
