package database

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/hatlonely/gox/rdb/aggregation"
	"github.com/hatlonely/gox/rdb/query"
)

// ESOptions Elasticsearch连接选项
type ESOptions struct {
	Addresses []string      `cfg:"addresses" def:"[\"http://localhost:9200\"]"`
	Username  string        `cfg:"username"`
	Password  string        `cfg:"password"`
	APIKey    string        `cfg:"apiKey"`
	Timeout   time.Duration `cfg:"timeout" def:"30s"`
	MaxRetries int          `cfg:"maxRetries" def:"3"`
}

// ES Elasticsearch数据库实现
type ES struct {
	client  *elasticsearch.Client
	builder *ESRecordBuilder
}

// NewESWithOptions 创建Elasticsearch实例
func NewESWithOptions(opts *ESOptions) (*ES, error) {
	cfg := elasticsearch.Config{
		Addresses: opts.Addresses,
		Username:  opts.Username,
		Password:  opts.Password,
		APIKey:    opts.APIKey,
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: opts.Timeout,
		},
		MaxRetries: opts.MaxRetries,
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create elasticsearch client: %v", err)
	}

	// 测试连接
	res, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to elasticsearch: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch connection error: %s", res.String())
	}

	return &ES{
		client:  client,
		builder: &ESRecordBuilder{},
	}, nil
}

// ESRecord Elasticsearch记录实现
type ESRecord struct {
	data   map[string]any
	id     string
	index  string
	source map[string]any
}

func (r *ESRecord) Scan(dest any) error {
	return mapToStruct(r.source, dest)
}

func (r *ESRecord) ScanStruct(dest any) error {
	return r.Scan(dest)
}

func (r *ESRecord) Fields() map[string]any {
	if r.source != nil {
		return r.source
	}
	return r.data
}

// ESRecordBuilder Elasticsearch记录构建器
type ESRecordBuilder struct{}

func (b *ESRecordBuilder) FromStruct(v any) Record {
	data := structToMap(v)
	return &ESRecord{data: data, source: data}
}

func (b *ESRecordBuilder) FromMap(data map[string]any, table string) Record {
	return &ESRecord{data: data, source: data, index: table}
}

// 实现Database接口的基础方法
func (es *ES) GetBuilder() RecordBuilder {
	return es.builder
}

func (es *ES) Close() error {
	// Elasticsearch客户端不需要显式关闭
	return nil
}// M
igrate 创建/更新索引映射
func (es *ES) Migrate(ctx context.Context, model *TableModel) error {
	// 构建索引映射
	mapping := es.buildIndexMapping(model)
	
	// 检查索引是否存在
	req := esapi.IndicesExistsRequest{
		Index: []string{model.Table},
	}
	
	res, err := req.Do(ctx, es.client)
	if err != nil {
		return fmt.Errorf("failed to check index existence: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		// 索引不存在，创建新索引
		return es.createIndex(ctx, model.Table, mapping)
	} else if res.StatusCode == 200 {
		// 索引存在，更新映射
		return es.updateIndexMapping(ctx, model.Table, mapping)
	}
	
	return fmt.Errorf("unexpected response status: %d", res.StatusCode)
}

// buildIndexMapping 构建索引映射
func (es *ES) buildIndexMapping(model *TableModel) map[string]any {
	properties := make(map[string]any)
	
	for _, field := range model.Fields {
		properties[field.Name] = es.mapFieldTypeToES(field.Type, field.Size)
	}
	
	mapping := map[string]any{
		"mappings": map[string]any{
			"properties": properties,
		},
	}
	
	// 添加索引设置
	settings := map[string]any{
		"number_of_shards":   1,
		"number_of_replicas": 0,
	}
	mapping["settings"] = settings
	
	return mapping
}

// mapFieldTypeToES 将字段类型映射为ES类型
func (es *ES) mapFieldTypeToES(fieldType FieldType, size int) map[string]any {
	switch fieldType {
	case FieldTypeString:
		return map[string]any{
			"type": "text",
			"fields": map[string]any{
				"keyword": map[string]any{
					"type":         "keyword",
					"ignore_above": 256,
				},
			},
		}
	case FieldTypeInt:
		return map[string]any{"type": "long"}
	case FieldTypeFloat:
		return map[string]any{"type": "double"}
	case FieldTypeBool:
		return map[string]any{"type": "boolean"}
	case FieldTypeDate:
		return map[string]any{
			"type":   "date",
			"format": "yyyy-MM-dd HH:mm:ss||yyyy-MM-dd||epoch_millis",
		}
	case FieldTypeJSON:
		return map[string]any{"type": "object"}
	default:
		return map[string]any{"type": "text"}
	}
}

// createIndex 创建新索引
func (es *ES) createIndex(ctx context.Context, index string, mapping map[string]any) error {
	body, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("failed to marshal mapping: %v", err)
	}
	
	req := esapi.IndicesCreateRequest{
		Index: index,
		Body:  strings.NewReader(string(body)),
	}
	
	res, err := req.Do(ctx, es.client)
	if err != nil {
		return fmt.Errorf("failed to create index: %v", err)
	}
	defer res.Body.Close()
	
	if res.IsError() {
		return fmt.Errorf("failed to create index: %s", res.String())
	}
	
	return nil
}

