package database

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/hatlonely/gox/rdb/aggregation"
	"github.com/hatlonely/gox/rdb/query"
	_ "github.com/mattn/go-sqlite3"
)

type SQLOptions struct {
	Driver   string `cfg:"driver" def:"mysql"`
	DSN      string `cfg:"dsn"`
	Host     string `cfg:"host" def:"localhost"`
	Port     string `cfg:"port" def:"3306"`
	Database string `cfg:"database"`
	Username string `cfg:"username"`
	Password string `cfg:"password"`
	Charset  string `cfg:"charset" def:"utf8mb4"`
	MaxConns int    `cfg:"maxConns" def:"10"`
	MaxIdle  int    `cfg:"maxIdle" def:"5"`
}

type SQL struct {
	db      *sql.DB
	builder *SQLRecordBuilder
	driver  string
}

func NewSQLWithOptions(options *SQLOptions) (*SQL, error) {
	dsn := options.DSN
	if dsn == "" {
		switch options.Driver {
		case "mysql":
			dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local",
				options.Username, options.Password, options.Host, options.Port, options.Database, options.Charset)
		case "sqlite3":
			dsn = options.Database
		default:
			return nil, fmt.Errorf("unsupported driver: %s", options.Driver)
		}
	}

	db, err := sql.Open(options.Driver, dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(options.MaxConns)
	db.SetMaxIdleConns(options.MaxIdle)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &SQL{
		db:      db,
		builder: &SQLRecordBuilder{},
		driver:  options.Driver,
	}, nil
}

type SQLRecord struct {
	data map[string]any
}

func (r *SQLRecord) Scan(dest any) error {
	return mapToStruct(r.data, dest)
}

func (r *SQLRecord) ScanStruct(dest any) error {
	return r.Scan(dest)
}

func (r *SQLRecord) Fields() map[string]any {
	return r.data
}

type SQLRecordBuilder struct{}

func (b *SQLRecordBuilder) FromStruct(v any) Record {
	data := structToMap(v)
	return &SQLRecord{data: data}
}

func (b *SQLRecordBuilder) FromMap(data map[string]any, table string) Record {
	return &SQLRecord{data: data}
}

// 辅助函数：结构体转换为 map
func structToMap(v any) map[string]any {
	result := make(map[string]any)
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

		// 检查 rdb 标签
		tag := field.Tag.Get("rdb")
		if tag == "-" {
			continue // 跳过被忽略的字段
		}

		fieldName := field.Name
		if tag != "" {
			if idx := strings.Index(tag, ","); idx != -1 {
				fieldName = tag[:idx]
			} else {
				fieldName = tag
			}
		}

		value := rv.Field(i).Interface()
		result[fieldName] = value
	}
	return result
}

// 辅助函数：map 转换为结构体
func mapToStruct(data map[string]any, dest any) error {
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
		}

		if value, exists := data[fieldName]; exists && value != nil {
			fieldValue := rv.Field(i)
			if fieldValue.CanSet() {
				if err := setFieldValue(fieldValue, value); err != nil {
					return fmt.Errorf("failed to set field %s: %v", fieldName, err)
				}
			}
		}
	}
	return nil
}

