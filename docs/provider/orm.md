# hade:orm

## 说明

hade:orm 是提供ORM服务的服务，可以用于获取数据库连接，获取表结构等。

## 使用方法

```
// ORMService 表示传入的参数
type ORMService interface {
	// GetDB  获取某个db
	GetDB(option ...DBOption) (*gorm.DB, error)

	// CanConnect 是否可以连接
	CanConnect(ctx context.Context, db *gorm.DB) (bool, error)

	// Table 相关
	GetTables(ctx context.Context, db *gorm.DB) ([]string, error)
	HasTable(ctx context.Context, db *gorm.DB, table string) (bool, error)
	GetTableColumns(ctx context.Context, db *gorm.DB, table string) ([]TableColumn, error)
}

```