// updateIndexMapping 更新索引映射
func (es *ES) updateIndexMapping(ctx context.Context, index string, mapping map[string]any) error {
	// ES只允许添加新字段，不能修改现有字段类型
	properties := mapping["mappings"].(map[string]any)["properties"]
	
	body, err := json.Marshal(map[string]any{
		"properties": properties,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal mapping: %v", err)
	}
	
	req := esapi.IndicesPutMappingRequest{
		Index: []string{index},
		Body:  strings.NewReader(string(body)),
	}
	
	res, err := req.Do(ctx, es.client)
	if err != nil {
		return fmt.Errorf("failed to update mapping: %v", err)
	}
	defer res.Body.Close()
	
	if res.IsError() {
		return fmt.Errorf("failed to update mapping: %s", res.String())
	}
	
	return nil
}

// DropTable 删除索引
func (es *ES) DropTable(ctx context.Context, table string) error {
	req := esapi.IndicesDeleteRequest{
		Index: []string{table},
	}
	
	res, err := req.Do(ctx, es.client)
	if err != nil {
		return fmt.Errorf("failed to delete index: %v", err)
	}
	defer res.Body.Close()
	
	if res.IsError() && res.StatusCode != 404 {
		return fmt.Errorf("failed to delete index: %s", res.String())
	}
	
	return nil
}// CRUD 
操作实现
func (es *ES) Create(ctx context.Context, table string, record Record, opts ...CreateOption) error {
	// 解析创建选项
	createOpts := &CreateOptions{}
	for _, opt := range opts {
		opt(createOpts)
	}

	fields := record.Fields()
	
	// 提取文档ID（如果存在）
	var docID string
	if id, exists := fields["_id"]; exists {
		docID = fmt.Sprintf("%v", id)
		delete(fields, "_id") // 从文档内容中移除_id
	}
	
	// 序列化文档
	body, err := json.Marshal(fields)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %v", err)
	}
	
	if createOpts.IgnoreConflict {
		// 使用create操作，如果文档已存在则忽略
		req := esapi.CreateRequest{
			Index:      table,
			DocumentID: docID,
			Body:       strings.NewReader(string(body)),
			Refresh:    "wait_for",
		}
		
		res, err := req.Do(ctx, es.client)
		if err != nil {
			return fmt.Errorf("failed to create document: %v", err)
		}
		defer res.Body.Close()
		
		if res.IsError() && res.StatusCode != 409 {
			return fmt.Errorf("failed to create document: %s", res.String())
		}
		
		return nil
	} else if createOpts.UpdateOnConflict {
		// 使用index操作，如果文档已存在则更新
		req := esapi.IndexRequest{
			Index:      table,
			DocumentID: docID,
			Body:       strings.NewReader(string(body)),
			Refresh:    "wait_for",
		}
		
		res, err := req.Do(ctx, es.client)
		if err != nil {
			return fmt.Errorf("failed to index document: %v", err)
		}
		defer res.Body.Close()
		
		if res.IsError() {
			return fmt.Errorf("failed to index document: %s", res.String())
		}
		
		return nil
	} else {
		// 默认的create操作
		req := esapi.CreateRequest{
			Index:      table,
			DocumentID: docID,
			Body:       strings.NewReader(string(body)),
			Refresh:    "wait_for",
		}
		
		res, err := req.Do(ctx, es.client)
		if err != nil {
			return fmt.Errorf("failed to create document: %v", err)
		}
		defer res.Body.Close()
		
		if res.IsError() {
			if res.StatusCode == 409 {
				return ErrDuplicateKey
			}
			return fmt.Errorf("failed to create document: %s", res.String())
		}
		
		return nil
	}
}

