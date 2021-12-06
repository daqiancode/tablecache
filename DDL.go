package tablecache

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type FKAction string

const (
	FKEmpty    FKAction = ""
	FKCascade  FKAction = "CASCADE"
	FKNoAction FKAction = "NO ACTION"
	FKSetNull  FKAction = "SET NULL"
	FKRestrict FKAction = "RESTRICT"
)

type DDL struct {
	db              *gorm.DB
	cacheStore      sync.Map // struct Type : Schema
	DefaultOnDelete FKAction
	DefaultOnUpdate FKAction
}

func NewDDL(db *gorm.DB) *DDL {
	return &DDL{
		db:              db,
		DefaultOnDelete: FKCascade,
		DefaultOnUpdate: FKCascade,
	}
}

func (s *DDL) AddTables(tables ...interface{}) {
	for _, v := range tables {
		schema.Parse(v, &s.cacheStore, s.db.NamingStrategy)
	}
}
func (s *DDL) Range(f func(structType reflect.Type, tableSchema *schema.Schema) bool) {
	s.cacheStore.Range(func(key, value interface{}) bool {
		return f(key.(reflect.Type), value.(*schema.Schema))
	})
}
func (s *DDL) AddFK(table, target interface{}, fk string) {
	srcSch := s.GetSchema(table)
	dstSch := s.GetSchema(target)
	s.AddForeignKey(srcSch.Table, fk, dstSch.Table, dstSch.PrimaryFieldDBNames[0], FKRestrict, FKCascade)
}
func (s *DDL) MakeFKName(table, fkey, target, targetCol string) string {
	return fmt.Sprintf("fk_%s_%s", table, fkey)
}

func (s *DDL) AddForeignKey(table, fkey, target, targetCol string, onDelete, onUpdate FKAction) {
	fkName := s.MakeFKName(table, fkey, target, targetCol)
	args := []interface{}{table, fkName, fkey, target, targetCol}
	tpl := "ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)"
	if onUpdate != FKEmpty {
		tpl += " ON UPDATE %s"
		args = append(args, onUpdate)
	}
	if onDelete != FKEmpty {
		tpl += " ON DELETE %s"
		args = append(args, onDelete)
	}
	ddl := fmt.Sprintf(tpl, args...)
	tx := s.db.Exec(ddl)
	fmt.Println(tx.Error)
}

// func (s *DDL) GetTableName(tableStruct interface{}) string {
// 	stmt := &gorm.Statement{DB: s.db}
// 	stmt.Parse(tableStruct)
// 	return stmt.Schema.Table
// }
// func (s *DDL) GetTablePK(tableStruct interface{}) string {
// 	stmt := &gorm.Statement{DB: s.db}
// 	stmt.Parse(tableStruct)
// 	return stmt.Schema.PrimaryFieldDBNames[0]
// }

func (s *DDL) AddFKs(table interface{}) {
	// stmt := &gorm.Statement{DB: s.db}
	sch, err := schema.Parse(table, &s.cacheStore, s.db.NamingStrategy)
	fmt.Println(sch, err)
	for _, f := range sch.Fields {
		fmt.Println(f.TagSettings)
	}

}
func (s *DDL) MakeFKs() {
	s.Range(func(structType reflect.Type, src *schema.Schema) bool {
		for _, f := range src.Fields {
			// fmt.Println(f.TagSettings)
			if v, ok := f.TagSettings["FK"]; ok {
				fkInfo := s.ParseFKInfo(v)
				dst := s.GetSchemaByStructName(fkInfo.StructName)
				if dst == nil {
					info := fmt.Sprintf("Make FK error. %s.%s can not ref struct %s\n", src.Name, f.Name, fkInfo.StructName)
					fmt.Println(info)
					panic(errors.New(info))

				}
				if fkInfo.OnDelete == "" {
					fkInfo.OnDelete = s.DefaultOnDelete
				}
				if fkInfo.OnUpdate == "" {
					fkInfo.OnUpdate = s.DefaultOnUpdate
				}
				fmt.Printf("Make FK: %s.%s -> %s.%s on delete %s on update %s\n", src.Table, f.DBName, dst.Table, dst.PrimaryFieldDBNames[0], fkInfo.OnDelete, fkInfo.OnUpdate)
				s.AddForeignKey(src.Table, f.DBName, dst.Table, dst.PrimaryFieldDBNames[0], fkInfo.OnDelete, fkInfo.OnUpdate)
			}
		}
		return true
	})
}

func (s *DDL) MatchTableName(structType reflect.Type, tableName string) bool {
	return strings.EqualFold(tableName, structType.Name())
}

func (s *DDL) GetSchemaByStructName(structName string) *schema.Schema {
	var r *schema.Schema
	s.Range(func(structType reflect.Type, tableSchema *schema.Schema) bool {
		if s.MatchTableName(structType, structName) {
			r = tableSchema
			return false
		}
		return true
	})
	return r
}
func (s *DDL) GetSchema(obj interface{}) *schema.Schema {
	r, _ := s.cacheStore.Load(reflect.Indirect(reflect.ValueOf(obj)).Type())
	return r.(*schema.Schema)
}

type FKInfo struct {
	StructName string
	OnDelete   FKAction
	OnUpdate   FKAction
}

// tag: eg. FK:User,CASCADE,CASCADE  Table,on delete %s,on update %s
func (s *DDL) ParseFKInfo(tag string) FKInfo {
	parts := strings.Split(tag, ",")
	r := FKInfo{}
	r.StructName = strings.TrimSpace(parts[0])
	if len(parts) > 1 {
		r.OnDelete = FKAction(strings.TrimSpace(parts[1]))
	}
	if len(parts) > 2 {
		r.OnUpdate = FKAction(strings.TrimSpace(parts[2]))
	}
	return r

}
