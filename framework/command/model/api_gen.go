package model

import (
    "context"
    "github.com/dave/jennifer/jen"
    "github.com/gohade/hade/framework/contract"
    "github.com/gohade/hade/framework/util/word"
    "github.com/pkg/errors"
    "os"
    "strings"
)

type ApiGenerator struct {
	table       string
	columns     []contract.TableColumn
	packageName string // 包名
}

func NewApiGenerator(table string, columns []contract.TableColumn) *ApiGenerator {
	return &ApiGenerator{
		table:   table,
		columns: columns,
	}
}

func (gen *ApiGenerator) SetPackageName(packageName string) {
	gen.packageName = packageName
}

func (gen *ApiGenerator) GenModelFile(ctx context.Context, file string) error {
	// table lower case
	tableLower := strings.ToLower(gen.table)
	// table camel title case
	tableCamel := strings.Title(tableLower)
	// model struct
	tableModel := tableCamel + "Model"

	f := jen.NewFile(gen.packageName)

	structs := make([]jen.Code, 0, len(gen.columns)+1)
	for _, column := range gen.columns {
		field := jen.Id(word.ToTitleCamel(column.Field))
		switch column.Type {
		case "int", "tinyint", "smallint", "mediumint", "bigint":
			field.Int64()
		case "int unsigned", "tinyint unsigned", "smallint unsigned", "mediumint unsigned", "bigint unsigned":
			field.Uint64()
		case "float", "double", "decimal":
			field.Float64()
		case "float unsigned", "double unsigned", "decimal unsigned":
			field.Float64()
		case "char", "varchar", "tinytext", "text", "mediumtext", "longtext":
			field.String()
		case "date", "time", "datetime", "timestamp":
			field.Qual("time", "Time")
		default:
			field.String()
		}
		field.Tag(map[string]string{"gorm": column.Field, "json": column.Field + ",omitempty"})
		structs = append(structs, field)
	}

	f.Type().Id(tableModel).Struct(structs...)
	f.Line()
	f.Func().Params(jen.Id(tableModel)).Id("TableName").Params().String().Block(
		jen.Return(jen.Lit(gen.table)),
	)

	fp, err := os.Create(file)
	if err != nil {
		return errors.Wrap(err, "create file error")
	}
	if err := f.Render(fp); err != nil {
		return errors.Wrap(err, "render file error")
	}
	return nil
}

func (gen *ApiGenerator) GenRouterFile(ctx context.Context, file string) error {
	// table lower case
	tableLower := strings.ToLower(gen.table)
	// table camel title case
	tableCamel := strings.Title(tableLower)
	// Api struct
	tableApi := tableCamel + "Api"

	f := jen.NewFile(gen.packageName)
	// define struct type
	f.Type().Id(tableApi).Struct()

	// define NewApi() function
	f.Func().Id("New" + tableApi).Params().Op("*").Id(tableApi).Block(
		jen.Return(jen.Op("&").Id(tableApi).Values()),
	)

	// define Register() function
	f.Func().Id(tableApi+"Register").Params(jen.Id("r").Op("*").Qual("github.com/gohade/hade/framework/gin",
		"Engine")).Error().Block(
		jen.Id("api").Op(":=").Id("New"+tableApi).Call(),
		jen.Id("r").Dot("GET").Call(
			jen.Lit("/"+tableLower+"/show"), jen.Id("api").Dot("Show"),
		),
		jen.Id("r").Dot("GET").Call(
			jen.Lit("/"+tableLower+"/list"), jen.Id("api").Dot("List"),
		),
		jen.Id("r").Dot("POST").Call(
			jen.Lit("/"+tableLower+"/create"), jen.Id("api").Dot("Create"),
		),
		jen.Id("r").Dot("POST").Call(
			jen.Lit("/"+tableLower+"/update"), jen.Id("api").Dot("Update"),
		),
		jen.Id("r").Dot("POST").Call(
			jen.Lit("/"+tableLower+"/delete"), jen.Id("api").Dot("Delete"),
		),
		jen.Return(jen.Nil()),
	)

	fp, err := os.Create(file)
	if err != nil {
		return errors.Wrap(err, "create file error")
	}
	if err := f.Render(fp); err != nil {
		return errors.Wrap(err, "render file error")
	}

	return nil
}

