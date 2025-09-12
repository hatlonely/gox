package provider

import (
	"sync"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ConfigData GORM 模型定义
type ConfigData struct {
	ID        string    `gorm:"primaryKey;column:id" json:"id"`
	Content   string    `gorm:"type:longtext;not null;column:content" json:"content"`
	Version   int64     `gorm:"autoIncrement;column:version" json:"version"`
	CreatedAt time.Time `gorm:"autoCreateTime;column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;column:updated_at" json:"updated_at"`
}

// TableName 指定表名
func (ConfigData) TableName() string {
	return "config_data"
}

// GormProvider 基于 GORM 的配置提供者
type GormProvider struct {
	configID    string
	db          *gorm.DB
	tableName   string
	mu          sync.RWMutex
	onChange    func(data []byte) error
	lastVersion int64

	// 变更监听
	stopChan     chan struct{}
	pollInterval time.Duration
}

// GormProviderOptions GORM Provider 配置选项
type GormProviderOptions struct {
	ConfigID     string                 // 配置 ID
	Driver       string                 // 数据库驱动：sqlite, mysql
	DSN          string                 // 数据源名称
	TableName    string                 // 表名，默认 config_data
	PollInterval time.Duration          // 轮询间隔，默认 5 秒
	GormConfig   *gorm.Config           // GORM 配置
	Extra        map[string]interface{} // 额外配置
}

// NewGormProviderWithOptions 创建 GORM Provider
func NewGormProviderWithOptions(options *GormProviderOptions) (*GormProvider, error) {
	if options == nil {
		return nil, &ProviderError{Msg: "gorm provider options is required"}
	}

	if options.ConfigID == "" {
		return nil, &ProviderError{Msg: "config ID is required"}
	}

	if options.Driver == "" {
		return nil, &ProviderError{Msg: "database driver is required"}
	}

	if options.DSN == "" {
		return nil, &ProviderError{Msg: "database DSN is required"}
	}

	if options.TableName == "" {
		options.TableName = "config_data"
	}

	if options.PollInterval == 0 {
		options.PollInterval = 5 * time.Second
	}

	if options.GormConfig == nil {
		options.GormConfig = &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		}
	}

	// 根据驱动类型创建 GORM 实例
	var db *gorm.DB
	var err error

	switch options.Driver {
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(options.DSN), options.GormConfig)
	case "mysql":
		db, err = gorm.Open(mysql.Open(options.DSN), options.GormConfig)
	default:
		return nil, &ProviderError{Msg: "unsupported database driver: " + options.Driver}
	}

	if err != nil {
		return nil, &ProviderError{Msg: "failed to connect database", Err: err}
	}

	provider := &GormProvider{
		configID:     options.ConfigID,
		db:           db,
		tableName:    options.TableName,
		pollInterval: options.PollInterval,
		stopChan:     make(chan struct{}),
	}

	// 自动迁移表结构
	if err := provider.autoMigrate(); err != nil {
		return nil, err
	}

	return provider, nil
}

// autoMigrate 自动迁移表结构
func (p *GormProvider) autoMigrate() error {
	// 设置自定义表名
	if p.tableName != "config_data" {
		err := p.db.Table(p.tableName).AutoMigrate(&ConfigData{})
		if err != nil {
			return &ProviderError{Msg: "failed to auto migrate table", Err: err}
		}
	} else {
		err := p.db.AutoMigrate(&ConfigData{})
		if err != nil {
			return &ProviderError{Msg: "failed to auto migrate table", Err: err}
		}
	}
	return nil
}

// Load 读取配置数据
func (p *GormProvider) Load() ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var config ConfigData
	result := p.db.Table(p.tableName).Where("id = ?", p.configID).First(&config)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, &ProviderError{Msg: "config not found: " + p.configID}
		}
		return nil, &ProviderError{Msg: "failed to load config", Err: result.Error}
	}

	p.lastVersion = config.Version
	return []byte(config.Content), nil
}

// Save 保存配置数据
func (p *GormProvider) Save(data []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 先查找是否存在
	var existingConfig ConfigData
	result := p.db.Table(p.tableName).Where("id = ?", p.configID).First(&existingConfig)

	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return &ProviderError{Msg: "failed to check existing config", Err: result.Error}
	}

	if result.Error == gorm.ErrRecordNotFound {
		// 不存在，执行插入
		config := ConfigData{
			ID:      p.configID,
			Content: string(data),
		}
		result = p.db.Table(p.tableName).Create(&config)
	} else {
		// 存在，执行更新（同时更新版本号）
		updates := map[string]interface{}{
			"content": string(data),
			"version": gorm.Expr("version + 1"),
		}
		result = p.db.Table(p.tableName).Where("id = ?", p.configID).Updates(updates)
	}

	if result.Error != nil {
		return &ProviderError{Msg: "failed to save config", Err: result.Error}
	}

	return nil
}

// OnChange 监听配置变更
func (p *GormProvider) OnChange(fn func(data []byte) error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.onChange = fn

	// 启动轮询监听
	go p.startPolling()
}

// startPolling 启动轮询监听配置变更
func (p *GormProvider) startPolling() {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.checkForChanges()
		case <-p.stopChan:
			return
		}
	}
}

// checkForChanges 检查配置变更
func (p *GormProvider) checkForChanges() {
	p.mu.RLock()
	onChange := p.onChange
	lastVersion := p.lastVersion
	p.mu.RUnlock()

	if onChange == nil {
		return
	}

	var config ConfigData
	result := p.db.Table(p.tableName).Where("id = ?", p.configID).First(&config)

	if result.Error != nil {
		return // 忽略错误，继续轮询
	}

	if config.Version > lastVersion {
		data := []byte(config.Content)
		if err := onChange(data); err == nil {
			p.mu.Lock()
			p.lastVersion = config.Version
			p.mu.Unlock()
		}
	}
}

// Close 关闭提供者，释放资源
func (p *GormProvider) Close() error {
	close(p.stopChan)

	sqlDB, err := p.db.DB()
	if err != nil {
		return &ProviderError{Msg: "failed to get underlying sql.DB", Err: err}
	}

	return sqlDB.Close()
}
