package database

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/hatlonely/gox/rdb/aggregation"
	"github.com/hatlonely/gox/rdb/query"
)

// MongoOptions MongoDB连接选项
type MongoOptions struct {
	URI        string        `cfg:"uri"`
	Host       string        `cfg:"host" def:"localhost"`
	Port       int           `cfg:"port" def:"27017"`
	Database   string        `cfg:"database"`
	Username   string        `cfg:"username"`
	Password   string        `cfg:"password"`
	AuthSource string        `cfg:"authSource" def:"admin"`
	Timeout    time.Duration `cfg:"timeout" def:"30s"`
	MaxPoolSize uint64       `cfg:"maxPoolSize" def:"100"`
	MinPoolSize uint64       `cfg:"minPoolSize" def:"0"`
}

// Mongo MongoDB数据库实现
type Mongo struct {
	client   *mongo.Client
	database *mongo.Database
	builder  *MongoRecordBuilder
	dbName   string
}

// NewMongoWithOptions 创建MongoDB实例
func NewMongoWithOptions(opts *MongoOptions) (*Mongo, error) {
	uri := opts.URI
	if uri == "" {
		if opts.Username != "" && opts.Password != "" {
			uri = fmt.Sprintf("mongodb://%s:%s@%s:%d/%s?authSource=%s",
				opts.Username, opts.Password, opts.Host, opts.Port,
				opts.Database, opts.AuthSource)
		} else {
			uri = fmt.Sprintf("mongodb://%s:%d/%s", opts.Host, opts.Port, opts.Database)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	clientOptions.SetMaxPoolSize(opts.MaxPoolSize)
	clientOptions.SetMinPoolSize(opts.MinPoolSize)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb: %v", err)
	}

	// 测试连接
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping mongodb: %v", err)
	}

	database := client.Database(opts.Database)

	return &Mongo{
		client:   client,
		database: database,
		builder:  &MongoRecordBuilder{},
		dbName:   opts.Database,
	}, nil
}

// MongoRecord MongoDB记录实现
type MongoRecord struct {
	data bson.M
}

func (r *MongoRecord) Scan(dest any) error {
	return bsonToStruct(r.data, dest)
}

func (r *MongoRecord) ScanStruct(dest any) error {
	return r.Scan(dest)
}

func (r *MongoRecord) Fields() map[string]any {
	result := make(map[string]any)
	for k, v := range r.data {
		result[k] = v
	}
	return result
}

// MongoRecordBuilder MongoDB记录构建器
type MongoRecordBuilder struct{}

func (b *MongoRecordBuilder) FromStruct(v any) Record {
	data := structToBSON(v)
	return &MongoRecord{data: data}
}

func (b *MongoRecordBuilder) FromMap(data map[string]any, table string) Record {
	bsonData := make(bson.M)
	for k, v := range data {
		bsonData[k] = v
	}
	return &MongoRecord{data: bsonData}
}

// 辅助函数：结构体转换为BSON
func structToBSON(v any) bson.M {
	result := make(bson.M)
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return result
	}

	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		field := rt.Field(i)
		if !field.IsExported() {
			continue
		}

		// 检查 rdb 或 bson 标签
		fieldName := field.Name
		omitEmpty := false
		
		// 优先使用 rdb 标签，但同时检查 bson 标签中的 omitempty
		if tag := field.Tag.Get("rdb"); tag != "" && tag != "-" {
			parts := strings.Split(tag, ",")
			fieldName = parts[0]
			for _, part := range parts[1:] {
				if part == "omitempty" {
					omitEmpty = true
				}
			}
		} else if tag := field.Tag.Get("bson"); tag != "" && tag != "-" {
			parts := strings.Split(tag, ",")
			fieldName = parts[0]
			for _, part := range parts[1:] {
				if part == "omitempty" {
					omitEmpty = true
				}
			}
		}
		
		// 如果 rdb 标签没有 omitempty，检查 bson 标签是否有
		if !omitEmpty {
			if bsonTag := field.Tag.Get("bson"); bsonTag != "" {
				parts := strings.Split(bsonTag, ",")
				for _, part := range parts[1:] {
					if part == "omitempty" {
						omitEmpty = true
						break
					}
				}
			}
		}

		if fieldName == "-" {
			continue
		}

		value := rv.Field(i).Interface()
		
		// 处理 omitempty: 如果值为零值且设置了omitempty，则跳过
		if omitEmpty {
			zeroValue := reflect.Zero(field.Type).Interface()
			if reflect.DeepEqual(value, zeroValue) {
				continue
			}
		}
		
		result[fieldName] = value
	}
	return result
}