func (es *ES) Get(ctx context.Context, table string, pk map[string]any) (Record, error) {
	// ES中主键通常是_id字段
	var docID string
	if id, exists := pk["_id"]; exists {
		docID = fmt.Sprintf("%v", id)
	} else if id, exists := pk["id"]; exists {
		docID = fmt.Sprintf("%v", id)
	} else {
		return nil, fmt.Errorf("document ID not found in primary key")
	}
	
	req := esapi.GetRequest{
		Index:      table,
		DocumentID: docID,
	}
	
	res, err := req.Do(ctx, es.client)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %v", err)
	}
	defer res.Body.Close()
	
	if res.StatusCode == 404 {
		return nil, ErrRecordNotFound
	}
	
	if res.IsError() {
		return nil, fmt.Errorf("failed to get document: %s", res.String())
	}
	
	// 解析响应
	var result map[string]any
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	
	// 检查文档是否存在
	if found, ok := result["found"].(bool); !ok || !found {
		return nil, ErrRecordNotFound
	}
	
	// 提取文档源数据
	source, ok := result["_source"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid document source")
	}
	
	// 添加文档ID到源数据
	source["_id"] = result["_id"]
	
	return &ESRecord{
		id:     docID,
		index:  table,
		source: source,
	}, nil
}

func (es *ES) Update(ctx context.Context, table string, pk map[string]any, record Record) error {
	// 提取文档ID
	var docID string
	if id, exists := pk["_id"]; exists {
		docID = fmt.Sprintf("%v", id)
	} else if id, exists := pk["id"]; exists {
		docID = fmt.Sprintf("%v", id)
	} else {
		return fmt.Errorf("document ID not found in primary key")
	}
	
	fields := record.Fields()
	
	// 构建更新文档
	updateDoc := map[string]any{
		"doc": fields,
	}
	
	body, err := json.Marshal(updateDoc)
	if err != nil {
		return fmt.Errorf("failed to marshal update document: %v", err)
	}
	
	req := esapi.UpdateRequest{
		Index:      table,
		DocumentID: docID,
		Body:       strings.NewReader(string(body)),
		Refresh:    "wait_for",
	}
	
	res, err := req.Do(ctx, es.client)
	if err != nil {
		return fmt.Errorf("failed to update document: %v", err)
	}
	defer res.Body.Close()
	
	if res.StatusCode == 404 {
		return ErrRecordNotFound
	}
	
	if res.IsError() {
		return fmt.Errorf("failed to update document: %s", res.String())
	}
	
	return nil
}

func (es *ES) Delete(ctx context.Context, table string, pk map[string]any) error {
	// 提取文档ID
	var docID string
	if id, exists := pk["_id"]; exists {
		docID = fmt.Sprintf("%v", id)
	} else if id, exists := pk["id"]; exists {
		docID = fmt.Sprintf("%v", id)
	} else {
		return fmt.Errorf("document ID not found in primary key")
	}
	
	req := esapi.DeleteRequest{
		Index:      table,
		DocumentID: docID,
		Refresh:    "wait_for",
	}
	
	res, err := req.Do(ctx, es.client)
	if err != nil {
		return fmt.Errorf("failed to delete document: %v", err)
	}
	defer res.Body.Close()
	
	if res.StatusCode == 404 {
		return ErrRecordNotFound
	}
	
	if res.IsError() {
		return fmt.Errorf("failed to delete document: %s", res.String())
	}
	
	return nil
}//
 查询和聚合功能实现
