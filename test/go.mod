module test

go 1.18

require (
	github.com/Bilibotter/light-flow-plugins/orm v0.0.0
	github.com/Bilibotter/light-flow/flow v0.0.1
	gorm.io/driver/mysql v1.5.7
	gorm.io/gorm v1.25.12
)

require (
	github.com/go-sql-driver/mysql v1.7.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace github.com/Bilibotter/light-flow-plugins/orm => ../orm