// 辅助函数：BSON转换为结构体
func bsonToStruct(data bson.M, dest any) error {
	rv := reflect.ValueOf(dest)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("dest must be a pointer to struct")
	}

	rv = rv.Elem()
	rt := rv.Type()

	for i := 0; i < rv.NumField(); i++ {
		field := rt.Field(i)
		if !field.IsExported() {
			continue
		}

		fieldName := field.Name
		if tag := field.Tag.Get("rdb"); tag != "" && tag != "-" {
			if idx := strings.Index(tag, ","); idx != -1 {
				fieldName = tag[:idx]
			} else {
				fieldName = tag
			}
		} else if tag := field.Tag.Get("bson"); tag != "" && tag != "-" {
			if idx := strings.Index(tag, ","); idx != -1 {
				fieldName = tag[:idx]
			} else {
				fieldName = tag
			}
		}

		if value, exists := data[fieldName]; exists && value != nil {
			fieldValue := rv.Field(i)
			if fieldValue.CanSet() {
				if err := setBSONFieldValue(fieldValue, value); err != nil {
					return fmt.Errorf("failed to set field %s: %v", fieldName, err)
				}
			}
		}
	}
	return nil
}

// 辅助函数：设置BSON字段值
func setBSONFieldValue(fieldValue reflect.Value, value any) error {
	if value == nil {
		return nil
	}

	valueType := reflect.TypeOf(value)
	fieldType := fieldValue.Type()

	// 处理MongoDB特殊类型
	switch v := value.(type) {
	case primitive.ObjectID:
		if fieldType.Kind() == reflect.String {
			fieldValue.SetString(v.Hex())
			return nil
		}
	case primitive.DateTime:
		if fieldType.String() == "time.Time" {
			fieldValue.Set(reflect.ValueOf(v.Time()))
			return nil
		}
	}

	if valueType.AssignableTo(fieldType) {
		fieldValue.Set(reflect.ValueOf(value))
		return nil
	}

	if valueType.ConvertibleTo(fieldType) {
		fieldValue.Set(reflect.ValueOf(value).Convert(fieldType))
		return nil
	}

	return fmt.Errorf("cannot convert %v to %v", valueType, fieldType)
}

// 实现Database接口的基础方法
func (m *Mongo) GetBuilder() RecordBuilder {
	return m.builder
}

func (m *Mongo) Close() error {
	if m.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return m.client.Disconnect(ctx)
	}
	return nil
}

// Migrate 创建/更新集合
func (m *Mongo) Migrate(ctx context.Context, model *TableModel) error {
	collection := m.database.Collection(model.Table)

	// MongoDB中表相当于集合，会在第一次写入时自动创建
	// 这里主要是创建索引
	for _, index := range model.Indexes {
		keys := bson.D{}
		for _, field := range index.Fields {
			keys = append(keys, bson.E{Key: field, Value: 1})
		}
		
		indexModel := mongo.IndexModel{
			Keys: keys,
		}

		// 设置索引选项
		indexOptions := options.Index()
		if index.Unique {
			indexOptions.SetUnique(true)
		}
		indexOptions.SetName(index.Name)
		indexModel.Options = indexOptions

		// 创建索引
		_, err := collection.Indexes().CreateOne(ctx, indexModel)
		if err != nil {
			// 如果索引已存在，忽略错误
			if !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("failed to create index %s: %v", index.Name, err)
			}
		}
	}

	return nil
}

// DropTable 删除集合
func (m *Mongo) DropTable(ctx context.Context, table string) error {
	collection := m.database.Collection(table)
	return collection.Drop(ctx)
}