func (es *ES) Find(ctx context.Context, table string, query query.Query, opts ...QueryOption) ([]Record, error) {
	// 解析查询选项
	queryOpts := &QueryOptions{}
	for _, opt := range opts {
		opt(queryOpts)
	}
	
	// 构建ES查询
	esQuery := query.ToES()
	
	// 构建搜索请求体
	searchBody := map[string]any{
		"query": esQuery,
	}
	
	// 添加分页
	if queryOpts.Limit > 0 {
		searchBody["size"] = queryOpts.Limit
	}
	if queryOpts.Offset > 0 {
		searchBody["from"] = queryOpts.Offset
	}
	
	// 添加排序
	if queryOpts.OrderBy != "" {
		order := "asc"
		if queryOpts.OrderDesc {
			order = "desc"
		}
		searchBody["sort"] = []map[string]any{
			{queryOpts.OrderBy: map[string]any{"order": order}},
		}
	}
	
	// 序列化请求体
	body, err := json.Marshal(searchBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search body: %v", err)
	}
	
	// 执行搜索
	req := esapi.SearchRequest{
		Index: []string{table},
		Body:  strings.NewReader(string(body)),
	}
	
	res, err := req.Do(ctx, es.client)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %v", err)
	}
	defer res.Body.Close()
	
	if res.IsError() {
		return nil, fmt.Errorf("search error: %s", res.String())
	}
	
	// 解析搜索结果
	var searchResult map[string]any
	if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
		return nil, fmt.Errorf("failed to decode search result: %v", err)
	}
	
	// 提取文档
	hits, ok := searchResult["hits"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid search result format")
	}
	
	hitsList, ok := hits["hits"].([]any)
	if !ok {
		return nil, fmt.Errorf("invalid hits format")
	}
	
	var records []Record
	for _, hit := range hitsList {
		hitMap, ok := hit.(map[string]any)
		if !ok {
			continue
		}
		
		source, ok := hitMap["_source"].(map[string]any)
		if !ok {
			continue
		}
		
		// 添加文档元数据
		source["_id"] = hitMap["_id"]
		source["_index"] = hitMap["_index"]
		
		records = append(records, &ESRecord{
			id:     fmt.Sprintf("%v", hitMap["_id"]),
			index:  table,
			source: source,
		})
	}
	
	return records, nil
}

