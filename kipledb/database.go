package kipledb

import (
	"errors"
	"fmt"
	"github.com/kataras/iris/v12"
	"github.com/m2c/kiplestar/commons/db_log"
	"reflect"
	"strings"
	"time"

	"github.com/m2c/kiplestar/kipledb/transaction"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	slog "github.com/m2c/kiplestar/commons/log"
	server_config "github.com/m2c/kiplestar/config"
)

type KipleDB struct {
	db   *gorm.DB
	name string //db name
}

func (slf *KipleDB) DB() *gorm.DB {
	return slf.db
}

// used to trace sql log between different services
func (slf *KipleDB) GetDBCtx(ctx iris.Context) *gorm.DB {
	l := db_log.NewDbLogger(ctx)
	slf.db.SetLogger(l)
	return slf.db
}

func (slf *KipleDB) Name() string {
	return slf.name
}

func (slf *KipleDB) StartDb(config server_config.DataBaseConfig) error {
	if slf.db != nil {
		return errors.New("Db already open")
	}
	slf.name = config.DbName
	driver := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=true&loc=Local",
		config.User,
		config.Pwd,
		config.Host,
		config.Port,
		config.DataBase)
	var err error
	slf.db, err = gorm.Open("mysql", driver)

	if err != nil {
		slog.Infof("conn Db  error %s", err)
		return err
	}
	slog.Infof("conn Db opened Host %s", config.Host)
	slf.db.DB().SetMaxIdleConns(config.MaxIdleCons)
	slf.db.DB().SetMaxOpenConns(config.MaxOpenCons)
	slf.db.DB().SetConnMaxLifetime(config.MaxLifeTime * time.Second)
	slf.db.SingularTable(true)
	if server_config.SC.SConfigure.Profile != "prod" {
		slf.db.LogMode(true)
	}

	slf.db.SetLogger(&slog.Slog)
	slf.db.Callback().Create().Before("gorm:create").Register("create", func(scope *gorm.Scope) {
		if scope.Value == nil {
			scope.SetColumn("update_time", time.Now())
			return
		}
		rtElem := reflect.TypeOf(scope.Value).Elem()
		rvElem := reflect.ValueOf(scope.Value).Elem()
		if _, ok := rtElem.FieldByName("CreateTime"); ok {
			if t, ok := rvElem.FieldByName("CreateTime").Interface().(time.Time); ok && t.IsZero() {
				scope.SetColumn("create_time", time.Now())
			}
		}
		if _, ok := rtElem.FieldByName("UpdateTime"); ok {
			if t, ok := rvElem.FieldByName("UpdateTime").Interface().(time.Time); ok && t.IsZero() {
				scope.SetColumn("update_time", time.Now())
			}
		}
		if _, ok := rtElem.FieldByName("CreatedAt"); ok {
			if t, ok := rvElem.FieldByName("CreatedAt").Interface().(time.Time); ok && t.IsZero() {
				scope.SetColumn("created_at", time.Now())
			}
		}
		if _, ok := rtElem.FieldByName("UpdatedAt"); ok {
			if t, ok := rvElem.FieldByName("UpdatedAt").Interface().(time.Time); ok && t.IsZero() {
				scope.SetColumn("updated_at", time.Now())
			}
		}
	})
	slf.db.Callback().Update().Before("gorm:update").Register("update", func(scope *gorm.Scope) {
		if scope.Value == nil {
			scope.SetColumn("update_time", time.Now())
			return
		}
		rtElem := reflect.TypeOf(scope.Value).Elem()
		rvElem := reflect.ValueOf(scope.Value).Elem()
		if rtElem.Kind() == reflect.Struct {
			if _, ok := rtElem.FieldByName("UpdateTime"); ok {
				if t, ok := rvElem.FieldByName("UpdateTime").Interface().(time.Time); ok && t.IsZero() {
					scope.SetColumn("update_time", time.Now())
				}
			}
			if _, ok := rtElem.FieldByName("UpdatedAt"); ok {
				if t, ok := rvElem.FieldByName("UpdatedAt").Interface().(time.Time); ok && t.IsZero() {
					scope.SetColumn("updated_at", time.Now())
				}
			}
		} else {
			scope.SetColumn("update_time", time.Now())
		}
	})

	return nil
}