// 辅助函数：设置字段值
func setFieldValue(fieldValue reflect.Value, value any) error {
	if value == nil {
		return nil
	}

	valueType := reflect.TypeOf(value)
	fieldType := fieldValue.Type()

	// 特殊处理：MySQL BOOLEAN 字段返回 int64，需要转换为 bool
	if fieldType.Kind() == reflect.Bool {
		switch v := value.(type) {
		case int64:
			fieldValue.SetBool(v != 0)
			return nil
		case int:
			fieldValue.SetBool(v != 0)
			return nil
		case bool:
			fieldValue.SetBool(v)
			return nil
		}
	}

	// 特殊处理：time.Time 字段
	if fieldType == reflect.TypeOf(time.Time{}) {
		switch v := value.(type) {
		case time.Time:
			fieldValue.Set(reflect.ValueOf(v))
			return nil
		case string:
			// 尝试多种时间格式解析
			timeFormats := []string{
				"2006-01-02 15:04:05.999999-07:00", // SQLite 格式
				"2006-01-02 15:04:05.999999+07:00", // SQLite 格式
				"2006-01-02 15:04:05",             // 标准格式
				time.RFC3339,                      // RFC3339
				time.RFC3339Nano,                  // RFC3339 with nanoseconds
			}
			
			var parsedTime time.Time
			var lastErr error
			for _, format := range timeFormats {
				parsedTime, lastErr = time.Parse(format, v)
				if lastErr == nil {
					fieldValue.Set(reflect.ValueOf(parsedTime))
					return nil
				}
			}
			return fmt.Errorf("cannot parse time string %s: %v", v, lastErr)
		}
	}

	// 特殊处理：数据库返回的数字类型转换
	if fieldType.Kind() == reflect.Int && valueType.Kind() == reflect.Int64 {
		fieldValue.SetInt(value.(int64))
		return nil
	}

	if fieldType.Kind() == reflect.Float64 && valueType.Kind() == reflect.Float32 {
		fieldValue.SetFloat(float64(value.(float32)))
		return nil
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

// 实现 Database 接口
func (s *SQL) Migrate(ctx context.Context, model *TableModel) error {
	// 构建 CREATE TABLE 语句
	createTableSQL := s.buildCreateTableSQL(model)

	// 执行创建表语句
	if _, err := s.db.ExecContext(ctx, createTableSQL); err != nil {
		// 如果表已存在，忽略错误（可根据需要调整策略）
		if !strings.Contains(err.Error(), "already exists") && !strings.Contains(err.Error(), "already exist") {
			return fmt.Errorf("failed to create table %s: %v", model.Table, err)
		}
	}

	// 创建索引
	for _, index := range model.Indexes {
		indexSQL := s.buildCreateIndexSQL(model.Table, index)
		if _, err := s.db.ExecContext(ctx, indexSQL); err != nil {
			// 如果索引已存在，忽略错误
			if !strings.Contains(err.Error(), "already exists") &&
				!strings.Contains(err.Error(), "already exist") &&
				!strings.Contains(err.Error(), "Duplicate key name") {
				return fmt.Errorf("failed to create index %s: %v", index.Name, err)
			}
		}
	}

	return nil
}

// buildCreateTableSQL 构建创建表的 SQL 语句
func (s *SQL) buildCreateTableSQL(model *TableModel) string {
	var columns []string

	// 构建字段定义
	for _, field := range model.Fields {
		columnDef := s.buildColumnDefinition(field)
		columns = append(columns, columnDef)
	}

	// 添加主键定义
	if len(model.PrimaryKey) > 0 {
		pkDef := fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(model.PrimaryKey, ", "))
		columns = append(columns, pkDef)
	}

	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n  %s\n)",
		model.Table, strings.Join(columns, ",\n  "))
}

// buildColumnDefinition 构建单个字段定义
func (s *SQL) buildColumnDefinition(field FieldDefinition) string {
	var parts []string

	// 字段名和类型
	parts = append(parts, field.Name)
	parts = append(parts, s.mapFieldTypeToSQL(field.Type, field.Size))

	// 是否必需
	if field.Required {
		parts = append(parts, "NOT NULL")
	}

	// 默认值
	if field.Default != nil {
		defaultValue := s.formatDefaultValue(field.Default)
		parts = append(parts, fmt.Sprintf("DEFAULT %s", defaultValue))
	}

	return strings.Join(parts, " ")
}

