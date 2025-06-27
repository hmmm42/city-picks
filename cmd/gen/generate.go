package main

import (
	"github.com/hmmm42/city-picks/internal/adapter/persistent"
	"github.com/hmmm42/city-picks/internal/config"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

//func init() {
//	defaultConfigPath := config.GetDefaultConfigPath()
//	configPath := pflag.StringP("config", "c", defaultConfigPath, "path to config file")
//	pflag.Parse()
//
//	config.InitConfig(*configPath)
//}

func main() {
	_, _ = config.NewOptions()
	mySQL, err := persistent.NewMySQL(config.MySQLOptions)
	if err != nil {
		panic("Failed to connect to MySQL: " + err.Error())
	}

	g := gen.NewGenerator(gen.Config{
		OutPath:           "./dal/query",
		Mode:              gen.WithQueryInterface | gen.WithoutContext | gen.WithDefaultQuery,
		FieldNullable:     false,
		FieldCoverable:    false,
		FieldSignable:     true,
		FieldWithIndexTag: false,
		FieldWithTypeTag:  true,
	})
	g.UseDB(mySQL)
	dataMap := map[string]func(columnType gorm.ColumnType) (dataType string){
		"tinyint":  func(columnType gorm.ColumnType) (dataType string) { return "int8" },
		"smallint": func(columnType gorm.ColumnType) (dataType string) { return "int16" },
		"bigint":   func(columnType gorm.ColumnType) (dataType string) { return "int64" },
		"int":      func(columnType gorm.ColumnType) (dataType string) { return "int64" },
	}

	g.WithDataTypeMap(dataMap)

	autoCreateTimeField := gen.FieldGORMTag("created_on", func(tag field.GormTag) field.GormTag {
		tag.Set("autoCreateTime", "")
		return tag
	})
	autoUpdateTimeField := gen.FieldGORMTag("modified_on", func(tag field.GormTag) field.GormTag {
		tag.Set("autoUpdateTime", "")
		return tag
	})
	fieldOpts := []gen.ModelOpt{
		autoCreateTimeField,
		autoUpdateTimeField,
	}

	allModel := g.GenerateAllTable(fieldOpts...)
	g.ApplyBasic(allModel...)
	g.Execute()
}
