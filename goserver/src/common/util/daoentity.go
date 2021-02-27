package util

import (
	"common/tlog"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

/*
example:
	type ItemModel struct {
		Id       int64  `db:"auto_increment"`
		Name     string
		Suffix   string `db:"null"`
	}
	type ItemModel struct {
		Id       int64  `db:"primary_key"`
		Name     string
		Suffix   string `db:"alter_suffix,null"`
	}

	itemEntity := util.NewDaoEntity("item_table", &ItemModel{})

	// Insert
	itemEntity.Dao(ThisServer.Mysql).Create(&ItemModel{Id:2})

	// Select
	var item ItemModel
	itemEntity.Dao(ThisServer.Mysql).Where("name=?", "zhangsan").Find(&items)
	itemEntity.Dao(ThisServer.Mysql).Match(31).Find(&item) // primary_key = 31
	itemEntity.Dao(ThisServer.Mysql).First(&item) // order by primary_key ASC
	itemEntity.Dao(ThisServer.Mysql).Last(&item)  // order by primary_key DESC

	var items []ItemModel
	var items []*ItemModel
	itemEntity.Dao(ThisServer.Mysql).Where("id IN(?)", []int{30, 31}).Find(&items)

	// Count
	count, err := itemEntity.Dao(ThisServer.Mysql).Where("id BETWEEN ? AND ?", 1, 34).Distinct("name").Count()

	// Update
	itemEntity.Dao(ThisServer.Mysql).Match(40).Update("name", "zhangsan")
	itemEntity.Dao(ThisServer.Mysql).Updates(&item)
	itemEntity.Dao(ThisServer.Mysql).Updates(map[string]interface{}{"id": 31, "suffix": "ddd"})
*/

const (
	InsertSqlTemplate = "INSERT INTO %s (%s) VALUES(%s)"

	DaoTagPrimaryKey    = "primary_key"
	DaoTagAutoIncrement = "auto_increment"
	DaoTagNull          = "null"

	DaoPrimaryKeyId = "id"

	DaoFlagNullString = 0x40000000
	DaoFlagMask       = 0x0FFFFFFF
)

var ErrorMissingWhereClause = errors.New("missing where clause")

type daoEntityInternal struct {
	tableName           string
	tableColumns        []string
	field2Column        map[string]string
	autoIncrementColumn string
	primaryKeyColumn    string
	nullColumns         map[string]bool
}

type DaoEntity struct {
	internal       *daoEntityInternal
	db             *sql.DB
	tx             *sql.Tx
	limit          int
	orderBy        string
	distinctColumn string
	selectColumns  []string
	omitColumns    []string
	querys         []string
	queryArgs      []interface{}
	orQuerys       []string
	orQueryArgs    []interface{}
}

func NewDaoEntity(tableName string, entity interface{}) *DaoEntity {
	ptr := reflect.TypeOf(entity)
	for ptr.Kind() != reflect.Struct {
		if ptr.Kind() == reflect.Ptr {
			ptr = ptr.Elem()
		}
		if ptr.Kind() == reflect.Interface {
			ptr = reflect.TypeOf(ptr)
		}
	}

	fieldCount := ptr.NumField()
	tableColumns := make([]string, 0, fieldCount)
	field2Column := make(map[string]string, fieldCount)
	nullColumns := make(map[string]bool)
	var autoIncrementColumn string
	var primaryKeyColumn string
	for k := 0; k < fieldCount; k++ {
		field := ptr.Field(k)
		fieldName := field.Name
		tagName := field.Tag.Get("db")
		var columnName string

		if tagName != "" {
			if tagName != "-" {
				ss := strings.Split(tagName, ",")
				hasPrimary := false
				hasAutoIncrement := false
				hasNull := false
				for i := 0; i < len(ss); i++ {
					switch ss[i] {
					case DaoTagPrimaryKey:
						hasPrimary = true
					case DaoTagAutoIncrement:
						hasAutoIncrement = true
					case DaoTagNull:
						hasNull = true
					default:
						if i == 0 {
							columnName = ss[0]
						}
					}
				}
				if columnName == "" {
					columnName = Camel2Case(fieldName)
				}

				if hasAutoIncrement {
					autoIncrementColumn = columnName
					primaryKeyColumn = columnName
				}
				if hasPrimary {
					primaryKeyColumn = columnName
				}
				if hasNull {
					nullColumns[columnName] = true
				}
			}
		} else {
			columnName = Camel2Case(fieldName)
		}

		if columnName != "" {
			tableColumns = append(tableColumns, columnName)
			field2Column[fieldName] = columnName
		}
	}

	if primaryKeyColumn == "" {
		for _, v := range tableColumns {
			if v == DaoPrimaryKeyId {
				primaryKeyColumn = DaoPrimaryKeyId
				break
			}
		}
	}

	return &DaoEntity{internal: &daoEntityInternal{tableName: tableName, tableColumns: tableColumns,
		field2Column: field2Column, autoIncrementColumn: autoIncrementColumn,
		primaryKeyColumn: primaryKeyColumn,
		nullColumns:      nullColumns},
	}
}

func (this *DaoEntity) Dao(dbOrTx interface{}) *DaoEntity {
	if db, ok := dbOrTx.(*sql.DB); ok {
		return &DaoEntity{internal: this.internal, db: db}
	} else if tx, ok := dbOrTx.(*sql.Tx); ok {
		return &DaoEntity{internal: this.internal, tx: tx}
	}
	return nil
}

func (this *DaoEntity) Select(columns ...string) *DaoEntity {
	this.selectColumns = append(this.selectColumns, columns...)
	return this
}

func (this *DaoEntity) Omit(columns ...string) *DaoEntity {
	this.omitColumns = append(this.omitColumns, columns...)
	return this
}

func (this *DaoEntity) Limit(limit int) *DaoEntity {
	this.limit = limit
	return this
}

func (this *DaoEntity) Order(ord string) *DaoEntity {
	this.orderBy = ord
	return this
}

func (this *DaoEntity) Distinct(column string) *DaoEntity {
	this.distinctColumn = column
	return this
}

func (this *DaoEntity) Where(query string, args ...interface{}) *DaoEntity {
	q, a := convertArgs(query, args...)
	this.querys = append(this.querys, q)
	this.queryArgs = append(this.queryArgs, a...)
	return this
}

func (this *DaoEntity) Match(arg interface{}) *DaoEntity {
	if this.internal.primaryKeyColumn != "" {
		this.Where(fmt.Sprintf("%s=?", this.internal.primaryKeyColumn), arg)
	}
	return this
}

func (this *DaoEntity) Or(query string, args ...interface{}) *DaoEntity {
	q, a := convertArgs(query, args...)
	this.orQuerys = append(this.orQuerys, q)
	this.orQueryArgs = append(this.orQueryArgs, a...)
	return this
}

func (this *DaoEntity) Create(entity interface{}) error {
	vals := this.convertToValues(entity)
	count := len(vals)

	bAutoIncrement := false
	bFirst := true
	var templ strings.Builder
	var templ2 strings.Builder
	tvals := make([]interface{}, 0, count)

	for i := 0; i < count; i++ {
		if i == 0 && vals[i].column == this.internal.autoIncrementColumn {
			bAutoIncrement = true
			continue
		}
		if bFirst {
			bFirst = false
		} else {
			templ.WriteString(",")
			templ2.WriteString(",")
		}
		templ.WriteString(vals[i].column)
		templ2.WriteString("?")
		tvals = append(tvals, vals[i].val)
	}
	if len(tvals) > 0 {
		ssql := fmt.Sprintf(InsertSqlTemplate, this.internal.tableName, templ.String(), templ2.String())
		tlog.Debug(ssql)
		var insertId int64
		var err error
		if this.db != nil {
			insertId, err = MysqlExecAndGetInsertId(this.db, ssql, tvals...)
		} else {
			insertId, err = TxExecAndGetInsertId(this.tx, ssql, tvals...)
		}
		if err != nil {
			return err
		}
		if bAutoIncrement {
			val := reflect.ValueOf(entity)
			if val.Kind() == reflect.Ptr {
				val.Elem().Field(0).SetInt(insertId)
			}
		}
		return nil
	} else {
		return errors.New("db create emtpy")
	}
}

func (this *DaoEntity) Update(column string, val interface{}) (int, error) {
	whereClause := this.getWhereClause()
	if whereClause == "" {
		return 0, ErrorMissingWhereClause
	}

	var b strings.Builder
	b.WriteString("UPDATE ")
	b.WriteString(this.internal.tableName)
	b.WriteString(" SET ")
	b.WriteString(column)
	b.WriteString("=?")
	b.WriteString(whereClause)

	ssql := b.String()
	tlog.Debug(ssql)
	vv := []interface{}{val}
	if this.db != nil {
		return MysqlExec(this.db, ssql, append(vv, append(this.queryArgs, this.orQueryArgs...)...)...)
	} else {
		return TxExec(this.tx, ssql, append(vv, append(this.queryArgs, this.orQueryArgs...)...)...)
	}
}

func (this *DaoEntity) Updates(entity interface{}) (int, error) {
	vals := this.convertToValues(entity)
	count := len(vals)

	var b strings.Builder
	b.WriteString("UPDATE ")
	b.WriteString(this.internal.tableName)
	b.WriteString(" SET ")

	bHasPrimaryKey := false
	bFirst := true
	tvals := make([]interface{}, 0, count)

	for i := 0; i < count; i++ {
		if i == 0 && vals[i].column == this.internal.primaryKeyColumn {
			bHasPrimaryKey = true
			continue
		}
		if bFirst {
			bFirst = false
		} else {
			b.WriteString(",")
		}
		b.WriteString(vals[i].column)
		b.WriteString("=?")
		tvals = append(tvals, vals[i].val)
	}

	if len(tvals) > 0 {
		if bHasPrimaryKey {
			b.WriteString(" WHERE ")
			b.WriteString(vals[0].column)
			b.WriteString("=?")

			ssql := b.String()
			tlog.Debug(ssql)
			if this.db != nil {
				return MysqlExec(this.db, ssql, append(tvals, vals[0].val)...)
			} else {
				return TxExec(this.tx, ssql, append(tvals, vals[0].val)...)
			}
		} else {
			whereClause := this.getWhereClause()
			if whereClause == "" {
				return 0, ErrorMissingWhereClause
			}
			b.WriteString(whereClause)

			ssql := b.String()
			tlog.Debug(ssql)
			if this.db != nil {
				return MysqlExec(this.db, ssql, append(tvals, append(this.queryArgs, this.orQueryArgs...)...)...)
			} else {
				return TxExec(this.tx, ssql, append(tvals, append(this.queryArgs, this.orQueryArgs...)...)...)
			}
		}
	} else {
		return 0, errors.New("db update emtpy")
	}
}

func (this *DaoEntity) Count() (int, error) {
	var b strings.Builder
	b.WriteString("SELECT ")
	if this.distinctColumn != "" {
		b.WriteString("COUNT(DISTINCT(")
		b.WriteString(this.distinctColumn)
		b.WriteString("))")
	} else {
		b.WriteString("COUNT(1)")
	}
	b.WriteString(" FROM ")
	b.WriteString(this.internal.tableName)

	b.WriteString(this.getWhereClause())

	ssql := b.String()
	tlog.Debug(ssql)
	var row *sql.Row
	if this.db != nil {
		row = this.db.QueryRow(ssql, append(this.queryArgs, this.orQueryArgs...)...)
	} else {
		row = this.tx.QueryRow(ssql, append(this.queryArgs, this.orQueryArgs...)...)
	}
	var count int
	err := row.Scan(&count)
	return count, err
}

func (this *DaoEntity) Find(out interface{}) error {
	for reflect.TypeOf(out).Kind() != reflect.Ptr {
		return errors.New("find: out must be pointer")
	}

	limit := this.limit
	outVal := reflect.ValueOf(out).Elem()
	arrayFlag := 0

	var elementType reflect.Type
	switch outVal.Type().Kind() {
	case reflect.Slice, reflect.Array:
		elementType = reflect.TypeOf(out).Elem().Elem()
		if elementType.Kind() == reflect.Ptr {
			elementType = elementType.Elem()
			arrayFlag = 2
		} else {
			arrayFlag = 1
		}
	default:
		elementType = outVal.Type()
		limit = 1
	}

	var b strings.Builder
	b.WriteString("SELECT ")
	if len(this.selectColumns) > 0 {
		b.WriteString(strings.Join(this.selectColumns, ","))
	} else {
		b.WriteString(strings.Join(this.internal.tableColumns, ","))
	}
	b.WriteString(" FROM ")
	b.WriteString(this.internal.tableName)

	b.WriteString(this.getWhereClause())

	if this.orderBy != "" {
		b.WriteString(" ORDER BY ")
		b.WriteString(this.orderBy)
	}
	if limit > 0 {
		b.WriteString(" LIMIT ")
		b.WriteString(strconv.FormatInt(int64(limit), 10))
	}

	ssql := b.String()
	tlog.Debug(ssql)
	var rows *sql.Rows
	var err error
	if this.db != nil {
		rows, err = this.db.Query(ssql, append(this.queryArgs, this.orQueryArgs...)...)
	} else {
		rows, err = this.tx.Query(ssql, append(this.queryArgs, this.orQueryArgs...)...)
	}
	if err != nil {
		tlog.Error(err)
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	colSize := len(columns)

	column2Index, valuePtrs := this.getScanColumns(columns, elementType)
	for rows.Next() {
		err = rows.Scan(valuePtrs...)
		if err == nil {
			if arrayFlag > 0 {
				rowValPtr := reflect.New(elementType)
				rowVal := rowValPtr.Elem()
				for i := 0; i < colSize; i++ {
					flag := column2Index[i]
					if flag >= 0 {
						setFieldValue(flag, rowVal, valuePtrs[i])
					}
				}
				if arrayFlag == 1 {
					outVal.Set(reflect.Append(outVal, rowVal))
				} else {
					outVal.Set(reflect.Append(outVal, rowValPtr))
				}
			} else {
				for i := 0; i < colSize; i++ {
					flag := column2Index[i]
					if flag >= 0 {
						setFieldValue(flag, outVal, valuePtrs[i])
					}
				}
				break
			}
		} else {
			tlog.Error(err)
		}
	}
	err = rows.Err()
	if err != nil {
		tlog.Error(err)
	}
	return err
}

func (this *DaoEntity) First(out interface{}) error {
	if this.internal.primaryKeyColumn == "" {
		return errors.New("No primary key")
	}
	this.Order(fmt.Sprintf("%s ASC", this.internal.primaryKeyColumn))
	this.Limit(1)
	return this.Find(out)
}

func (this *DaoEntity) Last(out interface{}) error {
	if this.internal.primaryKeyColumn == "" {
		return errors.New("No primary key")
	}
	this.Order(fmt.Sprintf("%s DESC", this.internal.primaryKeyColumn))
	this.Limit(1)
	return this.Find(out)
}

func (this *DaoEntity) getWhereClause() string {
	if len(this.querys) > 0 || len(this.orQuerys) > 0 {
		var b strings.Builder
		b.WriteString(" WHERE ")
		if len(this.querys) > 0 {
			b.WriteString(strings.Join(this.querys, " AND "))
			if len(this.orQuerys) > 0 {
				b.WriteString(" OR (")
				b.WriteString(strings.Join(this.orQuerys, " AND "))
				b.WriteString(")")
			}
		} else {
			b.WriteString(strings.Join(this.orQuerys, " AND "))
		}
		return b.String()
	}
	return ""
}

func setFieldValue(flag int, obj reflect.Value, val interface{}) {
	if (flag & DaoFlagNullString) != 0 {
		s := val.(*sql.NullString)
		obj.Field(flag & DaoFlagMask).SetString(s.String)
	} else {
		obj.Field(flag & DaoFlagMask).Set(reflect.ValueOf(val).Elem())
	}
}

func (this *DaoEntity) getScanColumns(columns []string, elementType reflect.Type) ([]int, []interface{}) {
	colSize := len(columns)
	fieldSize := elementType.NumField()
	column2Index := make([]int, colSize)
	valuePtrs := make([]interface{}, colSize)
	for i := 0; i < colSize; i++ {
		col := columns[i]
		column2Index[i] = -1
		for k := 0; k < fieldSize; k++ {
			fieldName := elementType.Field(k).Name
			if col == fieldName || col == this.internal.field2Column[fieldName] {
				flag := k
				ftype := elementType.Field(k).Type
				if ftype.Kind() == reflect.String && this.internal.nullColumns[col] {
					ftype = reflect.TypeOf(sql.NullString{})
					flag |= DaoFlagNullString
				}
				column2Index[i] = flag
				valuePtrs[i] = reflect.New(ftype).Interface()
				break
			}
		}
	}
	return column2Index, valuePtrs
}

func (this *DaoEntity) convertToValues(entity interface{}) []columnValue {
	val := reflect.ValueOf(entity)
	if val.Kind() == reflect.Map {
		vmap, ok := entity.(map[string]interface{})
		if ok {
			match := make([]columnValue, 0, len(vmap))
			primaryVal := columnValue{}
			for k, v := range vmap {
				if columnName, ok := this.convertToColumn(k); ok {
					if columnName == this.internal.primaryKeyColumn {
						primaryVal.column = columnName
						primaryVal.val = v
						continue
					}
					namedVal := columnValue{column: columnName, val: v}
					match = append(match, namedVal)
				}
			}
			if primaryVal.column == "" {
				return match
			} else {
				return append([]columnValue{primaryVal}, match...)
			}
		}
		return nil
	}

	for val.Kind() != reflect.Struct {
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		if val.Kind() == reflect.Interface {
			val = val.Elem()
		}
	}

	match := make([]columnValue, 0, val.NumField())
	for k := 0; k < val.NumField(); k++ {
		name := val.Type().Field(k).Name
		if columnName, ok := this.convertToColumn(name); ok {
			namedVal := columnValue{column: columnName, val: val.Field(k).Interface()}
			match = append(match, namedVal)
		}
	}
	return match
}

func (this *DaoEntity) convertToColumn(name string) (string, bool) {
	columnName := this.internal.field2Column[name]
	if columnName == "" {
		for _, c := range this.internal.tableColumns {
			if c == name {
				columnName = name
				break
			}
		}
	}
	if columnName != "" {
		bAllowed := true
		if len(this.selectColumns) > 0 {
			bAllowed = false
			for _, f := range this.selectColumns {
				if f == columnName {
					bAllowed = true
					break
				}
			}
		}
		if bAllowed && len(this.omitColumns) > 0 {
			for _, f := range this.omitColumns {
				if f == columnName {
					bAllowed = false
					break
				}
			}
		}
		return columnName, bAllowed
	}
	return "", false
}

func convertArgs(query string, args ...interface{}) (string, []interface{}) {
	if len(args) != 1 {
		return query, args
	}
	arg := args[0]
	switch reflect.TypeOf(arg).Kind() {
	case reflect.Slice, reflect.Array:
		val := reflect.ValueOf(arg)
		sz := val.Len()
		realArgs := make([]interface{}, sz)
		var b strings.Builder
		bFirst := true
		for i := 0; i < val.Len(); i++ {
			realArgs[i] = val.Index(i).Interface()
			if bFirst {
				bFirst = false
			} else {
				b.WriteByte(',')
			}
			b.WriteByte('?')
		}
		query = strings.ReplaceAll(query, "?", b.String())
		return query, realArgs
	default:
		return query, args
	}
}

func Camel2Case(name string) string {
	if name == "ID" {
		return "id"
	}

	b := strings.Builder{}
	for i, r := range name {
		if unicode.IsUpper(r) {
			if i != 0 {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

type columnValue struct {
	column string
	val    interface{}
}