// mapFieldTypeToSQL 将字段类型映射为 SQL 类型
func (s *SQL) mapFieldTypeToSQL(fieldType FieldType, size int) string {
	switch fieldType {
	case FieldTypeString:
		if s.driver == "sqlite3" {
			return "TEXT"
		}
		if size > 0 {
			return fmt.Sprintf("VARCHAR(%d)", size)
		}
		return "VARCHAR(255)"
	case FieldTypeInt:
		if s.driver == "sqlite3" {
			return "INTEGER"
		}
		return "INT"
	case FieldTypeFloat:
		if s.driver == "sqlite3" {
			return "REAL"
		}
		return "FLOAT"
	case FieldTypeBool:
		if s.driver == "sqlite3" {
			return "INTEGER"
		}
		return "BOOLEAN"
	case FieldTypeDate:
		if s.driver == "sqlite3" {
			return "TEXT"
		}
		return "DATETIME"
	case FieldTypeJSON:
		if s.driver == "mysql" {
			return "JSON"
		}
		return "TEXT"
	default:
		if s.driver == "sqlite3" {
			return "TEXT"
		}
		return "VARCHAR(255)"
	}
}

// formatDefaultValue 格式化默认值
func (s *SQL) formatDefaultValue(value any) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case bool:
		if v {
			return "1"
		}
		return "0"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// buildCreateIndexSQL 构建创建索引的 SQL 语句
func (s *SQL) buildCreateIndexSQL(table string, index IndexDefinition) string {
	indexType := "INDEX"
	if index.Unique {
		indexType = "UNIQUE INDEX"
	}

	// MySQL 不支持 IF NOT EXISTS 语法用于索引
	if s.driver == "mysql" {
		return fmt.Sprintf("CREATE %s %s ON %s (%s)",
			indexType, index.Name, table, strings.Join(index.Fields, ", "))
	}

	return fmt.Sprintf("CREATE %s IF NOT EXISTS %s ON %s (%s)",
		indexType, index.Name, table, strings.Join(index.Fields, ", "))
}

func (s *SQL) DropTable(ctx context.Context, table string) error {
	sqlStr := fmt.Sprintf("DROP TABLE IF EXISTS %s", table)
	_, err := s.db.ExecContext(ctx, sqlStr)
	return err
}

func (s *SQL) GetBuilder() RecordBuilder {
	return s.builder
}

func (s *SQL) Close() error {
	return s.db.Close()
}

// 辅助函数：将参数占位符格式化为对应数据库的格式
func (s *SQL) formatSQL(sqlStr string, args []any) (string, []any) {
	if s.driver == "postgres" {
		// PostgreSQL 使用 $1, $2, $3... 格式
		count := 1
		for strings.Contains(sqlStr, "?") {
			sqlStr = strings.Replace(sqlStr, "?", fmt.Sprintf("$%d", count), 1)
			count++
		}
	}
	return sqlStr, args
}

// 辅助函数：扫描数据库行到 Record
func (s *SQL) scanRowToRecord(rows *sql.Rows) (Record, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := rows.Scan(valuePtrs...); err != nil {
		return nil, err
	}

	data := make(map[string]any)
	for i, col := range columns {
		data[col] = values[i]
	}

	return &SQLRecord{data: data}, nil
}

// CRUD 操作实现
func (s *SQL) Create(ctx context.Context, table string, record Record, opts ...CreateOption) error {
	// 解析创建选项
	options := &CreateOptions{}
	for _, opt := range opts {
		opt(options)
	}

	fields := record.Fields()

	var columns []string
	var placeholders []string
	var args []any

	for col, val := range fields {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		args = append(args, val)
	}

	var sqlStr string
	if options.IgnoreConflict {
		// 使用 INSERT IGNORE 语法忽略冲突
		if s.driver == "mysql" {
			sqlStr = fmt.Sprintf("INSERT IGNORE INTO %s (%s) VALUES (%s)",
				table,
				strings.Join(columns, ", "),
				strings.Join(placeholders, ", "))
		} else {
			// SQLite 使用 INSERT OR IGNORE
			sqlStr = fmt.Sprintf("INSERT OR IGNORE INTO %s (%s) VALUES (%s)",
				table,
				strings.Join(columns, ", "),
				strings.Join(placeholders, ", "))
		}
	} else if options.UpdateOnConflict {
		// 使用 ON DUPLICATE KEY UPDATE 语法在冲突时更新
		if s.driver == "mysql" {
			var updateParts []string
			for col := range fields {
				updateParts = append(updateParts, fmt.Sprintf("%s = VALUES(%s)", col, col))
			}
			sqlStr = fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON DUPLICATE KEY UPDATE %s",
				table,
				strings.Join(columns, ", "),
				strings.Join(placeholders, ", "),
				strings.Join(updateParts, ", "))
		} else {
			// SQLite 使用 INSERT OR REPLACE
			sqlStr = fmt.Sprintf("INSERT OR REPLACE INTO %s (%s) VALUES (%s)",
				table,
				strings.Join(columns, ", "),
				strings.Join(placeholders, ", "))
		}
	} else {
		// 默认的 INSERT 语法
		sqlStr = fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			table,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "))
	}

	sqlStr, args = s.formatSQL(sqlStr, args)
	_, err := s.db.ExecContext(ctx, sqlStr, args...)
	return err
}

