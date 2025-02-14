// cmd/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"tp-santak-rtu/internal/config"
	"tp-santak-rtu/internal/handler"
	"tp-santak-rtu/internal/pkg/logger"
	"tp-santak-rtu/internal/platform"
	"tp-santak-rtu/internal/tcpserver"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/urfave/cli/v2"
)

func main() {
	// 首先设置基本的日志格式
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	logrus.Info("=================== SANTAK-RTU 插件服务启动 ===================")

	app := &cli.App{
		Name:    "tp-santak-rtu",
		Usage:   "tp-santak-rtu protocol plugin",
		Version: "0.0.1",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "../configs/config.yaml",
				Usage:   "config file path",
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		logrus.WithError(err).Fatal("程序运行失败")
	}
}

func run(c *cli.Context) error {
	// 1. 配置文件检查
	configPath := c.String("config")
	logrus.Infof("正在检查配置文件路径: %s", configPath)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		logrus.WithError(err).Errorf("配置文件不存在: %s", configPath)
		return fmt.Errorf("配置文件不存在: %s", configPath)
	}

	// 2. 加载配置
	logrus.Info("开始加载配置文件...")
	cfg, err := loadConfig(configPath)
	if err != nil {
		logrus.WithError(err).Error("加载配置文件失败")
		return fmt.Errorf("加载配置文件失败: %v", err)
	}
	logrus.WithFields(logrus.Fields{
		"port":            cfg.Server.Port,
		"http_port":       cfg.Server.HTTPPort,
		"max_connections": cfg.Server.MaxConnections,
		"heartbeat":       cfg.Server.HeartbeatTimeout,
		"log_level":       cfg.Log.Level,
		"log_path":        cfg.Log.FilePath,
		"url":             cfg.Platform.URL,
		"mqtt_broker":     cfg.Platform.MQTTBroker,
	}).Info("配置加载成功")

	// 3. 日志目录检查和初始化
	logrus.Info("正在初始化日志系统...")
	if err := ensureLogDir(cfg.Log.FilePath); err != nil {
		logrus.WithError(err).Error("创建日志目录失败")
		return fmt.Errorf("创建日志目录失败: %v", err)
	}
	logger.InitLogger(&cfg.Log)
	logrus.Info("日志系统初始化完成")

	// 4. 创建平台客户端
	logrus.Info("正在初始化平台客户端...")
	platformClient, err := platform.NewPlatformClient(platform.Config{
		BaseURL:      cfg.Platform.URL,
		MQTTBroker:   cfg.Platform.MQTTBroker,
		MQTTUsername: cfg.Platform.MQTTUsername,
		MQTTPassword: cfg.Platform.MQTTPassword,
	}, logrus.StandardLogger())
	if err != nil {
		return fmt.Errorf("创建平台客户端失败: %v", err)
	}
	defer platformClient.Close()
	logrus.Info("平台客户端初始化成功")

	// // 5. 创建并初始化服务管理器
	// logrus.Info("正在初始化服务管理器...")
	// serviceMgr := manager.NewServiceManager(
	// 	platformClient,
	// 	manager.Config{
	// 		UpdateInterval:  time.Minute * 1,        // 每分钟更新一次服务列表
	// 		ConnectTimeout:  time.Second * 30,       // 连接超时30秒
	// 		RequestTimeout:  time.Second * 10,       // 请求超时10秒
	// 		PublishInterval: time.Millisecond * 500, // 发布间隔500ms
	// 	},
	// 	logrus.StandardLogger(),
	// )

	// // 启动服务管理器
	// if err := serviceMgr.Start(); err != nil {
	// 	logrus.WithError(err).Error("启动服务管理器失败")
	// 	return fmt.Errorf("启动服务管理器失败: %v", err)
	// }
	// defer serviceMgr.Stop()
	// logrus.Info("服务管理器启动成功")

	// 6. 创建并启动HTTP服务
	httpHandler := handler.NewHTTPHandler(platformClient, logrus.StandardLogger())
	handlers := httpHandler.RegisterHandlers()
	httpPort := cfg.Server.HTTPPort
	go func() {
		logrus.Infof("正在启动HTTP服务，端口: %d", httpPort)
		if err := handlers.Start(fmt.Sprintf(":%d", httpPort)); err != nil {
			logrus.Errorf("HTTP服务启动失败: %v", err)
		}
	}()

	logrus.Info("插件HTTP服务启动成功")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go StartHeartbeatTask(ctx, platformClient, cfg.Platform.ServiceIdentifier)

	logrus.Info("心跳任务已启动")
	Port := cfg.Server.Port
	tcpServer := tcpserver.NewTCPServer(platformClient, fmt.Sprintf("%d", cfg.Server.Port), logrus.StandardLogger())
	go func() {
		logrus.Infof("正在启动TCP服务，端口: %d", Port)
		if err := tcpServer.Start(); err != nil {
			logrus.Errorf("TCP服务启动失败: %v", err)
		}
	}()
	// 7. 阻塞主goroutine,等待信号
	select {}
}

func loadConfig(configPath string) (*config.Config, error) {

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml") // 指定配置文件类型

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	// 从环境变量中加载配置
	viper.SetEnvPrefix("SANTAK") // 设置环境变量的前缀，例如 APP_PORT
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv() // 自动从环境变量中加载配置
	var cfg config.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func ensureLogDir(logPath string) error {
	dir := filepath.Dir(logPath)
	return os.MkdirAll(dir, 0755)
}

func StartHeartbeatTask(ctx context.Context, client *platform.PlatformClient, serviceIdentifier string) {
	ticker := time.NewTicker(30 * time.Second) // 创建一个每 30 秒触发一次的 Ticker
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done(): // 检查上下文是否被取消
			fmt.Println("心跳任务被取消")
			return
		case <-ticker.C: // 每 30 秒触发一次
			err := client.SendHeartbeat(ctx, serviceIdentifier)
			if err != nil {
				logrus.Errorf("发送心跳失败: %v\n", err)
			} else {
				logrus.Println("心跳发送成功")
			}
		}
	}
}
