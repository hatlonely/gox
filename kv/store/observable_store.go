package store

import (
	"context"
	"fmt"
	"time"

	"github.com/hatlonely/gox/log"
	"github.com/hatlonely/gox/log/logger"
	"github.com/hatlonely/gox/ref"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type ObservableStoreOptions struct {
	// Store 被包装的底层存储配置
	Store *ref.TypeOptions `cfg:"store" validate:"required"`

	// Logger 日志记录器配置
	Logger *ref.TypeOptions `cfg:"logger"`

	// EnableMetrics 是否启用指标收集
	EnableMetrics bool `cfg:"enableMetrics" def:"true"`

	// EnableLogging 是否启用日志记录
	EnableLogging bool `cfg:"enableLogging" def:"true"`

	// EnableTracing 是否启用分布式追踪
	EnableTracing bool `cfg:"enableTracing" def:"false"`

	// Name 组件名称标识，用于所有观测维度
	// - Metrics: 作为指标名前缀
	// - Logging: 作为 component 字段值
	// - Tracing: 作为 span 的 component 属性
	Name string `cfg:"name" def:"store"`
}

// ObservableMetrics 封装 prometheus 指标
type ObservableMetrics struct {
	operationCounter   *prometheus.CounterVec
	operationDuration  *prometheus.HistogramVec
	activeOperations   *prometheus.GaugeVec
	batchSizeHistogram *prometheus.HistogramVec
}

// NewObservableMetrics 创建指标收集器
func NewObservableMetrics(name string) *ObservableMetrics {
	metrics := &ObservableMetrics{
		operationCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: name + "_operations_total",
				Help: "Total number of store operations",
			},
			[]string{"operation", "status"},
		),
		operationDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    name + "_operation_duration_seconds",
				Help:    "Duration of store operations in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
			},
			[]string{"operation"},
		),
		activeOperations: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: name + "_active_operations",
				Help: "Number of active store operations",
			},
			[]string{"operation"},
		),
		batchSizeHistogram: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    name + "_batch_size",
				Help:    "Size of batch operations",
				Buckets: []float64{1, 5, 10, 50, 100, 500, 1000},
			},
			[]string{"operation"},
		),
	}

	// 注册到默认 prometheus registry
	prometheus.MustRegister(
		metrics.operationCounter,
		metrics.operationDuration,
		metrics.activeOperations,
		metrics.batchSizeHistogram,
	)

	return metrics
}

// ObservableStore 装饰器，为任何 Store 添加观测能力
type ObservableStore[K comparable, V any] struct {
	store Store[K, V]

	logger        logger.Logger
	metrics       *ObservableMetrics
	tracer        trace.Tracer
	name          string
	enableMetrics bool
	enableLogging bool
	enableTracing bool
}

func NewObservableStoreWithOptions[K comparable, V any](options *ObservableStoreOptions) (*ObservableStore[K, V], error) {
	if options == nil {
		return nil, errors.New("options is nil")
	}

	// 创建底层 store
	store, err := NewStoreWithOptions[K, V](options.Store)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create underlying store")
	}

	obs := &ObservableStore[K, V]{
		store:         store,
		name:          options.Name,
		enableMetrics: options.EnableMetrics,
		enableLogging: options.EnableLogging,
		enableTracing: options.EnableTracing,
	}

	// 创建 logger（可选）
	if options.EnableLogging && options.Logger != nil {
		l, err := log.NewLoggerWithOptions(options.Logger)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to create logger")
		}
		obs.logger = l.WithGroup("observableStore")
	}

	// 创建 metrics（可选）
	if options.EnableMetrics {
		obs.metrics = NewObservableMetrics(options.Name)
	}

	// 创建 tracer（可选）
	if options.EnableTracing {
		obs.tracer = otel.Tracer(fmt.Sprintf("store.%s", options.Name))
	}

	return obs, nil
}

// observeOperation 统一的操作观测逻辑
func (obs *ObservableStore[K, V]) observeOperation(ctx context.Context, operation string, fn func(context.Context) error) error {
	start := time.Now()
	
	// 创建 tracing span
	var span trace.Span
	if obs.enableTracing && obs.tracer != nil {
		ctx, span = obs.tracer.Start(ctx, fmt.Sprintf("store.%s", operation),
			trace.WithAttributes(
				attribute.String("component", obs.name),
				attribute.String("operation", operation),
			),
		)
		defer span.End()
	}

	// 记录活跃操作数
	if obs.enableMetrics && obs.metrics != nil {
		obs.metrics.activeOperations.WithLabelValues(operation).Inc()
		defer obs.metrics.activeOperations.WithLabelValues(operation).Dec()
	}

	// 执行实际操作
	err := fn(ctx)
	duration := time.Since(start)

	// 更新 tracing span
	if obs.enableTracing && span != nil {
		span.SetAttributes(
			attribute.Int64("duration_ms", duration.Milliseconds()),
		)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		} else {
			span.SetStatus(codes.Ok, "")
		}
	}

	// 记录指标
	if obs.enableMetrics && obs.metrics != nil {
		status := "success"
		if err != nil {
			status = "error"
		}
		obs.metrics.operationCounter.WithLabelValues(operation, status).Inc()
		obs.metrics.operationDuration.WithLabelValues(operation).Observe(duration.Seconds())
	}

	// 记录日志
	if obs.enableLogging && obs.logger != nil {
		if err != nil {
			obs.logger.ErrorContext(ctx, "store operation failed",
				"component", obs.name,
				"operation", operation,
				"duration_ms", duration.Milliseconds(),
				"error", err.Error(),
			)
		} else {
			obs.logger.InfoContext(ctx, "store operation completed",
				"component", obs.name,
				"operation", operation,
				"duration_ms", duration.Milliseconds(),
			)
		}
	}

	return err
}