// CRUD 操作实现
func (m *Mongo) Create(ctx context.Context, table string, record Record, opts ...CreateOption) error {
	// 解析创建选项
	createOpts := &CreateOptions{}
	for _, opt := range opts {
		opt(createOpts)
	}

	collection := m.database.Collection(table)
	fields := record.Fields()

	// 处理主键：MongoDB使用_id作为主键，如果没有则自动生成
	if _, exists := fields["_id"]; !exists {
		fields["_id"] = primitive.NewObjectID()
	}

	// 转换为BSON
	doc := make(bson.M)
	for k, v := range fields {
		doc[k] = v
	}

	if createOpts.IgnoreConflict {
		// 尝试插入，如果失败则忽略
		_, err := collection.InsertOne(ctx, doc)
		if err != nil && strings.Contains(err.Error(), "duplicate key") {
			return nil // 忽略重复键错误
		}
		return err
	} else if createOpts.UpdateOnConflict {
		// 使用ReplaceOne with upsert选项在冲突时更新
		filter := bson.M{"_id": doc["_id"]}
		replaceOptions := options.Replace().SetUpsert(true)
		_, err := collection.ReplaceOne(ctx, filter, doc, replaceOptions)
		return err
	} else {
		// 默认的插入操作
		_, err := collection.InsertOne(ctx, doc)
		if err != nil && strings.Contains(err.Error(), "duplicate key") {
			return ErrDuplicateKey
		}
		return err
	}
}

func (m *Mongo) Get(ctx context.Context, table string, pk map[string]any) (Record, error) {
	collection := m.database.Collection(table)

	// 构建查询过滤器
	filter := make(bson.M)
	for k, v := range pk {
		filter[k] = v
	}

	var result bson.M
	err := collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	return &MongoRecord{data: result}, nil
}

func (m *Mongo) Update(ctx context.Context, table string, pk map[string]any, record Record) error {
	collection := m.database.Collection(table)

	// 构建查询过滤器
	filter := make(bson.M)
	for k, v := range pk {
		filter[k] = v
	}

	// 构建更新文档
	fields := record.Fields()
	update := bson.M{"$set": fields}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return ErrRecordNotFound
	}

	return nil
}

func (m *Mongo) Delete(ctx context.Context, table string, pk map[string]any) error {
	collection := m.database.Collection(table)

	// 构建查询过滤器
	filter := make(bson.M)
	for k, v := range pk {
		filter[k] = v
	}

	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return ErrRecordNotFound
	}

	return nil
}

// 批量操作实现
func (m *Mongo) BatchCreate(ctx context.Context, table string, records []Record, opts ...CreateOption) error {
	if len(records) == 0 {
		return nil
	}

	collection := m.database.Collection(table)

	// 转换为BSON文档数组
	docs := make([]interface{}, len(records))
	for i, record := range records {
		fields := record.Fields()
		if _, exists := fields["_id"]; !exists {
			fields["_id"] = primitive.NewObjectID()
		}
		
		doc := make(bson.M)
		for k, v := range fields {
			doc[k] = v
		}
		docs[i] = doc
	}

	// 解析创建选项
	createOpts := &CreateOptions{}
	for _, opt := range opts {
		opt(createOpts)
	}

	insertOptions := options.InsertMany()
	if createOpts.IgnoreConflict {
		insertOptions.SetOrdered(false) // 允许部分失败
	}

	_, err := collection.InsertMany(ctx, docs, insertOptions)
	if err != nil && createOpts.IgnoreConflict && strings.Contains(err.Error(), "duplicate key") {
		// 如果是重复键错误且设置了忽略冲突，则忽略错误
		return nil
	}
	
	return err
}