func (gen *ApiGenerator) GenApiCreateFile(ctx context.Context, file string) error {
	// table lower case
	table := gen.table
	tableLower := strings.ToLower(gen.table)
	// table camel title case
	tableCamel := strings.Title(tableLower)
	// Api struct
	tableApi := tableCamel + "Api"
	tableModel := tableCamel + "Model"

	f := jen.NewFile(gen.packageName)

	f.Func().Params(
		jen.Id("api").Op("*").Id(tableApi),
	).Id("Create").Params(
		jen.Id("c").Op("*").Qual("github.com/gohade/hade/framework/gin", "Context"),
	).Block(
		jen.Id("logger").Op(":=").Id("c").Dot("MustMakeLog").Call(),

		jen.Var().Id(table).Qual("", tableModel),

		jen.If(
			jen.Err().Op(":=").Id("c").Dot("BindJSON").Call(jen.Op("&").Id(table)),
			jen.Err().Op("!=").Nil(),
		).Block(
			jen.Id("c").Dot("JSON").Call(jen.Lit(400), jen.Op("&").Qual("github.com/gohade/hade/framework/gin", "H").Values(jen.Dict{
				jen.Lit("code"): jen.Lit("Invalid parameter"),
			})),
			jen.Return(),
		),

		jen.Var().Id("db").Op("*").Qual("gorm.io/gorm", "DB"),
		jen.Var().Err().Error(),

		// gormService := c.MustMake(contract.ORMKey).(contract.ORMService)
		jen.Id("gormService").Op(":=").Id("c").Dot("MustMake").Call(jen.Qual("github."+
			"com/gohade/hade/framework/contract", "ORMKey")).Assert(jen.Qual("github.com/gohade/hade/framework/contract", "ORMService")),
		// db, err := gormService.GetDB(orm.WithConfigPath("database.default"))
		jen.Id("db").Op(",").Id("err").Op("=").Id("gormService").Dot("GetDB").Call(jen.Qual("github.com/gohade/hade/framework/provider/orm", "WithConfigPath").Call(jen.Lit("database.default"))),
		jen.If(jen.Err().Op("!=").Nil().Block(
			jen.Id("logger").Dot("Error").Call(jen.Id("c"), jen.Err().Dot("Error").Call(), jen.Nil()),
			jen.Id("_").Op("=").Id("c").Dot("AbortWithError").Call(jen.Lit(50001), jen.Err()),
			jen.Return(),
		)),
		jen.If(
			jen.Err().Op(":=").Id("db").Dot("Create").Call(jen.Op("&").Id(table)).Dot("Error"),
			jen.Err().Op("!=").Nil(),
		).Block(
			jen.Id("c").Dot("JSON").Call(jen.Lit(500), jen.Op("&").Qual("github.com/gohade/hade/framework/gin", "H").Values(jen.Dict{
				jen.Lit("error"): jen.Lit("Server error"),
			})),
			jen.Return(),
		),

		jen.Id("c").Dot("JSON").Call(jen.Lit(200), jen.Op("&").Id(table)),
	)

	fp, err := os.Create(file)
	if err != nil {
		return errors.Wrap(err, "create file error")
	}
	if err := f.Render(fp); err != nil {
		return errors.Wrap(err, "render file error")
	}
	return nil
}

