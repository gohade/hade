package gorm

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"

	_ "github.com/jinzhu/gorm/dialects/mssql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/pkg/errors"
)

func NewGormDB(params ...interface{}) (interface{}, error) {
	c := params[0].(map[string]string)
	if c == nil {
		return nil, errors.New("config is empty")
	}
	s := fmt.Sprintf("%s:%s@(%s)/%s?charset=%s&parseTime=True&loc=Local", c["username"], c["password"], c["hostname"], c["database"], c["charset"])
	db, err := gorm.Open(c["driver"], s)
	if err != nil {
		return nil, errors.Wrap(err, "new gorm error")
	}
	return db, nil
}