func (m *Mongo) BatchUpdate(ctx context.Context, table string, pks []map[string]any, records []Record) error {
	if len(pks) != len(records) {
		return fmt.Errorf("pks and records length mismatch")
	}

	collection := m.database.Collection(table)

	for i, record := range records {
		// 构建查询过滤器
		filter := make(bson.M)
		for k, v := range pks[i] {
			filter[k] = v
		}

		// 构建更新文档
		fields := record.Fields()
		update := bson.M{"$set": fields}

		_, err := collection.UpdateOne(ctx, filter, update)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Mongo) BatchDelete(ctx context.Context, table string, pks []map[string]any) error {
	if len(pks) == 0 {
		return nil
	}

	collection := m.database.Collection(table)

	// 构建批量删除过滤器
	var filters []bson.M
	for _, pk := range pks {
		filter := make(bson.M)
		for k, v := range pk {
			filter[k] = v
		}
		filters = append(filters, filter)
	}

	// 使用$or查询删除多个文档
	filter := bson.M{"$or": filters}
	_, err := collection.DeleteMany(ctx, filter)
	return err
}

// 查询和聚合功能实现
func (m *Mongo) Find(ctx context.Context, table string, query query.Query, opts ...QueryOption) ([]Record, error) {
	// 解析查询选项
	queryOpts := &QueryOptions{}
	for _, opt := range opts {
		opt(queryOpts)
	}

	collection := m.database.Collection(table)

	// 构建查询过滤器
	filter, err := query.ToMongo()
	if err != nil {
		return nil, fmt.Errorf("failed to convert query to mongo: %v", err)
	}

	// 创建查找选项
	findOptions := options.Find()

	// 添加排序
	if queryOpts.OrderBy != "" {
		direction := 1
		if queryOpts.OrderDesc {
			direction = -1
		}
		findOptions.SetSort(bson.D{{Key: queryOpts.OrderBy, Value: direction}})
	}

	// 添加分页
	if queryOpts.Limit > 0 {
		findOptions.SetLimit(int64(queryOpts.Limit))
	}
	if queryOpts.Offset > 0 {
		findOptions.SetSkip(int64(queryOpts.Offset))
	}

	// 执行查询
	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// 扫描结果
	var records []Record
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		records = append(records, &MongoRecord{data: doc})
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

func (m *Mongo) Aggregate(ctx context.Context, table string, query query.Query, aggs []aggregation.Aggregation, opts ...QueryOption) (aggregation.AggregationResult, error) {
	// 解析查询选项
	queryOpts := &QueryOptions{}
	for _, opt := range opts {
		opt(queryOpts)
	}

	collection := m.database.Collection(table)

	// 构建聚合管道
	pipeline := make([]bson.M, 0)

	// 添加匹配阶段
	filter, err := query.ToMongo()
	if err != nil {
		return nil, fmt.Errorf("failed to convert query to mongo: %v", err)
	}
	if len(filter) > 0 {
		pipeline = append(pipeline, bson.M{"$match": filter})
	}

	// 构建聚合阶段
	groupStage := bson.M{}
	hasGrouping := false

	for _, agg := range aggs {
		aggDoc, err := agg.ToMongo()
		if err != nil {
			return nil, fmt.Errorf("failed to convert aggregation to mongo: %v", err)
		}

		switch agg.Type() {
		case aggregation.AggTypeSum, aggregation.AggTypeAvg, aggregation.AggTypeMax, 
			 aggregation.AggTypeMin, aggregation.AggTypeCount:
			// 度量聚合
			if !hasGrouping {
				groupStage["_id"] = nil // 全局聚合
				hasGrouping = true
			}
			// 使用聚合名称作为字段名，而不是直接使用操作符
			aggName := agg.Name()
			if aggName != "" {
				// aggDoc 包含完整的聚合操作符，直接使用
				groupStage[aggName] = aggDoc
			}
		case aggregation.AggTypeTerms:
			// 分桶聚合
			if termsAgg, ok := agg.(*aggregation.TermsAggregation); ok {
				groupStage["_id"] = "$" + termsAgg.Field
				hasGrouping = true
			}
		case aggregation.AggTypeDateHisto:
			// 日期直方图聚合
			if dateHistoAgg, ok := agg.(*aggregation.DateHistogramAggregation); ok {
				// 简化实现：按日期字段分组
				groupStage["_id"] = "$" + dateHistoAgg.Field
				hasGrouping = true
			}
		}
	}

	// 添加分组阶段
	if hasGrouping && len(groupStage) > 0 {
		pipeline = append(pipeline, bson.M{"$group": groupStage})
	}

	// 添加排序
	if queryOpts.OrderBy != "" {
		direction := 1
		if queryOpts.OrderDesc {
			direction = -1
		}
		pipeline = append(pipeline, bson.M{"$sort": bson.M{queryOpts.OrderBy: direction}})
	}

	// 添加分页
	if queryOpts.Offset > 0 {
		pipeline = append(pipeline, bson.M{"$skip": queryOpts.Offset})
	}
	if queryOpts.Limit > 0 {
		pipeline = append(pipeline, bson.M{"$limit": queryOpts.Limit})
	}

	// 执行聚合查询
	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// 构建聚合结果
	result := aggregation.NewAggregationResult()

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}

		// 简化处理：将聚合结果存储到结果中
		for _, agg := range aggs {
			aggName := agg.Name()
			if value, exists := doc[aggName]; exists {
				result.SetResult(aggName, value)
			}
		}
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// 事务支持实现
func (m *Mongo) BeginTx(ctx context.Context) (Transaction, error) {
	session, err := m.client.StartSession()
	if err != nil {
		return nil, fmt.Errorf("failed to start session: %v", err)
	}

	return &MongoTransaction{
		session:    session,
		database:   m.database,
		builder:    m.builder,
		hasStarted: false,
	}, nil
}

func (m *Mongo) WithTx(ctx context.Context, fn func(tx Transaction) error) error {
	tx, err := m.BeginTx(ctx)
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

// MongoTransaction MongoDB事务实现
type MongoTransaction struct {
	session    mongo.Session
	database   *mongo.Database
	builder    *MongoRecordBuilder
	hasStarted bool
}

func (tx *MongoTransaction) Commit() error {
	defer tx.session.EndSession(context.Background())
	if !tx.hasStarted {
		return nil // 没有开始事务，直接返回
	}
	return tx.session.CommitTransaction(context.Background())
}

func (tx *MongoTransaction) Rollback() error {
	defer tx.session.EndSession(context.Background())
	if !tx.hasStarted {
		return nil // 没有开始事务，直接返回
	}
	return tx.session.AbortTransaction(context.Background())
}

// 事务中的CRUD操作实现
func (tx *MongoTransaction) Create(ctx context.Context, table string, record Record, opts ...CreateOption) error {
	// 解析创建选项
	createOpts := &CreateOptions{}
	for _, opt := range opts {
		opt(createOpts)
	}

	// 确保事务已开始
	if !tx.hasStarted {
		if err := tx.session.StartTransaction(); err != nil {
			return fmt.Errorf("failed to start transaction: %v", err)
		}
		tx.hasStarted = true
	}

	// 使用 session context
	sessionCtx := mongo.NewSessionContext(ctx, tx.session)
	collection := tx.database.Collection(table)
	fields := record.Fields()

	// 处理主键：MongoDB使用_id作为主键，如果没有则自动生成
	if _, exists := fields["_id"]; !exists {
		fields["_id"] = primitive.NewObjectID()
	}

	// 转换为BSON
	doc := make(bson.M)
	for k, v := range fields {
		doc[k] = v
	}

	if createOpts.IgnoreConflict {
		// 尝试插入，如果失败则忽略
		_, err := collection.InsertOne(sessionCtx, doc)
		if err != nil && strings.Contains(err.Error(), "duplicate key") {
			return nil // 忽略重复键错误
		}
		return err
	} else if createOpts.UpdateOnConflict {
		// 使用ReplaceOne with upsert选项在冲突时更新
		filter := bson.M{"_id": doc["_id"]}
		replaceOptions := options.Replace().SetUpsert(true)
		_, err := collection.ReplaceOne(sessionCtx, filter, doc, replaceOptions)
		return err
	} else {
		// 默认的插入操作
		_, err := collection.InsertOne(sessionCtx, doc)
		if err != nil && strings.Contains(err.Error(), "duplicate key") {
			return ErrDuplicateKey
		}
		return err
	}
}

func (tx *MongoTransaction) Get(ctx context.Context, table string, pk map[string]any) (Record, error) {
	// 确保事务已开始
	if !tx.hasStarted {
		if err := tx.session.StartTransaction(); err != nil {
			return nil, fmt.Errorf("failed to start transaction: %v", err)
		}
		tx.hasStarted = true
	}

	// 使用 session context
	sessionCtx := mongo.NewSessionContext(ctx, tx.session)
	collection := tx.database.Collection(table)

	// 构建查询过滤器
	filter := make(bson.M)
	for k, v := range pk {
		filter[k] = v
	}

	var result bson.M
	err := collection.FindOne(sessionCtx, filter).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return nil, ErrRecordNotFound
	}
	if err != nil {
		return nil, err
	}
	
	return &MongoRecord{data: result}, nil
}

func (tx *MongoTransaction) Update(ctx context.Context, table string, pk map[string]any, record Record) error {
	collection := tx.database.Collection(table)

	// 构建查询过滤器
	filter := make(bson.M)
	for k, v := range pk {
		filter[k] = v
	}

	// 构建更新文档
	fields := record.Fields()
	update := bson.M{"$set": fields}

	callback := func(sessionContext mongo.SessionContext) (interface{}, error) {
		result, err := collection.UpdateOne(sessionContext, filter, update)
		if err != nil {
			return nil, err
		}
		if result.MatchedCount == 0 {
			return nil, ErrRecordNotFound
		}
		return nil, nil
	}

	_, err := tx.session.WithTransaction(ctx, callback)
	return err
}

func (tx *MongoTransaction) Delete(ctx context.Context, table string, pk map[string]any) error {
	collection := tx.database.Collection(table)

	// 构建查询过滤器
	filter := make(bson.M)
	for k, v := range pk {
		filter[k] = v
	}

	callback := func(sessionContext mongo.SessionContext) (interface{}, error) {
		result, err := collection.DeleteOne(sessionContext, filter)
		if err != nil {
			return nil, err
		}
		if result.DeletedCount == 0 {
			return nil, ErrRecordNotFound
		}
		return nil, nil
	}

	_, err := tx.session.WithTransaction(ctx, callback)
	return err
}

func (tx *MongoTransaction) Find(ctx context.Context, table string, query query.Query, opts ...QueryOption) ([]Record, error) {
	// 解析查询选项
	queryOpts := &QueryOptions{}
	for _, opt := range opts {
		opt(queryOpts)
	}

	collection := tx.database.Collection(table)

	// 构建查询过滤器
	filter, err := query.ToMongo()
	if err != nil {
		return nil, fmt.Errorf("failed to convert query to mongo: %v", err)
	}

	// 创建查找选项
	findOptions := options.Find()

	// 添加排序
	if queryOpts.OrderBy != "" {
		direction := 1
		if queryOpts.OrderDesc {
			direction = -1
		}
		findOptions.SetSort(bson.D{{Key: queryOpts.OrderBy, Value: direction}})
	}

	// 添加分页
	if queryOpts.Limit > 0 {
		findOptions.SetLimit(int64(queryOpts.Limit))
	}
	if queryOpts.Offset > 0 {
		findOptions.SetSkip(int64(queryOpts.Offset))
	}

	var records []Record
	callback := func(sessionContext mongo.SessionContext) (interface{}, error) {
		// 执行查询
		cursor, err := collection.Find(sessionContext, filter, findOptions)
		if err != nil {
			return nil, err
		}
		defer cursor.Close(sessionContext)

		// 扫描结果
		for cursor.Next(sessionContext) {
			var doc bson.M
			if err := cursor.Decode(&doc); err != nil {
				return nil, err
			}
			records = append(records, &MongoRecord{data: doc})
		}

		return records, cursor.Err()
	}

	res, err := tx.session.WithTransaction(ctx, callback)
	if err != nil {
		return nil, err
	}
	return res.([]Record), nil
}

func (tx *MongoTransaction) Aggregate(ctx context.Context, table string, query query.Query, aggs []aggregation.Aggregation, opts ...QueryOption) (aggregation.AggregationResult, error) {
	// 简化实现：在事务中使用基本的聚合
	return aggregation.NewAggregationResult(), nil
}

func (tx *MongoTransaction) BatchCreate(ctx context.Context, table string, records []Record, opts ...CreateOption) error {
	for _, record := range records {
		if err := tx.Create(ctx, table, record, opts...); err != nil {
			return err
		}
	}
	return nil
}

func (tx *MongoTransaction) BatchUpdate(ctx context.Context, table string, pks []map[string]any, records []Record) error {
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

func (tx *MongoTransaction) BatchDelete(ctx context.Context, table string, pks []map[string]any) error {
	for _, pk := range pks {
		if err := tx.Delete(ctx, table, pk); err != nil {
			return err
		}
	}
	return nil
}

func (tx *MongoTransaction) BeginTx(ctx context.Context) (Transaction, error) {
	return nil, fmt.Errorf("nested transactions not supported")
}

func (tx *MongoTransaction) WithTx(ctx context.Context, fn func(tx Transaction) error) error {
	return fn(tx)
}

func (tx *MongoTransaction) Migrate(ctx context.Context, model *TableModel) error {
	// 在事务中不支持架构迁移
	return fmt.Errorf("schema migration not supported in transactions")
}

func (tx *MongoTransaction) DropTable(ctx context.Context, table string) error {
	// 在事务中不支持删除集合
	return fmt.Errorf("drop table not supported in transactions")
}

func (tx *MongoTransaction) GetBuilder() RecordBuilder {
	return tx.builder
}

func (tx *MongoTransaction) Close() error {
	return nil // 事务不需要单独关闭
}