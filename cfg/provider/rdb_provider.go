package provider

import (
	"context"
	"sync"
	"time"

	"github.com/hatlonely/gox/rdb"
	"github.com/hatlonely/gox/rdb/query"
	"github.com/hatlonely/gox/rdb/repository"
	"github.com/hatlonely/gox/ref"
	"github.com/pkg/errors"
)

// RdbConfigData RDB 配置数据模型
type RdbConfigData struct {
	ID        string    `rdb:"id,primary_key"`
	Content   string    `rdb:"content,not_null"`
	Version   int64     `rdb:"version,auto_increment"`
	CreatedAt time.Time `rdb:"created_at,default=CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `rdb:"updated_at,on_update=CURRENT_TIMESTAMP"`
}

// RdbProvider 基于 RDB 的配置提供者
type RdbProvider struct {
	configID    string
	repo        repository.Repository[RdbConfigData]
	mu          sync.RWMutex
	onChange    []func(data []byte) error
	lastVersion int64

	// 变更监听
	stopChan     chan struct{}
	pollInterval time.Duration
	watching     bool
	once         sync.Once
}

// RdbProviderOptions RDB Provider 配置选项
type RdbProviderOptions struct {
	ConfigID     string                 // 配置 ID
	Database     *ref.TypeOptions       // 数据库配置
	PollInterval time.Duration          // 轮询间隔，默认 5 秒
	Extra        map[string]any // 额外配置
}

// NewRdbProviderWithOptions 创建 RDB Provider
func NewRdbProviderWithOptions(options *RdbProviderOptions) (*RdbProvider, error) {
	if options == nil {
		return nil, errors.New("rdb provider options is required")
	}

	if options.ConfigID == "" {
		return nil, errors.New("config ID is required")
	}

	if options.Database == nil {
		return nil, errors.New("database config is required")
	}

	if options.PollInterval == 0 {
		options.PollInterval = 5 * time.Second
	}

	// 创建数据库连接
	db, err := rdb.NewDatabaseWithOptions(options.Database)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create database")
	}

	// 创建 Repository
	repo, err := rdb.NewRepository[RdbConfigData](db)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create repository")
	}

	provider := &RdbProvider{
		configID:     options.ConfigID,
		repo:         repo,
		pollInterval: options.PollInterval,
		stopChan:     make(chan struct{}),
	}

	// 自动迁移表结构
	ctx := context.Background()
	if err := repo.Migrate(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to migrate table")
	}

	return provider, nil
}

// Load 读取配置数据
func (p *RdbProvider) Load() ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	ctx := context.Background()
	config, err := p.repo.FindOne(ctx, &query.TermQuery{Field: "id", Value: p.configID})
	if err != nil {
		return nil, errors.Wrapf(err, "config not found: %s", p.configID)
	}

	p.lastVersion = config.Version
	return []byte(config.Content), nil
}

// Save 保存配置数据
func (p *RdbProvider) Save(data []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	ctx := context.Background()

	// 检查是否存在
	exists, err := p.repo.Exists(ctx, &query.TermQuery{Field: "id", Value: p.configID})
	if err != nil {
		return errors.Wrap(err, "failed to check existing config")
	}

	if !exists {
		// 不存在，执行插入
		config := &RdbConfigData{
			ID:      p.configID,
			Content: string(data),
		}
		err = p.repo.Create(ctx, config)
	} else {
		// 存在，执行更新
		config, err := p.repo.FindOne(ctx, &query.TermQuery{Field: "id", Value: p.configID})
		if err != nil {
			return errors.Wrap(err, "failed to find config for update")
		}

		config.Content = string(data)
		config.Version++ // 手动增加版本号
		err = p.repo.Update(ctx, config)
	}

	if err != nil {
		return errors.Wrap(err, "failed to save config")
	}

	return nil
}

// OnChange 注册配置变更回调函数
func (p *RdbProvider) OnChange(fn func(data []byte) error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.onChange = append(p.onChange, fn)
}

// Watch 启动配置变更监听
func (p *RdbProvider) Watch() error {
	p.once.Do(func() {
		p.mu.Lock()
		defer p.mu.Unlock()

		p.watching = true
		go p.startPolling()
	})

	return nil
}

// startPolling 启动轮询监听配置变更
func (p *RdbProvider) startPolling() {
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
func (p *RdbProvider) checkForChanges() {
	p.mu.RLock()
	handlers := make([]func(data []byte) error, len(p.onChange))
	copy(handlers, p.onChange)
	lastVersion := p.lastVersion
	p.mu.RUnlock()

	if len(handlers) == 0 {
		return
	}

	ctx := context.Background()
	config, err := p.repo.FindOne(ctx, &query.TermQuery{Field: "id", Value: p.configID})
	if err != nil {
		return // 忽略错误，继续轮询
	}

	if config.Version > lastVersion {
		data := []byte(config.Content)
		// 调用所有注册的回调函数
		for _, handler := range handlers {
			if handler != nil {
				if err := handler(data); err != nil {
					// 如果某个回调失败，记录但不影响其他回调
					continue
				}
			}
		}
		// 更新版本号
		p.mu.Lock()
		p.lastVersion = config.Version
		p.mu.Unlock()
	}
}

// Close 关闭提供者，释放资源
func (p *RdbProvider) Close() error {
	close(p.stopChan)
	return nil
}