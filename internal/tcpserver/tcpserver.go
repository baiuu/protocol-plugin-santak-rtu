// internal/tcpserver/tcpserver.go
package tcpserver

import (
	"bufio"
	"io"
	"math"
	"net"
	"strconv"
	"strings"
	"time"
	"tp-santak-rtu/internal/platform"

	"github.com/sirupsen/logrus"
)

// TCPServer 代表一个 TCP 服务器
type TCPServer struct {
	platform *platform.PlatformClient
	port     string
	logger   *logrus.Logger
}

// NewTCPServer 创建一个新的 TCP 服务器
func NewTCPServer(platform *platform.PlatformClient, port string, logger *logrus.Logger) *TCPServer {
	return &TCPServer{
		platform: platform,
		port:     port,
		logger:   logger,
	}
}

// Start 启动 TCP 服务器
func (s *TCPServer) Start() error {
	listener, err := net.Listen("tcp", ":"+s.port)
	if err != nil {
		s.logger.WithError(err).Error("启动 TCP 服务器失败")
		return err
	}
	defer listener.Close()

	s.logger.Infof("TCP 服务器启动成功，监听端口: %s", s.port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			s.logger.WithError(err).Error("接受 TCP 连接失败")
			continue
		}
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		go s.handleConnection(conn)
	}
}

// handleConnection 处理每个客户端连接
func (s *TCPServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	clientAddr := conn.RemoteAddr()
	s.logger.Infof("客户端连接: %s", clientAddr.String())
	var accessToken string
	var deviceReg string
	var res string
	reader := bufio.NewReader(conn)
	var deviceid string
	for {
		var buf [512]byte
		n, err := reader.Read(buf[:])
		if err != nil {
			if err == io.EOF {
				s.platform.ClearDeviceCacheByVoucher(accessToken)
				s.logger.Warnf("客户端主动断开连接: %s", clientAddr.String())
			} else {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					s.logger.Warnf("读取超时，执行额外逻辑")
					if deviceid != "" {
						s.platform.SendDeviceStatus(deviceid, "0") // 发送设备离线状态
						s.logger.Infof("设备更新状态离线: %s", deviceid)
					} else {
						s.logger.Warnf("设备为空，无法发送状态")
					}
					// 在这里执行额外的逻辑
					// 例如：关闭连接、记录日志、发送通知等
					conn.Close()
					break
				}
				s.logger.Errorf("读取客户端消息失败: %v", err)
			}
			break
		}
		message := string(buf[:n])
		// 打印客户端发送的消息
		if deviceReg == "" {
			s.logger.Infof("注册客户端消息: %s", message)
		} else {
			s.logger.Debugf("%s 客户端%s应答: %s", deviceReg, res, message)
		}
		if accessToken == "" {
			accessToken = "{\"santak_reg_pkg\":\"" + message + "\"}"
			deviceReg = message
			s.logger.Infof("获取设备AccessToken: %s", accessToken)
			device, err := s.platform.GetDeviceByVoucher(accessToken)
			if err != nil {
				s.logger.Infof("获取设备失败: %v", err)
				break
			}
			deviceid = device.ID
			s.logger.Infof("Device: %v", device)
			if device.ID != "" {
				// 处理消息
				s.platform.SendDeviceStatus(device.ID, "1") // 发送设备在线状态
				s.logger.Infof("设备更新状态在线: %s", deviceid)
				res = "WA"
				response := strings.TrimSpace(res)
				_, err = conn.Write([]byte(response + "\r"))
				if err != nil {
					s.logger.Errorf("发送响应失败: %v", err)
				}
				conn.SetReadDeadline(time.Now().Add(10 * time.Second))
			} else {
				s.logger.Warnf("验证失败，断开连接")
				s.platform.ClearDeviceCacheByVoucher(accessToken)
				s.logger.Warnf("客户端断开连接: %s", clientAddr.String())
				break
			}
		} else {
			if res == "WA" {
				parts := s.splitMessage(message)
				if len(parts) == 13 {
					err := s.waMessageUpload(parts, deviceid)
					if err != nil {
						s.logger.Errorf("%sWA上传数据失败: %v", deviceReg, err)
					}
				} else {
					s.logger.Debugf("%s数据WA长度不对", deviceReg)
				}
				res = "Q6"
				response := strings.TrimSpace(res)
				_, err = conn.Write([]byte(response + "\r"))
				if err != nil {
					s.logger.Errorf("发送响应失败: %v", err)
				}
				conn.SetReadDeadline(time.Now().Add(10 * time.Second))
			} else if res == "Q6" {
				parts := s.splitMessage(message)
				if len(parts) == 20 {
					err := s.q6MessageUpload(parts, deviceid)
					if err != nil {
						s.logger.Errorf("%sQA上传数据失败: %v", deviceReg, err)
					}
				} else {
					s.logger.Debugf("%s数据Q6长度不对", deviceReg)
				}
				res = "WA"
				response := strings.TrimSpace(res)
				_, err = conn.Write([]byte(response + "\r"))
				if err != nil {
					s.logger.Errorf("发送响应失败: %v", err)
				}
				conn.SetReadDeadline(time.Now().Add(10 * time.Second))
			} else {
				s.platform.ClearDeviceCacheByVoucher(accessToken)
				s.platform.SendDeviceStatus(deviceid, "0") // 发送设备离线状态
				s.logger.Infof("设备更新状态离线: %s", deviceid)
				s.logger.Errorf("未知的应答: %s", res)
				break
			}

		}
	}
}

