package rdb

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/hatlonely/gox/rdb/aggregation"
	"github.com/hatlonely/gox/rdb/query"
	_ "github.com/go-sql-driver/mysql"
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

		fieldName := field.Name
		if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
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
		if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
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
	valueType := reflect.TypeOf(value)
	fieldType := fieldValue.Type()

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

// 实现 RDB 接口
func (s *SQL) Migrate(ctx context.Context, table string, model any) error {
	// 简单实现：假设表已存在，不做自动迁移
	return nil
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
	fields := record.Fields()
	
	var columns []string
	var placeholders []string
	var args []any
	
	for col, val := range fields {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		args = append(args, val)
	}
	
	sqlStr := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))
	
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
	fields := record.Fields()
	
	var columns []string
	var placeholders []string
	var args []any
	
	for col, val := range fields {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		args = append(args, val)
	}
	
	sqlStr := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))
	
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

func (tx *SQLTransaction) Migrate(ctx context.Context, table string, model any) error {
	return nil
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