func (s *SQL) Get(ctx context.Context, table string, pk map[string]any) (Record, error) {
	var whereParts []string
	var args []any

	for col, val := range pk {
		whereParts = append(whereParts, fmt.Sprintf("%s = ?", col))
		args = append(args, val)
	}

	sqlStr := fmt.Sprintf("SELECT * FROM %s WHERE %s",
		table, strings.Join(whereParts, " AND "))

	sqlStr, args = s.formatSQL(sqlStr, args)
	rows, err := s.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, ErrRecordNotFound
	}

	return s.scanRowToRecord(rows)
}

func (s *SQL) Update(ctx context.Context, table string, pk map[string]any, record Record) error {
	fields := record.Fields()

	var setParts []string
	var args []any

	for col, val := range fields {
		setParts = append(setParts, fmt.Sprintf("%s = ?", col))
		args = append(args, val)
	}

	var whereParts []string
	for col, val := range pk {
		whereParts = append(whereParts, fmt.Sprintf("%s = ?", col))
		args = append(args, val)
	}

	sqlStr := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		table,
		strings.Join(setParts, ", "),
		strings.Join(whereParts, " AND "))

	sqlStr, args = s.formatSQL(sqlStr, args)
	_, err := s.db.ExecContext(ctx, sqlStr, args...)
	return err
}

func (s *SQL) Delete(ctx context.Context, table string, pk map[string]any) error {
	var whereParts []string
	var args []any

	for col, val := range pk {
		whereParts = append(whereParts, fmt.Sprintf("%s = ?", col))
		args = append(args, val)
	}

	sqlStr := fmt.Sprintf("DELETE FROM %s WHERE %s",
		table, strings.Join(whereParts, " AND "))

	sqlStr, args = s.formatSQL(sqlStr, args)
	_, err := s.db.ExecContext(ctx, sqlStr, args...)
	return err
}