func (s *TCPServer) splitMessage(message string) []string {
	//将消息切分为数组
	cleaned := strings.ReplaceAll(message, "(NAK\r", "")
	cleaned = strings.TrimPrefix(cleaned, "(")
	parts := strings.Fields(cleaned)
	return parts
}

func (s *TCPServer) waMessageUpload(message []string, deviceid string) error {
	//将WA消息解析发送到MQTT
	data := map[string]interface{}{
		"loadpower":            s.stringToFloadt32(message[0]),
		"loadvirtualpower":     s.stringToFloadt32(message[3]),
		"loadpercentage":       s.stringToFloadt32(message[11]),
		"utilityfailstatus":    s.SliceString(message[12], 1),
		"batterylowstatus":     s.SliceString(message[12], 2),
		"bypassstatus":         s.SliceString(message[12], 3),
		"upsfailedstatus":      s.SliceString(message[12], 4),
		"upstypestatus":        s.SliceString(message[12], 5),
		"testinprogressstatus": s.SliceString(message[12], 6),
		"shutdownstatus":       s.SliceString(message[12], 7),
	}
	s.logger.Infof("%s设备WA数据: %v", deviceid, data)
	s.platform.SendTelemetry(deviceid, data)
	return nil
}

func (s *TCPServer) q6MessageUpload(message []string, deviceid string) error {
	//将Q6消息解析发送到MQTT
	data := map[string]interface{}{
		"batterylevel":       s.stringToFloadt32(message[15]),
		"batterytemperature": s.stringToFloadt32(message[16]),
		"outputvoltage":      s.stringToFloadt32(message[4]),
		"inputfrequency":     s.stringToFloadt32(message[3]),
		"outputfrequency":    s.stringToFloadt32(message[7]),
		"batteryvoltage":     s.stringToFloadt32(message[11]),
		"inputvoltage":       s.stringToFloadt32(message[0]),
	}
	s.logger.Infof("%s设备Q6数据: %v", deviceid, data)
	s.platform.SendTelemetry(deviceid, data)
	return nil
}

func (s *TCPServer) stringToFloadt32(message string) interface{} {
	//将String消息解析Float32
	float64Value, err := strconv.ParseFloat(message, 32)
	var stringValue = "___._"
	if err != nil {
		s.logger.Errorf("转换失败: %v\n", err)
		return stringValue
	} else {
		roundedValue := math.Round(float64Value*10) / 10
		return roundedValue
	}
}

func (s *TCPServer) SliceString(message string, n int) interface{} {
	//将String消息拆分
	// 确保 n 不超过字符串长度
	if n > len(message) {
		s.logger.Errorf("输入参数超出字符串长度")
		return nil
	}
	// 截取字符串
	subStr := message[n-1 : n]

	// 转换为整数
	num, err := strconv.Atoi(subStr)
	if err != nil {
		s.logger.Errorf("转换整数失败: %v", err)
		return 0
	}

	return num
}