func (gen *ApiGenerator) GenApiDeleteFile(ctx context.Context, file string) error {
	// table lower case
	tableLower := strings.ToLower(gen.table)
	// table camel title case
	tableCamel := strings.Title(tableLower)
	// Api struct
	tableApi := tableCamel + "Api"
	tableModel := tableCamel + "Model"

	f := jen.NewFile(gen.packageName)

	f.Func().Params(
		jen.Id("api").Op("*").Id(tableApi),
	).Id("Delete").Params(
		jen.Id("c").Op("*").Qual("github.com/gohade/hade/framework/gin", "Context"),
	).Block(
		jen.Id("id").Op(",").Id("err").Op(":=").Qual("strconv", "Atoi").Call(jen.Id("c").Dot("Query").Call(jen.Lit("id"))),
		jen.If(jen.Err().Op("!=").Nil().Block(
			jen.Id("c").Dot("JSON").Call(jen.Lit(400), jen.Op("&").Qual("github.com/gohade/hade/framework/gin", "H").Values(jen.Dict{
				jen.Lit("error"): jen.Lit("Invalid parameter"),
			})),
			jen.Return(),
		)),
		jen.Id("logger").Op(":=").Id("c").Dot("MustMakeLog").Call(),
		jen.Id("gormService").Op(":=").Id("c").Dot("MustMake").Call(jen.Qual("github.com/gohade/hade/framework/contract", "ORMKey")).Assert(jen.Qual("github.com/gohade/hade/framework/contract", "ORMService")),
		jen.Id("db").Op(",").Id("err").Op(":=").Id("gormService").Dot("GetDB").Call(jen.Qual("github."+
			"com/gohade/hade/framework/provider/orm", "WithConfigPath").Call(jen.Lit("database.default"))),
		jen.If(jen.Err().Op("!=").Nil().Block(
			jen.Id("logger").Dot("Error").Call(jen.Id("c"), jen.Err().Dot("Error").Call(), jen.Nil()),
			jen.Id("_").Op("=").Id("c").Dot("AbortWithError").Call(jen.Lit(50001), jen.Err()),
			jen.Return(),
		)),
		jen.If(
			jen.Err().Op(":=").Id("db").Dot("Delete").Call(jen.Op("&").Id(tableModel).Block(), jen.Id("id")).Dot("Error"),
			jen.Err().Op("!=").Nil(),
		).Block(
			jen.If(
				jen.Err().Op("==").Qual("gorm.io/gorm", "ErrRecordNotFound"),
			).Block(
				jen.Id("c").Dot("JSON").Call(jen.Lit(404), jen.Op("&").Qual("github.com/gohade/hade/framework/gin", "H").Values(jen.Dict{
					jen.Lit("error"): jen.Lit("Record not found"),
				})),
				jen.Return(),
			).Else().Block(
				jen.Id("c").Dot("JSON").Call(jen.Lit(500), jen.Op("&").Qual("github.com/gohade/hade/framework/gin", "H").Values(jen.Dict{
					jen.Lit("error"): jen.Lit("Server error"),
				})),
				jen.Return(),
			),
		),

		jen.Id("c").Dot("Status").Call(jen.Lit(200)),
	)

	// print generated code
	fp, err := os.Create(file)
	if err != nil {
		return errors.Wrap(err, "create file error")
	}
	if err := f.Render(fp); err != nil {
		return errors.Wrap(err, "render file error")
	}
	return nil
}