// 查询和聚合功能实现
func (s *SQL) Find(ctx context.Context, table string, query query.Query, opts ...QueryOption) ([]Record, error) {
	// 解析查询选项
	options := &QueryOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// 构建 WHERE 条件
	whereSQL, whereArgs, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	// 构建完整 SQL
	sqlStr := fmt.Sprintf("SELECT * FROM %s WHERE %s", table, whereSQL)

	// 添加排序
	if options.OrderBy != "" {
		direction := "ASC"
		if options.OrderDesc {
			direction = "DESC"
		}
		sqlStr += fmt.Sprintf(" ORDER BY %s %s", options.OrderBy, direction)
	}

	// 添加分页
	if options.Limit > 0 {
		sqlStr += fmt.Sprintf(" LIMIT %d", options.Limit)
	}
	if options.Offset > 0 {
		sqlStr += fmt.Sprintf(" OFFSET %d", options.Offset)
	}

	// 执行查询
	sqlStr, whereArgs = s.formatSQL(sqlStr, whereArgs)
	rows, err := s.db.QueryContext(ctx, sqlStr, whereArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 扫描结果
	var records []Record
	for rows.Next() {
		record, err := s.scanRowToRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil
}

func (s *SQL) Aggregate(ctx context.Context, table string, query query.Query, aggs []aggregation.Aggregation, opts ...QueryOption) (aggregation.AggregationResult, error) {
	// 解析查询选项
	options := &QueryOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// 构建 WHERE 条件
	whereSQL, whereArgs, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	// 构建聚合查询
	var selectParts []string
	var groupByParts []string

	for _, agg := range aggs {
		aggSQL, _, err := agg.ToSQL()
		if err != nil {
			return nil, err
		}
		selectParts = append(selectParts, aggSQL)

		// 处理桶聚合的 GROUP BY
		switch agg.Type() {
		case aggregation.AggTypeTerms:
			if termsAgg, ok := agg.(*aggregation.TermsAggregation); ok {
				selectParts = append(selectParts, termsAgg.Field)
				groupByParts = append(groupByParts, termsAgg.Field)
			}
		case aggregation.AggTypeDateHisto:
			if dateHistoAgg, ok := agg.(*aggregation.DateHistogramAggregation); ok {
				// 简化实现：直接使用字段进行分组
				selectParts = append(selectParts, dateHistoAgg.Field)
				groupByParts = append(groupByParts, dateHistoAgg.Field)
			}
		}
	}

	// 构建完整 SQL
	sqlStr := fmt.Sprintf("SELECT %s FROM %s WHERE %s",
		strings.Join(selectParts, ", "), table, whereSQL)

	if len(groupByParts) > 0 {
		sqlStr += " GROUP BY " + strings.Join(groupByParts, ", ")
	}

	// 添加排序
	if options.OrderBy != "" {
		direction := "ASC"
		if options.OrderDesc {
			direction = "DESC"
		}
		sqlStr += fmt.Sprintf(" ORDER BY %s %s", options.OrderBy, direction)
	}

	// 添加分页
	if options.Limit > 0 {
		sqlStr += fmt.Sprintf(" LIMIT %d", options.Limit)
	}

	// 执行聚合查询
	sqlStr, whereArgs = s.formatSQL(sqlStr, whereArgs)
	rows, err := s.db.QueryContext(ctx, sqlStr, whereArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 构建聚合结果
	result := aggregation.NewAggregationResult()

	for rows.Next() {
		record, err := s.scanRowToRecord(rows)
		if err != nil {
			return nil, err
		}

		// 简化处理：将第一个聚合的结果作为主要结果
		if len(aggs) > 0 {
			data := record.Fields()
			aggName := aggs[0].Name()
			if value, exists := data[aggName]; exists {
				result.SetResult(aggName, value)
			}
		}
	}

	return result, nil
}

// 批量操作实现
func (s *SQL) BatchCreate(ctx context.Context, table string, records []Record, opts ...CreateOption) error {
	for _, record := range records {
		if err := s.Create(ctx, table, record, opts...); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQL) BatchUpdate(ctx context.Context, table string, pks []map[string]any, records []Record) error {
	if len(pks) != len(records) {
		return fmt.Errorf("pks and records length mismatch")
	}

	for i, record := range records {
		if err := s.Update(ctx, table, pks[i], record); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQL) BatchDelete(ctx context.Context, table string, pks []map[string]any) error {
	for _, pk := range pks {
		if err := s.Delete(ctx, table, pk); err != nil {
			return err
		}
	}
	return nil
}

// 事务相关实现
func (s *SQL) BeginTx(ctx context.Context) (Transaction, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &SQLTransaction{
		tx:      tx,
		builder: s.builder,
		driver:  s.driver,
	}, nil
}

func (s *SQL) WithTx(ctx context.Context, fn func(tx Transaction) error) error {
	tx, err := s.BeginTx(ctx)
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

// SQL 事务实现
type SQLTransaction struct {
	tx      *sql.Tx
	builder *SQLRecordBuilder
	driver  string
}

func (tx *SQLTransaction) Commit() error {
	return tx.tx.Commit()
}

func (tx *SQLTransaction) Rollback() error {
	return tx.tx.Rollback()
}

// 事务中的 CRUD 操作实现 (复用 SQL 的逻辑，但使用事务连接)
func (tx *SQLTransaction) Create(ctx context.Context, table string, record Record, opts ...CreateOption) error {
	// 解析创建选项
	options := &CreateOptions{}
	for _, opt := range opts {
		opt(options)
	}

	fields := record.Fields()

	var columns []string
	var placeholders []string
	var args []any

	for col, val := range fields {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		args = append(args, val)
	}

	var sqlStr string
	if options.IgnoreConflict {
		// 使用 INSERT IGNORE 语法忽略冲突
		if tx.driver == "mysql" {
			sqlStr = fmt.Sprintf("INSERT IGNORE INTO %s (%s) VALUES (%s)",
				table,
				strings.Join(columns, ", "),
				strings.Join(placeholders, ", "))
		} else {
			// SQLite 使用 INSERT OR IGNORE
			sqlStr = fmt.Sprintf("INSERT OR IGNORE INTO %s (%s) VALUES (%s)",
				table,
				strings.Join(columns, ", "),
				strings.Join(placeholders, ", "))
		}
	} else if options.UpdateOnConflict {
		// 使用 ON DUPLICATE KEY UPDATE 语法在冲突时更新
		if tx.driver == "mysql" {
			var updateParts []string
			for col := range fields {
				updateParts = append(updateParts, fmt.Sprintf("%s = VALUES(%s)", col, col))
			}
			sqlStr = fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON DUPLICATE KEY UPDATE %s",
				table,
				strings.Join(columns, ", "),
				strings.Join(placeholders, ", "),
				strings.Join(updateParts, ", "))
		} else {
			// SQLite 使用 INSERT OR REPLACE
			sqlStr = fmt.Sprintf("INSERT OR REPLACE INTO %s (%s) VALUES (%s)",
				table,
				strings.Join(columns, ", "),
				strings.Join(placeholders, ", "))
		}
	} else {
		// 默认的 INSERT 语法
		sqlStr = fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			table,
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "))
	}

	sqlStr, args = tx.formatSQL(sqlStr, args)
	_, err := tx.tx.ExecContext(ctx, sqlStr, args...)
	return err
}

func (tx *SQLTransaction) Get(ctx context.Context, table string, pk map[string]any) (Record, error) {
	var whereParts []string
	var args []any

	for col, val := range pk {
		whereParts = append(whereParts, fmt.Sprintf("%s = ?", col))
		args = append(args, val)
	}

	sqlStr := fmt.Sprintf("SELECT * FROM %s WHERE %s",
		table, strings.Join(whereParts, " AND "))

	sqlStr, args = tx.formatSQL(sqlStr, args)
	rows, err := tx.tx.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, ErrRecordNotFound
	}

	return tx.scanRowToRecord(rows)
}

func (tx *SQLTransaction) Update(ctx context.Context, table string, pk map[string]any, record Record) error {
	fields := record.Fields()

	var setParts []string
	var args []any

	for col, val := range fields {
		setParts = append(setParts, fmt.Sprintf("%s = ?", col))
		args = append(args, val)
	}

	var whereParts []string
	for col, val := range pk {
		whereParts = append(whereParts, fmt.Sprintf("%s = ?", col))
		args = append(args, val)
	}

	sqlStr := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		table,
		strings.Join(setParts, ", "),
		strings.Join(whereParts, " AND "))

	sqlStr, args = tx.formatSQL(sqlStr, args)
	_, err := tx.tx.ExecContext(ctx, sqlStr, args...)
	return err
}

func (tx *SQLTransaction) Delete(ctx context.Context, table string, pk map[string]any) error {
	var whereParts []string
	var args []any

	for col, val := range pk {
		whereParts = append(whereParts, fmt.Sprintf("%s = ?", col))
		args = append(args, val)
	}

	sqlStr := fmt.Sprintf("DELETE FROM %s WHERE %s",
		table, strings.Join(whereParts, " AND "))

	sqlStr, args = tx.formatSQL(sqlStr, args)
	_, err := tx.tx.ExecContext(ctx, sqlStr, args...)
	return err
}

func (tx *SQLTransaction) Find(ctx context.Context, table string, query query.Query, opts ...QueryOption) ([]Record, error) {
	// 解析查询选项
	options := &QueryOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// 构建 WHERE 条件
	whereSQL, whereArgs, err := query.ToSQL()
	if err != nil {
		return nil, err
	}

	// 构建完整 SQL
	sqlStr := fmt.Sprintf("SELECT * FROM %s WHERE %s", table, whereSQL)

	// 添加排序和分页
	if options.OrderBy != "" {
		direction := "ASC"
		if options.OrderDesc {
			direction = "DESC"
		}
		sqlStr += fmt.Sprintf(" ORDER BY %s %s", options.OrderBy, direction)
	}

	if options.Limit > 0 {
		sqlStr += fmt.Sprintf(" LIMIT %d", options.Limit)
	}
	if options.Offset > 0 {
		sqlStr += fmt.Sprintf(" OFFSET %d", options.Offset)
	}

	// 执行查询
	sqlStr, whereArgs = tx.formatSQL(sqlStr, whereArgs)
	rows, err := tx.tx.QueryContext(ctx, sqlStr, whereArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 扫描结果
	var records []Record
	for rows.Next() {
		record, err := tx.scanRowToRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil
}

// 事务中的其他方法实现（简化版本）
func (tx *SQLTransaction) Aggregate(ctx context.Context, table string, query query.Query, aggs []aggregation.Aggregation, opts ...QueryOption) (aggregation.AggregationResult, error) {
	return aggregation.NewAggregationResult(), nil // 简化实现
}

func (tx *SQLTransaction) BatchCreate(ctx context.Context, table string, records []Record, opts ...CreateOption) error {
	for _, record := range records {
		if err := tx.Create(ctx, table, record, opts...); err != nil {
			return err
		}
	}
	return nil
}

func (tx *SQLTransaction) BatchUpdate(ctx context.Context, table string, pks []map[string]any, records []Record) error {
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

func (tx *SQLTransaction) BatchDelete(ctx context.Context, table string, pks []map[string]any) error {
	for _, pk := range pks {
		if err := tx.Delete(ctx, table, pk); err != nil {
			return err
		}
	}
	return nil
}

func (tx *SQLTransaction) BeginTx(ctx context.Context) (Transaction, error) {
	return nil, fmt.Errorf("nested transactions not supported")
}

func (tx *SQLTransaction) WithTx(ctx context.Context, fn func(tx Transaction) error) error {
	return fn(tx)
}

func (tx *SQLTransaction) Migrate(ctx context.Context, model *TableModel) error {
	// 构建 CREATE TABLE 语句
	createTableSQL := tx.buildCreateTableSQL(model)

	// 执行创建表语句
	if _, err := tx.tx.ExecContext(ctx, createTableSQL); err != nil {
		// 如果表已存在，忽略错误（可根据需要调整策略）
		if !strings.Contains(err.Error(), "already exists") && !strings.Contains(err.Error(), "already exist") {
			return fmt.Errorf("failed to create table %s: %v", model.Table, err)
		}
	}

	// 创建索引
	for _, index := range model.Indexes {
		indexSQL := tx.buildCreateIndexSQL(model.Table, index)
		if _, err := tx.tx.ExecContext(ctx, indexSQL); err != nil {
			// 如果索引已存在，忽略错误
			if !strings.Contains(err.Error(), "already exists") &&
				!strings.Contains(err.Error(), "already exist") &&
				!strings.Contains(err.Error(), "Duplicate key name") {
				return fmt.Errorf("failed to create index %s: %v", index.Name, err)
			}
		}
	}

	return nil
}

func (tx *SQLTransaction) DropTable(ctx context.Context, table string) error {
	sqlStr := fmt.Sprintf("DROP TABLE IF EXISTS %s", table)
	_, err := tx.tx.ExecContext(ctx, sqlStr)
	return err
}

func (tx *SQLTransaction) GetBuilder() RecordBuilder {
	return tx.builder
}

func (tx *SQLTransaction) Close() error {
	return nil // 事务不需要单独关闭
}

// 事务的辅助方法
func (tx *SQLTransaction) formatSQL(sqlStr string, args []any) (string, []any) {
	if tx.driver == "postgres" {
		count := 1
		for strings.Contains(sqlStr, "?") {
			sqlStr = strings.Replace(sqlStr, "?", fmt.Sprintf("$%d", count), 1)
			count++
		}
	}
	return sqlStr, args
}

// buildCreateTableSQL 构建创建表的 SQL 语句 (事务版本)
func (tx *SQLTransaction) buildCreateTableSQL(model *TableModel) string {
	var columns []string

	// 构建字段定义
	for _, field := range model.Fields {
		columnDef := tx.buildColumnDefinition(field)
		columns = append(columns, columnDef)
	}

	// 添加主键定义
	if len(model.PrimaryKey) > 0 {
		pkDef := fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(model.PrimaryKey, ", "))
		columns = append(columns, pkDef)
	}

	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n  %s\n)",
		model.Table, strings.Join(columns, ",\n  "))
}

// buildColumnDefinition 构建单个字段定义 (事务版本)
func (tx *SQLTransaction) buildColumnDefinition(field FieldDefinition) string {
	var parts []string

	// 字段名和类型
	parts = append(parts, field.Name)
	parts = append(parts, tx.mapFieldTypeToSQL(field.Type, field.Size))

	// 是否必需
	if field.Required {
		parts = append(parts, "NOT NULL")
	}

	// 默认值
	if field.Default != nil {
		defaultValue := tx.formatDefaultValue(field.Default)
		parts = append(parts, fmt.Sprintf("DEFAULT %s", defaultValue))
	}

	return strings.Join(parts, " ")
}

// mapFieldTypeToSQL 将字段类型映射为 SQL 类型 (事务版本)
func (tx *SQLTransaction) mapFieldTypeToSQL(fieldType FieldType, size int) string {
	switch fieldType {
	case FieldTypeString:
		if tx.driver == "sqlite3" {
			return "TEXT"
		}
		if size > 0 {
			return fmt.Sprintf("VARCHAR(%d)", size)
		}
		return "VARCHAR(255)"
	case FieldTypeInt:
		if tx.driver == "sqlite3" {
			return "INTEGER"
		}
		return "INT"
	case FieldTypeFloat:
		if tx.driver == "sqlite3" {
			return "REAL"
		}
		return "FLOAT"
	case FieldTypeBool:
		if tx.driver == "sqlite3" {
			return "INTEGER"
		}
		return "BOOLEAN"
	case FieldTypeDate:
		if tx.driver == "sqlite3" {
			return "TEXT"
		}
		return "DATETIME"
	case FieldTypeJSON:
		if tx.driver == "mysql" {
			return "JSON"
		}
		return "TEXT"
	default:
		if tx.driver == "sqlite3" {
			return "TEXT"
		}
		return "VARCHAR(255)"
	}
}

// formatDefaultValue 格式化默认值 (事务版本)
func (tx *SQLTransaction) formatDefaultValue(value any) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case bool:
		if v {
			return "1"
		}
		return "0"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// buildCreateIndexSQL 构建创建索引的 SQL 语句 (事务版本)
func (tx *SQLTransaction) buildCreateIndexSQL(table string, index IndexDefinition) string {
	indexType := "INDEX"
	if index.Unique {
		indexType = "UNIQUE INDEX"
	}

	// MySQL 不支持 IF NOT EXISTS 语法用于索引
	if tx.driver == "mysql" {
		return fmt.Sprintf("CREATE %s %s ON %s (%s)",
			indexType, index.Name, table, strings.Join(index.Fields, ", "))
	}

	return fmt.Sprintf("CREATE %s IF NOT EXISTS %s ON %s (%s)",
		indexType, index.Name, table, strings.Join(index.Fields, ", "))
}

func (tx *SQLTransaction) scanRowToRecord(rows *sql.Rows) (Record, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := rows.Scan(valuePtrs...); err != nil {
		return nil, err
	}

	data := make(map[string]any)
	for i, col := range columns {
		data[col] = values[i]
	}

	return &SQLRecord{data: data}, nil
}