func (es *ES) Aggregate(ctx context.Context, table string, query query.Query, aggs []aggregation.Aggregation, opts ...QueryOption) (aggregation.AggregationResult, error) {
	// 解析查询选项
	queryOpts := &QueryOptions{}
	for _, opt := range opts {
		opt(queryOpts)
	}
	
	// 构建ES查询
	esQuery := query.ToES()
	
	// 构建聚合
	esAggs := make(map[string]any)
	for _, agg := range aggs {
		aggName := agg.Name()
		if aggName == "" {
			aggName = fmt.Sprintf("%s_agg", agg.Type())
		}
		esAggs[aggName] = agg.ToES()
	}
	
	// 构建搜索请求体
	searchBody := map[string]any{
		"query": esQuery,
		"aggs":  esAggs,
		"size":  0, // 只返回聚合结果，不返回文档
	}
	
	// 序列化请求体
	body, err := json.Marshal(searchBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search body: %v", err)
	}
	
	// 执行搜索
	req := esapi.SearchRequest{
		Index: []string{table},
		Body:  strings.NewReader(string(body)),
	}
	
	res, err := req.Do(ctx, es.client)
	if err != nil {
		return nil, fmt.Errorf("failed to execute aggregation: %v", err)
	}
	defer res.Body.Close()
	
	if res.IsError() {
		return nil, fmt.Errorf("aggregation error: %s", res.String())
	}
	
	// 解析聚合结果
	var searchResult map[string]any
	if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
		return nil, fmt.Errorf("failed to decode aggregation result: %v", err)
	}
	
	// 提取聚合结果
	aggregations, ok := searchResult["aggregations"].(map[string]any)
	if !ok {
		return aggregation.NewAggregationResult(), nil
	}
	
	// 构建聚合结果
	result := aggregation.NewAggregationResult()
	for _, agg := range aggs {
		aggName := agg.Name()
		if aggName == "" {
			aggName = fmt.Sprintf("%s_agg", agg.Type())
		}
		
		if aggResult, exists := aggregations[aggName]; exists {
			// 根据聚合类型解析结果
			switch agg.Type() {
			case aggregation.AggTypeSum, aggregation.AggTypeAvg, 
				 aggregation.AggTypeMax, aggregation.AggTypeMin:
				if aggMap, ok := aggResult.(map[string]any); ok {
					if value, exists := aggMap["value"]; exists {
						result.SetResult(aggName, value)
					}
				}
			case aggregation.AggTypeCount:
				if aggMap, ok := aggResult.(map[string]any); ok {
					if docCount, exists := aggMap["doc_count"]; exists {
						result.SetResult(aggName, docCount)
					}
				}
			case aggregation.AggTypeTerms:
				if aggMap, ok := aggResult.(map[string]any); ok {
					if buckets, exists := aggMap["buckets"]; exists {
						result.SetResult(aggName, buckets)
					}
				}
			case aggregation.AggTypeDateHisto:
				if aggMap, ok := aggResult.(map[string]any); ok {
					if buckets, exists := aggMap["buckets"]; exists {
						result.SetResult(aggName, buckets)
					}
				}
			}
		}
	}
	
	return result, nil
}// 批
量操作实现
func (es *ES) BatchCreate(ctx context.Context, table string, records []Record, opts ...CreateOption) error {
	if len(records) == 0 {
		return nil
	}
	
	// 解析创建选项
	createOpts := &CreateOptions{}
	for _, opt := range opts {
		opt(createOpts)
	}
	
	// 构建批量请求体
	var bulkBody strings.Builder
	
	for _, record := range records {
		fields := record.Fields()
		
		// 提取文档ID（如果存在）
		var docID string
		if id, exists := fields["_id"]; exists {
			docID = fmt.Sprintf("%v", id)
			delete(fields, "_id")
		}
		
		// 构建操作头
		var action string
		if createOpts.UpdateOnConflict {
			action = "index"
		} else {
			action = "create"
		}
		
		actionHeader := map[string]any{
			action: map[string]any{
				"_index": table,
			},
		}
		
		if docID != "" {
			actionHeader[action].(map[string]any)["_id"] = docID
		}
		
		// 写入操作头
		headerBytes, err := json.Marshal(actionHeader)
		if err != nil {
			return fmt.Errorf("failed to marshal action header: %v", err)
		}
		bulkBody.Write(headerBytes)
		bulkBody.WriteString("\n")
		
		// 写入文档内容
		docBytes, err := json.Marshal(fields)
		if err != nil {
			return fmt.Errorf("failed to marshal document: %v", err)
		}
		bulkBody.Write(docBytes)
		bulkBody.WriteString("\n")
	}
	
	// 执行批量操作
	req := esapi.BulkRequest{
		Body:    strings.NewReader(bulkBody.String()),
		Refresh: "wait_for",
	}
	
	res, err := req.Do(ctx, es.client)
	if err != nil {
		return fmt.Errorf("failed to execute bulk create: %v", err)
	}
	defer res.Body.Close()
	
	if res.IsError() {
		return fmt.Errorf("bulk create error: %s", res.String())
	}
	
	// 解析批量响应
	var bulkResult map[string]any
	if err := json.NewDecoder(res.Body).Decode(&bulkResult); err != nil {
		return fmt.Errorf("failed to decode bulk result: %v", err)
	}
	
	// 检查是否有错误
	if errors, ok := bulkResult["errors"].(bool); ok && errors {
		if !createOpts.IgnoreConflict {
			return fmt.Errorf("bulk operation contains errors")
		}
	}
	
	return nil
}