func (gen *ApiGenerator) GenApiListFile(ctx context.Context, file string) error {
	// table lower case
	tableLower := strings.ToLower(gen.table)
	// table camel title case
	tableCamel := strings.Title(tableLower)
	// Api struct
	tableApi := tableCamel + "Api"
	tableModel := tableCamel + "Model"

	f := jen.NewFile(gen.packageName)

	f.Func().Params(jen.Id("api").Op("*").Id(tableApi)).Id("List").Params(jen.Id("c").Op("*").Qual("github.com/gohade/hade/framework/gin", "Context")).Block(
		jen.List(jen.Id("offset"), jen.Id("err")).Op(":=").Qual("strconv",
			"Atoi").Call(jen.Id("c").Dot("Query").Call(jen.Lit("offset"))),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Id("c.JSON(400, gin.H{\"error\": \"Invalid parameter\"})"),
			jen.Return(),
		),
		jen.List(jen.Id("size"), jen.Id("err")).Op(":=").Qual("strconv",
			"Atoi").Call(jen.Id("c").Dot("Query").Call(jen.Lit("size"))),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Id("c.JSON(400, gin.H{\"error\": \"Invalid parameter\"})"),
			jen.Return(),
		),
		jen.Id("logger").Op(":=").Id("c").Dot("MustMakeLog").Call(),
		jen.Id("gormService").Op(":=").Id("c").Dot("MustMake").Call(jen.Qual("github.com/gohade/hade/framework/contract", "ORMKey")).Assert(jen.Qual("github.com/gohade/hade/framework/contract", "ORMService")),
		jen.Id("db").Op(",").Id("err").Op(":=").Id("gormService").Dot("GetDB").Call(jen.Qual("github."+
			"com/gohade/hade/framework/provider/orm", "WithConfigPath").Call(jen.Lit("database.default"))),
		jen.If(jen.Err().Op("!=").Nil().Block(
			jen.Id("logger").Dot("Error").Call(jen.Id("c"), jen.Err().Dot("Error").Call(), jen.Nil()),
			jen.Id("_").Op("=").Id("c").Dot("AbortWithError").Call(jen.Lit(50001), jen.Err()),
			jen.Return(),
		)),
		jen.Id("var total int64"),
		jen.If(jen.Err().Op(":=").Id("db").Dot("Model").Call(jen.Op("&").Id(tableModel).Block()).Dot("Count").Call(jen.Op("&").Id("total")).Dot("Error"),
			jen.Err().Op("!=").Nil(),
		).Block(
			jen.Id("c.JSON(500, gin.H{\"error\": \"Server error\"})"),
			jen.Return(),
		),
		jen.Id("var "+tableLower+"s []"+tableModel),
		jen.If(jen.Err().Op(":=").Id("db").Dot("Offset").Call(jen.Id("offset")).Dot("Limit").Call(jen.Id("size")).Dot("Find").Call(jen.Op("&").Id(tableLower+"s")).Dot("Error"),
			jen.Err().Op("!=").Nil(),
		).Block(
			jen.Id("c.JSON(500, gin.H{\"error\": \"Server error\"})"),
			jen.Return(),
		),

		jen.Comment("Return result"),
		jen.Id("c.JSON(200, gin.H{\"total\": total, \"data\": "+tableLower+"s})"),
	)

	// print generated code
	fp, err := os.Create(file)
	if err != nil {
		return errors.Wrap(err, "create file error")
	}
	if err := f.Render(fp); err != nil {
		return errors.Wrap(err, "render file error")
	}
	return nil
}

func (gen *ApiGenerator) GenApiShowFile(ctx context.Context, file string) error {
	// table lower case
	tableLower := strings.ToLower(gen.table)
	// table camel title case
	tableCamel := strings.Title(tableLower)
	// Api struct
	tableApi := tableCamel + "Api"
	tableModel := tableCamel + "Model"

	f := jen.NewFile(gen.packageName)

	f.Func().Params(jen.Id("api").Op("*").Id(tableApi)).Id("Show").Params(jen.Id("c").Op("*").Qual("github.com/gohade/hade/framework/gin", "Context")).Block(

		jen.List(jen.Id("id"), jen.Id("err")).Op(":=").Qual("strconv", "Atoi").Call(jen.Id("c").Dot("Query").Call(jen.Lit("id"))),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Id("c.JSON(400, gin.H{\"error\": \"Invalid parameter\"})"),
			jen.Return(),
		),

		jen.Id("logger").Op(":=").Id("c").Dot("MustMakeLog").Call(),
		jen.Id("gormService").Op(":=").Id("c").Dot("MustMake").Call(jen.Qual("github.com/gohade/hade/framework/contract", "ORMKey")).Assert(jen.Qual("github.com/gohade/hade/framework/contract", "ORMService")),
		jen.Id("db").Op(",").Id("err").Op(":=").Id("gormService").Dot("GetDB").Call(jen.Qual("github."+
			"com/gohade/hade/framework/provider/orm", "WithConfigPath").Call(jen.Lit("database.default"))),
		jen.If(jen.Err().Op("!=").Nil().Block(
			jen.Id("logger").Dot("Error").Call(jen.Id("c"), jen.Err().Dot("Error").Call(), jen.Nil()),
			jen.Id("_").Op("=").Id("c").Dot("AbortWithError").Call(jen.Lit(50001), jen.Err()),
			jen.Return(),
		)),

		// var student StudentModel
		jen.Id("var "+tableLower+" "+tableModel),
		jen.If(jen.Err().Op(":=").Id("db").Dot("First").Call(jen.Op("&").Id(tableLower),
			jen.Id("id")).Dot("Error"), jen.Err().Op("!=").Nil()).Block(
			jen.If(jen.Err().Op("==").Qual("gorm.io/gorm", "ErrRecordNotFound")).Block(
				jen.Id("c.JSON(404, gin.H{\"error\": \"Record not found\"})"),
				jen.Return(),
			).Else().Block(
				jen.Id("c.JSON(500, gin.H{\"error\": \"Server error\"})"),
				jen.Return(),
			),
		),

		jen.Id("c.JSON(200, "+tableLower+")"),
	)

	// print generated code
	fp, err := os.Create(file)
	if err != nil {
		return errors.Wrap(err, "create file error")
	}
	if err := f.Render(fp); err != nil {
		return errors.Wrap(err, "render file error")
	}
	return nil
}

