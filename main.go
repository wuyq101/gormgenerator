package main

import (
	"bytes"
	"database/sql"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"strings"
	"unicode"

	_ "github.com/go-sql-driver/mysql"
)

var (
	dsn   string
	table string
)

func main() {
	flag.StringVar(&dsn, "dsn", "", "mysql 连接串")
	flag.StringVar(&table, "table", "", "table name")
	flag.Parse()
	if len(dsn) == 0 || len(table) == 0 {
		usage()
		os.Exit(-1)
	}
	//start to connect to db
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	result, err := Generate(db, table)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Generate go strut for table %s:\n\n%s\n", table, result)
}
func usage() {
	s := `-dsn mysql连接字符串: username:password@protocol(address)/dbname?param=value
-table table name`
	fmt.Println(s)
}

var structTmpl = `type {{.StructName}} struct { {{range $item := .Columns}}
	{{$item.FieldName}}	{{$item.FieldType}}	` + "`" + `gorm:"column:{{$item.ColumnName}}"` + "`" + ` {{end}}
}`

// Generate 根据表面生成对应的结构
func Generate(db *sql.DB, table string) (string, error) {
	rows, err := db.Query("show create table " + table + " ")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	var t, s string
	data := map[string]interface{}{
		"StructName": camelCase(table),
	}
	for rows.Next() {
		err = rows.Scan(&t, &s)
		if err != nil {
			return "", err
		}
		data["Columns"] = parseTable(s)
	}
	tmpl, err := template.New("t").Parse(structTmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	return buf.String(), nil
}

func parseTable(s string) []map[string]interface{} {
	columns := make([]map[string]interface{}, 0)
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		line = strings.Trim(line, " ")
		if strings.HasPrefix(line, "`") {
			p := strings.Split(line, " ")
			name := strings.Trim(p[0], "`")
			dataType := p[1]
			columns = append(columns, map[string]interface{}{
				"FieldName":  camelCase(name),
				"FieldType":  fieldType(dataType),
				"ColumnName": name,
			})
		}
	}
	return columns
}

func fieldType(dataType string) string {
	if strings.HasPrefix(dataType, "bigint") {
		return "int64"
	}
	if strings.HasPrefix(dataType, "int") || strings.HasPrefix(dataType, "smallint") || strings.HasPrefix(dataType, "tinyint") ||
		strings.HasPrefix(dataType, "mediumint") {
		return "int"
	}
	if strings.HasPrefix(dataType, "decimal") || strings.HasPrefix(dataType, "numeric") ||
		strings.HasPrefix(dataType, "float") || strings.HasPrefix(dataType, "double") {
		return "float64"
	}
	if strings.HasPrefix(dataType, "datetime") || strings.HasPrefix(dataType, "date") || strings.HasPrefix(dataType, "timestamp") {
		return "time.Time"
	}
	if strings.HasPrefix(dataType, "varchar") || strings.HasPrefix(dataType, "char") ||
		strings.HasPrefix(dataType, "text") {
		return "string"
	}
	return "unknown"
}

// camelCase user_profile ---> UserProfile
func camelCase(s string) string {
	var buf bytes.Buffer
	var flag = false
	for i, c := range s {
		if c == '_' {
			flag = true
			continue
		}
		if i == 0 {
			buf.WriteRune(unicode.ToUpper(c))
			continue
		}
		if flag {
			buf.WriteRune(unicode.ToUpper(c))
			flag = false
			continue
		}
		buf.WriteRune(c)
	}
	return buf.String()
}