func (es *ES) BatchUpdate(ctx context.Context, table string, pks []map[string]any, records []Record) error {
	if len(pks) != len(records) {
		return fmt.Errorf("pks and records length mismatch")
	}
	
	if len(records) == 0 {
		return nil
	}
	
	// 构建批量更新请求体
	var bulkBody strings.Builder
	
	for i, record := range records {
		// 提取文档ID
		var docID string
		if id, exists := pks[i]["_id"]; exists {
			docID = fmt.Sprintf("%v", id)
		} else if id, exists := pks[i]["id"]; exists {
			docID = fmt.Sprintf("%v", id)
		} else {
			return fmt.Errorf("document ID not found in primary key at index %d", i)
		}
		
		// 构建更新操作头
		actionHeader := map[string]any{
			"update": map[string]any{
				"_index": table,
				"_id":    docID,
			},
		}
		
		// 写入操作头
		headerBytes, err := json.Marshal(actionHeader)
		if err != nil {
			return fmt.Errorf("failed to marshal action header: %v", err)
		}
		bulkBody.Write(headerBytes)
		bulkBody.WriteString("\n")
		
		// 构建更新文档
		updateDoc := map[string]any{
			"doc": record.Fields(),
		}
		
		// 写入更新内容
		docBytes, err := json.Marshal(updateDoc)
		if err != nil {
			return fmt.Errorf("failed to marshal update document: %v", err)
		}
		bulkBody.Write(docBytes)
		bulkBody.WriteString("\n")
	}
	
	// 执行批量更新
	req := esapi.BulkRequest{
		Body:    strings.NewReader(bulkBody.String()),
		Refresh: "wait_for",
	}
	
	res, err := req.Do(ctx, es.client)
	if err != nil {
		return fmt.Errorf("failed to execute bulk update: %v", err)
	}
	defer res.Body.Close()
	
	if res.IsError() {
		return fmt.Errorf("bulk update error: %s", res.String())
	}
	
	return nil
}

func (es *ES) BatchDelete(ctx context.Context, table string, pks []map[string]any) error {
	if len(pks) == 0 {
		return nil
	}
	
	// 构建批量删除请求体
	var bulkBody strings.Builder
	
	for _, pk := range pks {
		// 提取文档ID
		var docID string
		if id, exists := pk["_id"]; exists {
			docID = fmt.Sprintf("%v", id)
		} else if id, exists := pk["id"]; exists {
			docID = fmt.Sprintf("%v", id)
		} else {
			return fmt.Errorf("document ID not found in primary key")
		}
		
		// 构建删除操作头
		actionHeader := map[string]any{
			"delete": map[string]any{
				"_index": table,
				"_id":    docID,
			},
		}
		
		// 写入操作头
		headerBytes, err := json.Marshal(actionHeader)
		if err != nil {
			return fmt.Errorf("failed to marshal action header: %v", err)
		}
		bulkBody.Write(headerBytes)
		bulkBody.WriteString("\n")
	}
	
	// 执行批量删除
	req := esapi.BulkRequest{
		Body:    strings.NewReader(bulkBody.String()),
		Refresh: "wait_for",
	}
	
	res, err := req.Do(ctx, es.client)
	if err != nil {
		return fmt.Errorf("failed to execute bulk delete: %v", err)
	}
	defer res.Body.Close()
	
	if res.IsError() {
		return fmt.Errorf("bulk delete error: %s", res.String())
	}
	
	return nil
}// 
事务支持实现（ES不支持传统事务，使用文档版本控制模拟）
func (es *ES) BeginTx(ctx context.Context) (Transaction, error) {
	// Elasticsearch不支持传统的ACID事务
	// 这里返回一个模拟的事务实现，主要用于批量操作的一致性
	return &ESTransaction{
		es:         es,
		operations: make([]ESOperation, 0),
	}, nil
}

