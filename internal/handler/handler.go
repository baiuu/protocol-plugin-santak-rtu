// internal/handler/handler.go
package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	formjson "tp-santak-rtu/internal/form_json"
	"tp-santak-rtu/internal/platform"

	"github.com/ThingsPanel/tp-protocol-sdk-go/handler"
	"github.com/sirupsen/logrus"
)

// logrusWriter 实现 io.Writer 接口用于适配logrus
type logrusWriter struct {
	logger *logrus.Logger
}

func (w *logrusWriter) Write(p []byte) (n int, err error) {
	w.logger.Info(string(p))
	return len(p), nil
}

// HTTPHandler HTTP服务处理器
type HTTPHandler struct {
	platform *platform.PlatformClient
	logger   *logrus.Logger
	stdlog   *log.Logger
}

// NewHTTPHandler 创建HTTP处理器
func NewHTTPHandler(platform *platform.PlatformClient, logger *logrus.Logger) *HTTPHandler {
	// 创建适配器
	writer := &logrusWriter{logger: logger}
	stdlog := log.New(writer, "[HTTP] ", log.Ldate|log.Ltime|log.Lshortfile)

	return &HTTPHandler{
		platform: platform,
		logger:   logger,
		stdlog:   stdlog,
	}
}

// RegisterHandlers 注册所有HTTP处理器
func (h *HTTPHandler) RegisterHandlers() *handler.Handler {
	// 创建处理器，使用标准库Logger
	hdl := handler.NewHandler(handler.HandlerConfig{
		Logger: h.stdlog,
	})

	// 设置表单配置处理函数
	hdl.SetFormConfigHandler(h.handleGetFormConfig)

	// 设置设备断开连接处理函数
	hdl.SetDeviceDisconnectHandler(h.handleDeviceDisconnect)

	// 设置通知处理函数
	hdl.SetNotificationHandler(h.handleNotification)

	// 设置获取设备列表处理函数
	hdl.SetGetDeviceListHandler(h.handleGetDeviceList)

	return hdl
}

// handleGetFormConfig 处理获取表单配置请求
func (h *HTTPHandler) handleGetFormConfig(req *handler.GetFormConfigRequest) (interface{}, error) {
	h.logger.WithFields(logrus.Fields{
		"protocol_type": req.ProtocolType,
		"device_type":   req.DeviceType,
		"form_type":     req.FormType,
	}).Info("收到获取表单配置请求")

	if req.ProtocolType == "SANTAK-RTU" && req.DeviceType == "1" {
		// 根据请求类型返回不同的配置表单
		switch req.FormType {
		case "CFG": // 设备配置表单
			return nil, nil
		case "VCR": // 设备凭证表单
			return readFormConfigByPath("../internal/form_json/form_voucher.json"), nil
		case "VCRT": // 设备凭证表单
			return readFormConfigByPath("../internal/form_json/form_voucherT.json"), nil
		case "SVCR": // 服务接入点凭证表单
			return nil, nil
		default:
			return nil, fmt.Errorf("不支持的表单类型: %s", req.FormType)
		}
	} else {
		return nil, fmt.Errorf("不符合插件要求: %s,%s", req.ProtocolType, req.DeviceType)
	}
}

// ./form_config.json
func readFormConfigByPath(path string) interface{} {
	filePtr, err := os.Open(path)
	if err != nil {
		logrus.Warn("文件打开失败...", err.Error())
		return nil
	}
	defer filePtr.Close()
	var info interface{}
	// 创建json解码器
	decoder := json.NewDecoder(filePtr)
	err = decoder.Decode(&info)
	if err != nil {
		logrus.Warn("解码失败", err.Error())
		return info
	} else {
		logrus.Infof("读取文件%s成功...", path)
		return info
	}
}

// handleDeviceDisconnect 处理设备断开连接请求
func (h *HTTPHandler) handleDeviceDisconnect(req *handler.DeviceDisconnectRequest) error {
	h.logger.WithField("device_id", req.DeviceID).Info("收到设备断开连接请求")

	// 清理设备缓存
	// Note: 因为原缓存是按 device_number 存储的,这里要先查出设备信息
	device, err := h.platform.GetDeviceByID(req.DeviceID)
	if err == nil { // 如果能找到设备就清理缓存
		h.platform.ClearDeviceCache(device.DeviceNumber)
	}

	// 发送设备离线状态
	err = h.platform.SendDeviceStatus(req.DeviceID, "0")
	if err != nil {
		h.logger.WithError(err).Error("发送设备离线状态失败")
		return err
	}

	return nil
}

// handleNotification 处理通知请求
func (h *HTTPHandler) handleNotification(req *handler.NotificationRequest) error {
	h.logger.WithFields(logrus.Fields{
		"message_type": req.MessageType,
		"message":      req.Message,
	}).Info("收到通知请求")

	// 解析消息内容
	var msgData map[string]interface{}
	if err := json.Unmarshal([]byte(req.Message), &msgData); err != nil {
		h.logger.WithError(err).Error("解析通知消息失败")
		return err
	}

	// 处理不同类型的通知
	switch req.MessageType {
	case "1": // 服务配置修改
		h.logger.Info("处理服务配置修改通知")
		// TODO: 实现服务配置修改逻辑
	case "2": // 设备配置修改
		h.logger.Info("处理设备配置修改通知")
		// TODO: 实现设备配置修改逻辑
	default:
		h.logger.Warnf("未知的通知类型: %s", req.MessageType)
	}

	return nil
}

// handleGetDeviceList 处理获取设备列表请求
func (h *HTTPHandler) handleGetDeviceList(req *handler.GetDeviceListRequest) (*handler.DeviceListResponse, error) {
	h.logger.WithFields(logrus.Fields{
		"voucher":            req.Voucher,
		"service_identifier": req.ServiceIdentifier,
		"page":               req.Page,
		"page_size":          req.PageSize,
	}).Info("收到获取设备列表请求")

	// 讲req的Voucher解析到formjson.SVCRForm结构体
	var svcrForm formjson.SVCRForm
	if err := json.Unmarshal([]byte(req.Voucher), &svcrForm); err != nil {
		h.logger.WithError(err).Error("解析凭证失败")
		return nil, err
	}

	rsp := handler.DeviceListResponse{
		Code:    200,
		Message: "获取成功",
	}

	return &rsp, nil
}