// observeBatchOperation 批量操作的观测逻辑
func (obs *ObservableStore[K, V]) observeBatchOperation(ctx context.Context, operation string, batchSize int, fn func(context.Context) error) error {
	start := time.Now()
	
	// 创建 tracing span
	var span trace.Span
	if obs.enableTracing && obs.tracer != nil {
		ctx, span = obs.tracer.Start(ctx, fmt.Sprintf("store.%s", operation),
			trace.WithAttributes(
				attribute.String("component", obs.name),
				attribute.String("operation", operation),
				attribute.Int("batch_size", batchSize),
			),
		)
		defer span.End()
	}

	// 记录批量大小
	if obs.enableMetrics && obs.metrics != nil {
		obs.metrics.batchSizeHistogram.WithLabelValues(operation).Observe(float64(batchSize))
		obs.metrics.activeOperations.WithLabelValues(operation).Inc()
		defer obs.metrics.activeOperations.WithLabelValues(operation).Dec()
	}

	// 执行实际操作
	err := fn(ctx)
	duration := time.Since(start)

	// 更新 tracing span
	if obs.enableTracing && span != nil {
		span.SetAttributes(
			attribute.Int64("duration_ms", duration.Milliseconds()),
		)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		} else {
			span.SetStatus(codes.Ok, "")
		}
	}

	// 记录指标
	if obs.enableMetrics && obs.metrics != nil {
		status := "success"
		if err != nil {
			status = "error"
		}
		obs.metrics.operationCounter.WithLabelValues(operation, status).Inc()
		obs.metrics.operationDuration.WithLabelValues(operation).Observe(duration.Seconds())
	}

	// 记录日志
	if obs.enableLogging && obs.logger != nil {
		if err != nil {
			obs.logger.ErrorContext(ctx, "batch store operation failed",
				"component", obs.name,
				"operation", operation,
				"batch_size", batchSize,
				"duration_ms", duration.Milliseconds(),
				"error", err.Error(),
			)
		} else {
			obs.logger.InfoContext(ctx, "batch store operation completed",
				"component", obs.name,
				"operation", operation,
				"batch_size", batchSize,
				"duration_ms", duration.Milliseconds(),
			)
		}
	}

	return err
}

func (obs *ObservableStore[K, V]) Set(ctx context.Context, key K, value V, opts ...setOption) error {
	return obs.observeOperation(ctx, "set", func(ctx context.Context) error {
		return obs.store.Set(ctx, key, value, opts...)
	})
}

func (obs *ObservableStore[K, V]) Get(ctx context.Context, key K) (V, error) {
	var result V
	err := obs.observeOperation(ctx, "get", func(ctx context.Context) error {
		var getErr error
		result, getErr = obs.store.Get(ctx, key)
		return getErr
	})
	return result, err
}

func (obs *ObservableStore[K, V]) Del(ctx context.Context, key K) error {
	return obs.observeOperation(ctx, "del", func(ctx context.Context) error {
		return obs.store.Del(ctx, key)
	})
}

func (obs *ObservableStore[K, V]) BatchSet(ctx context.Context, keys []K, vals []V, opts ...setOption) ([]error, error) {
	var result []error
	err := obs.observeBatchOperation(ctx, "batch_set", len(keys), func(ctx context.Context) error {
		var batchErr error
		result, batchErr = obs.store.BatchSet(ctx, keys, vals, opts...)
		return batchErr
	})
	return result, err
}

func (obs *ObservableStore[K, V]) BatchGet(ctx context.Context, keys []K) ([]V, []error, error) {
	var vals []V
	var errs []error
	err := obs.observeBatchOperation(ctx, "batch_get", len(keys), func(ctx context.Context) error {
		var batchErr error
		vals, errs, batchErr = obs.store.BatchGet(ctx, keys)
		return batchErr
	})
	return vals, errs, err
}

func (obs *ObservableStore[K, V]) BatchDel(ctx context.Context, keys []K) ([]error, error) {
	var result []error
	err := obs.observeBatchOperation(ctx, "batch_del", len(keys), func(ctx context.Context) error {
		var batchErr error
		result, batchErr = obs.store.BatchDel(ctx, keys)
		return batchErr
	})
	return result, err
}

func (obs *ObservableStore[K, V]) Close() error {
	return obs.observeOperation(context.Background(), "close", func(ctx context.Context) error {
		return obs.store.Close()
	})
}