func (es *ES) WithTx(ctx context.Context, fn func(tx Transaction) error) error {
	tx, err := es.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// ESOperation 表示一个ES操作
type ESOperation struct {
	Type   string
	Table  string
	DocID  string
	Data   map[string]any
	PK     map[string]any
}

// ESTransaction ES事务实现（模拟）
type ESTransaction struct {
	es         *ES
	operations []ESOperation
	committed  bool
	rolledBack bool
}

func (tx *ESTransaction) Commit() error {
	if tx.rolledBack {
		return fmt.Errorf("transaction has been rolled back")
	}
	if tx.committed {
		return fmt.Errorf("transaction has already been committed")
	}

	// 执行所有操作（使用批量API提高性能）
	if len(tx.operations) > 0 {
		err := tx.executeBulkOperations(context.Background())
		if err != nil {
			return err
		}
	}

	tx.committed = true
	return nil
}

func (tx *ESTransaction) Rollback() error {
	if tx.committed {
		return fmt.Errorf("transaction has already been committed")
	}
	if tx.rolledBack {
		return fmt.Errorf("transaction has already been rolled back")
	}

	// ES不支持真正的回滚，只能清空操作队列
	tx.operations = nil
	tx.rolledBack = true
	return nil
}

// executeBulkOperations 执行批量操作
func (tx *ESTransaction) executeBulkOperations(ctx context.Context) error {
	if len(tx.operations) == 0 {
		return nil
	}

	// 构建批量请求体
	var bulkBody strings.Builder

	for _, op := range tx.operations {
		switch op.Type {
		case "create":
			actionHeader := map[string]any{
				"create": map[string]any{
					"_index": op.Table,
				},
			}
			if op.DocID != "" {
				actionHeader["create"].(map[string]any)["_id"] = op.DocID
			}

			headerBytes, _ := json.Marshal(actionHeader)
			bulkBody.Write(headerBytes)
			bulkBody.WriteString("\n")

			docBytes, _ := json.Marshal(op.Data)
			bulkBody.Write(docBytes)
			bulkBody.WriteString("\n")

		case "update":
			actionHeader := map[string]any{
				"update": map[string]any{
					"_index": op.Table,
					"_id":    op.DocID,
				},
			}

			headerBytes, _ := json.Marshal(actionHeader)
			bulkBody.Write(headerBytes)
			bulkBody.WriteString("\n")

			updateDoc := map[string]any{"doc": op.Data}
			docBytes, _ := json.Marshal(updateDoc)
			bulkBody.Write(docBytes)
			bulkBody.WriteString("\n")

		case "delete":
			actionHeader := map[string]any{
				"delete": map[string]any{
					"_index": op.Table,
					"_id":    op.DocID,
				},
			}

			headerBytes, _ := json.Marshal(actionHeader)
			bulkBody.Write(headerBytes)
			bulkBody.WriteString("\n")
		}
	}

	// 执行批量操作
	req := esapi.BulkRequest{
		Body:    strings.NewReader(bulkBody.String()),
		Refresh: "wait_for",
	}

	res, err := req.Do(ctx, tx.es.client)
	if err != nil {
		return fmt.Errorf("failed to execute bulk operations: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("bulk operations error: %s", res.String())
	}

	return nil
}

// 事务中的CRUD操作实现
func (tx *ESTransaction) Create(ctx context.Context, table string, record Record, opts ...CreateOption) error {
	if tx.committed || tx.rolledBack {
		return fmt.Errorf("transaction is not active")
	}

	fields := record.Fields()
	
	// 提取文档ID
	var docID string
	if id, exists := fields["_id"]; exists {
		docID = fmt.Sprintf("%v", id)
		delete(fields, "_id")
	}

	// 添加到操作队列
	tx.operations = append(tx.operations, ESOperation{
		Type:  "create",
		Table: table,
		DocID: docID,
		Data:  fields,
	})

	return nil
}

func (tx *ESTransaction) Get(ctx context.Context, table string, pk map[string]any) (Record, error) {
	if tx.committed || tx.rolledBack {
		return nil, fmt.Errorf("transaction is not active")
	}

	// 在事务中，直接调用ES的Get方法
	return tx.es.Get(ctx, table, pk)
}

func (tx *ESTransaction) Update(ctx context.Context, table string, pk map[string]any, record Record) error {
	if tx.committed || tx.rolledBack {
		return fmt.Errorf("transaction is not active")
	}

	// 提取文档ID
	var docID string
	if id, exists := pk["_id"]; exists {
		docID = fmt.Sprintf("%v", id)
	} else if id, exists := pk["id"]; exists {
		docID = fmt.Sprintf("%v", id)
	} else {
		return fmt.Errorf("document ID not found in primary key")
	}

	// 添加到操作队列
	tx.operations = append(tx.operations, ESOperation{
		Type:  "update",
		Table: table,
		DocID: docID,
		Data:  record.Fields(),
		PK:    pk,
	})

	return nil
}

func (tx *ESTransaction) Delete(ctx context.Context, table string, pk map[string]any) error {
	if tx.committed || tx.rolledBack {
		return fmt.Errorf("transaction is not active")
	}

	// 提取文档ID
	var docID string
	if id, exists := pk["_id"]; exists {
		docID = fmt.Sprintf("%v", id)
	} else if id, exists := pk["id"]; exists {
		docID = fmt.Sprintf("%v", id)
	} else {
		return fmt.Errorf("document ID not found in primary key")
	}

	// 添加到操作队列
	tx.operations = append(tx.operations, ESOperation{
		Type:  "delete",
		Table: table,
		DocID: docID,
		PK:    pk,
	})

	return nil
}

// 事务中的其他方法实现
func (tx *ESTransaction) Find(ctx context.Context, table string, query query.Query, opts ...QueryOption) ([]Record, error) {
	if tx.committed || tx.rolledBack {
		return nil, fmt.Errorf("transaction is not active")
	}
	return tx.es.Find(ctx, table, query, opts...)
}

func (tx *ESTransaction) Aggregate(ctx context.Context, table string, query query.Query, aggs []aggregation.Aggregation, opts ...QueryOption) (aggregation.AggregationResult, error) {
	if tx.committed || tx.rolledBack {
		return nil, fmt.Errorf("transaction is not active")
	}
	return tx.es.Aggregate(ctx, table, query, aggs, opts...)
}

func (tx *ESTransaction) BatchCreate(ctx context.Context, table string, records []Record, opts ...CreateOption) error {
	if tx.committed || tx.rolledBack {
		return fmt.Errorf("transaction is not active")
	}
	
	for _, record := range records {
		if err := tx.Create(ctx, table, record, opts...); err != nil {
			return err
		}
	}
	return nil
}

func (tx *ESTransaction) BatchUpdate(ctx context.Context, table string, pks []map[string]any, records []Record) error {
	if tx.committed || tx.rolledBack {
		return fmt.Errorf("transaction is not active")
	}
	
	if len(pks) != len(records) {
		return fmt.Errorf("pks and records length mismatch")
	}

	for i, record := range records {
		if err := tx.Update(ctx, table, pks[i], record); err != nil {
			return err
		}
	}
	return nil
}

func (tx *ESTransaction) BatchDelete(ctx context.Context, table string, pks []map[string]any) error {
	if tx.committed || tx.rolledBack {
		return fmt.Errorf("transaction is not active")
	}
	
	for _, pk := range pks {
		if err := tx.Delete(ctx, table, pk); err != nil {
			return err
		}
	}
	return nil
}

func (tx *ESTransaction) BeginTx(ctx context.Context) (Transaction, error) {
	return nil, fmt.Errorf("nested transactions not supported")
}

func (tx *ESTransaction) WithTx(ctx context.Context, fn func(tx Transaction) error) error {
	return fn(tx)
}

func (tx *ESTransaction) Migrate(ctx context.Context, model *TableModel) error {
	return fmt.Errorf("schema migration not supported in transactions")
}

func (tx *ESTransaction) DropTable(ctx context.Context, table string) error {
	return fmt.Errorf("drop table not supported in transactions")
}

func (tx *ESTransaction) GetBuilder() RecordBuilder {
	return tx.es.builder
}

func (tx *ESTransaction) Close() error {
	return nil
}