func (slf *KipleDB) StopDb() error {
	if slf.db != nil {
		err := slf.db.Close()
		if err != nil {
			slf.db = nil
		}
		return err
	}
	return errors.New("Db is nil")
}

func (slf *KipleDB) Tx(f func(db *gorm.DB) error) error {
	tx := slf.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if tx.Error != nil {
		return tx.Error
	}
	if e := f(tx); e != nil {
		tx.Rollback()
		return e
	}
	if e1 := tx.Commit().Error; e1 != nil {
		return e1
	}
	return nil
}
func (slf *KipleDB) ExecuteSql(f func(db *gorm.DB) (interface{}, error)) (interface{}, error) {
	result, ok := f(slf.db)
	if ok != nil {
		return result, ok
	}
	return result, nil
}

/*
	type AutoGenerated struct {
		A   int    `json:"a"`
		B  int `json:"b"`
	}

	b := []interface{}{}
	for i:=0;i < 10000;i++{
		b = append(b,AutoGenerated{
			A: i,
			B: i,
		})
	}
	fmt.Println(BuildBulkInsertSql("test",[]string{"a","b"},b...))

*/
func (slf *KipleDB) BuildBulkInsertSql(tableName string, columns []string, values ...interface{}) (err error, sqlStr string) {
	//check parameter
	if tableName == "" {
		return errors.New("table name is empty"), ""
	}
	if len(columns) == 0 {
		return errors.New("columns is empty"), ""
	}
	//calc
	columnLen := len(columns)
	//insert into %s(%s) values%s;
	var sql strings.Builder
	sql.WriteString("insert into ")
	sql.WriteString(tableName)

	//build columns
	var strColumns strings.Builder
	strColumns.WriteString("(")
	for i, v := range columns {
		strColumns.WriteString(v)
		if i != columnLen-1 {
			strColumns.WriteString(",")
		}
	}
	strColumns.WriteString(")")
	sql.WriteString(strColumns.String())
	sql.WriteString("values")
	//build values
	valueLen := len(values)
	valueArray := make([]string, columnLen)
	for i, v := range values {
		valueBuffer := strings.Builder{}
		valueBuffer.WriteString("(")
		tp := reflect.TypeOf(v)
		if tp.Kind() != reflect.Struct {
			return fmt.Errorf("struct index %d is not struct", i), ""
		}
		ve := reflect.ValueOf(v)
		fieldNum := ve.NumField()

		for i := 0; i < fieldNum; i++ {
			field := ve.Field(i)
			columnIndex := -1
			//find columns index
			for ci, v := range columns {
				tag := tp.Field(i).Tag.Get("json")
				name := tp.Field(i).Name
				if v == tag || v == name {
					columnIndex = ci
				}
			}
			if columnIndex == -1 {
				continue
			}
			switch field.Kind() {
			case reflect.String:
				valueArray[columnIndex] = fmt.Sprintf("'%s'", field.Interface())
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				valueArray[columnIndex] = fmt.Sprintf("%d", field.Interface())
			case reflect.Float32, reflect.Float64:
				valueArray[columnIndex] = fmt.Sprintf("%f", field.Interface())
			default:
				return fmt.Errorf("type %s not support", field.Kind().String()), ""
			}
		}
		for i, v := range valueArray {
			valueBuffer.WriteString(v)
			if i != columnLen-1 {
				valueBuffer.WriteString(",")
			}
		}
		valueBuffer.WriteString(")")
		if i != valueLen-1 {
			valueBuffer.WriteString(",")
		}

		sql.WriteString(valueBuffer.String())
	}
	sql.WriteString(";")
	return nil, sql.String()
}

func (slf *KipleDB) NewTransaction() *transaction.TxUnits {
	return transaction.NewTxUnits(slf.db)
}