func (gen *ApiGenerator) GenApiUpdateFile(ctx context.Context, file string) error {
	// table lower case
	tableLower := strings.ToLower(gen.table)
	// table camel title case
	tableCamel := strings.Title(tableLower)
	// Api struct
	tableApi := tableCamel + "Api"
	tableModel := tableCamel + "Model"

	f := jen.NewFile(gen.packageName)

	f.Func().Params(jen.Id("api").Op("*").Id(tableApi)).Id("Update").Params(jen.Id("c").Op("*").Qual("github.com/gohade/hade/framework/gin", "Context")).Block(
		jen.List(jen.Id("id"), jen.Id("err")).Op(":=").Qual("strconv", "Atoi").Call(jen.Id("c").Dot("Query").Call(jen.Lit("id"))),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Id("c.JSON(400, gin.H{\"error\": \"Invalid parameter\"})"),
			jen.Return(),
		),

		jen.Id("logger").Op(":=").Id("c").Dot("MustMakeLog").Call(),
		jen.Id("gormService").Op(":=").Id("c").Dot("MustMake").Call(jen.Qual("github.com/gohade/hade/framework/contract", "ORMKey")).Assert(jen.Qual("github.com/gohade/hade/framework/contract", "ORMService")),
		jen.Id("db").Op(",").Id("err").Op(":=").Id("gormService").Dot("GetDB").Call(jen.Qual("github."+
			"com/gohade/hade/framework/provider/orm", "WithConfigPath").Call(jen.Lit("database.default"))),
		jen.If(jen.Err().Op("!=").Nil().Block(
			jen.Id("logger").Dot("Error").Call(jen.Id("c"), jen.Err().Dot("Error").Call(), jen.Nil()),
			jen.Id("_").Op("=").Id("c").Dot("AbortWithError").Call(jen.Lit(50001), jen.Err()),
			jen.Return(),
		)),
		// var student StudentModel
		jen.Id("var "+tableLower+" "+tableModel),
		jen.If(jen.Err().Op(":=").Id("db").Dot("First").Call(jen.Op("&").Id(tableLower),
			jen.Id("id")).Dot("Error"), jen.Err().Op("!=").Nil()).Block(
			jen.If(jen.Err().Op("==").Qual("gorm.io/gorm", "ErrRecordNotFound")).Block(
				jen.Id("c.JSON(404, gin.H{\"error\": \"Record not found\"})"),
				jen.Return(),
			).Else().Block(
				jen.Id("c.JSON(500, gin.H{\"error\": \"Server error\"})"),
				jen.Return(),
			),
		),
		jen.Comment("Bind request body"),
		jen.If(jen.Err().Op(":=").Id("c").Dot("BindJSON").Call(jen.Op("&").Id(tableLower)),
			jen.Err().Op("!=").Nil()).Block(
			jen.Id("c.JSON(400, gin.H{\"error\": \"Invalid JSON\"})"),
			jen.Return(),
		),
		jen.If(jen.Err().Op(":=").Id("db").Dot("Save").Call(jen.Op("&").Id(tableLower)).Dot("Error"),
			jen.Err().Op("!=").Nil()).Block(
			jen.Id("c.JSON(500, gin.H{\"error\": \"Server error\"})"),
			jen.Return(),
		),
		// c.JSON(200, student)
		jen.Id("c.JSON(200, "+tableLower+")"),
	)

	// print generated code
	fp, err := os.Create(file)
	if err != nil {
		return errors.Wrap(err, "create file error")
	}
	if err := f.Render(fp); err != nil {
		return errors.Wrap(err, "render file error")
	}
	return nil
}
