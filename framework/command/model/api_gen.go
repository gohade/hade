package model

import (
	"context"
	"github.com/dave/jennifer/jen"
	"github.com/gohade/hade/framework/contract"
	"github.com/pkg/errors"
	"os"
	"strings"
)

type ApiGenerator struct {
	table   string
	columns []contract.TableColumn
}

func NewApiGenerator(table string, columns []contract.TableColumn) *ApiGenerator {
	return &ApiGenerator{
		table:   table,
		columns: columns,
	}
}

func (gen *ApiGenerator) GenModelFile(ctx context.Context, file string) error {
	// table lower case
	tableLower := strings.ToLower(gen.table)
	// table camel title case
	tableCamel := strings.Title(tableLower)
	// model struct
	tableModel := tableCamel + "Model"

	f := jen.NewFile("gen")

	structs := make([]jen.Code, 0, len(gen.columns)+1)
	for _, column := range gen.columns {
		field := jen.Id(strings.Title(column.Field))
		switch column.Type {
		case "int", "tinyint", "smallint", "mediumint", "bigint":
			field.Int64()
		case "float", "double", "decimal":
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

	f := jen.NewFile("gen")
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

	f := jen.NewFile("gen")

	f.Func().Params(
		jen.Id("api").Op("*").Id(tableApi),
	).Id("Create").Params(
		jen.Id("c").Op("*").Qual("github.com/gin-gonic/gin", "Context"),
	).Block(
		jen.Id("logger").Op(":=").Id("c").Dot("MustMakeLog").Call(),

		jen.Var().Id(table).Qual("", tableModel),

		jen.If(
			jen.Err().Op(":=").Id("c").Dot("BindJSON").Call(jen.Op("&").Id(table)),
			jen.Err().Op("!=").Nil(),
		).Block(
			jen.Id("c").Dot("JSON").Call(jen.Lit(400), jen.Op("&").Qual("github.com/gin-gonic/gin", "H").Values(jen.Dict{
				jen.Id("error"): jen.Lit("Invalid JSON"),
			})),
			jen.Return(),
		),

		jen.Var().Id("db").Op("*").Qual("", "gorm.DB"),
		jen.Var().Err().Error(),

		jen.List(
			jen.Id("gormService"),
			jen.Err(),
		).Op(":=").Id("c").Dot("MustMake").Call(jen.Qual("", "ORMKey")),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Id("logger").Dot("Error").Call(jen.Id("c"), jen.Err().Dot("Error").Call(), jen.Nil()),
			jen.Id("_").Op("=").Id("c").Dot("AbortWithError").Call(jen.Lit(50001), jen.Err()),
			jen.Return(),
		),

		jen.If(
			jen.Err().Op(":=").Id("db").Dot("Create").Call(jen.Op("&").Id(table)).Dot("Error").Call(),
			jen.Err().Op("!=").Nil(),
		).Block(
			jen.Id("c").Dot("JSON").Call(jen.Lit(500), jen.Op("&").Qual("github.com/gin-gonic/gin", "H").Values(jen.Dict{
				jen.Id("error"): jen.Lit("Server error"),
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

	f := jen.NewFile("gen")

	f.Func().Params(jen.Id("api").Op("*").Id(tableApi)).Id("Delete").Params(jen.Id("c").Op("*").Qual("github."+
		"com/gin-gonic/gin", "Context")).Block(
		jen.Id("id, err := strconv.Atoi(c.Query(\"id\"))"),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Id("c.JSON(400, gin.H{\"error\": \"Invalid parameter\"})"),
			jen.Return(),
		),
		jen.Id("logger := c.MustMakeLog()"),
		jen.Comment("Initialize an orm.DB"),
		jen.Id("gormService := c.MustMake(contract.ORMKey).(contract.ORMService)"),
		jen.Id("db, err := gormService.GetDB(orm.WithConfigPath(\"database.default\"))"),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Id("logger.Error(c, err.Error(), nil)"),
			jen.Id("_ = c.AbortWithError(50001, err)"),
			jen.Return(),
		),
		jen.List(jen.Id("err")).Op(":=").Id("db.Delete").Call(jen.Op("&").Qual("", tableModel+"{}"),
			jen.Id("id")).Dot("Error"),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.If(jen.Err().Op("==").Qual("github.com/jinzhu/gorm", "ErrRecordNotFound")).Block(
				jen.Id("c.JSON(404, gin.H{\"error\": \"Record not found\"})"),
			).Else().Block(
				jen.Id("c.JSON(500, gin.H{\"error\": \"Server error\"})"),
			),
			jen.Return(),
		),
		jen.Comment("Return result"),
		jen.Id("c.Status(200)"),
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
	return nil
}

func (gen *ApiGenerator) GenApiShowFile(ctx context.Context, file string) error {
	// table lower case
	tableLower := strings.ToLower(gen.table)
	// table camel title case
	tableCamel := strings.Title(tableLower)
	tableModel := tableCamel + "Model"
	// Api struct
	tableApi := tableCamel + "Api"

	f := &jen.File{}
	f.PackageComment("Code generated by github.com/dave/jennifer. DO NOT EDIT.")
	/*
	    import (
	   	"github.com/gohade/hade/framework/contract"
	   	"github.com/gohade/hade/framework/gin"
	   	"github.com/gohade/hade/framework/provider/orm"
	   	"gorm.io/gorm"
	   	"strconv"
	   )
	*/
	f.ImportAlias("github.com/gohade/hade/framework/contract", "contract")
	f.ImportAlias("github.com/gohade/hade/framework/gin", "gin")
	f.ImportAlias("github.com/gohade/hade/framework/provider/orm", "orm")
	f.ImportAlias("gorm.io/gorm", "gorm")
	f.ImportAlias("strconv", "strconv")

	// Generate function signature
	// func (api *StudentApi) Show(c *gin.Context) {
	f.Func().Params(jen.Id("api").Op("*").Id(tableApi)).Id("Show").Params(jen.Id("c").Op("*").Qual(
		"github.com/gohade/hade/framework/gin", "Context")).Block(
		// id, err := strconv.Atoi(c.Query("id"))
		jen.Id("id").Op(",").Id("err").Op(":=").Qual("strconv", "Atoi").Call(jen.Id("c").Dot("Query").Call(jen.Lit("id"))),
		/*
		   if err != nil {
		   		c.JSON(400, gin.H{"error": "Invalid parameter"})
		   		return
		   	}
		*/
		jen.If(jen.Id("err").Op("!=").Nil()).Block(
			jen.Id("c").Dot("JSON").Call(
				jen.Lit(400),
				jen.Qual("github.com/gohade/hade/framework/gin", "H").Values(
					jen.Dict{
						jen.Lit("error"): jen.Lit("Invalid parameter"),
					}),
			),
			jen.Return(),
		),

		// logger := c.MustMakeLog()
		jen.Id("logger").Op(":=").Id("c").Dot("MustMakeLog").Call(),
		// 	gormService := c.MustMake(contract.ORMKey).(contract.ORMService)
		jen.Id("gormService").Op(":=").Id("c").Dot("MustMake").Call(jen.Lit("orm")).Assert(jen.Op("*").Qual("github.com/gohade/hade/framework/provider/orm", "ORMService")),
		// 	db, err := gormService.GetDB(orm.WithConfigPath("database.default"))
		jen.Id("db").Op(",").Id("err").Op(":=").Id("gormService").Dot("GetDB").Call(jen.Qual("github.com/gohade/hade/framework/provider/orm", "WithConfigPath").Call(jen.Lit("database.default"))),
		/*
		   if err != nil {
		   		logger.Error(c, err.Error(), nil)
		   		_ = c.AbortWithError(50001, err)
		   		return
		   	}
		*/
		jen.If(jen.Id("err").Op("!=").Nil()).Block(
			jen.Id("logger").Dot("Error").Call(jen.Id("c"), jen.Id("err").Dot("Error").Call(), jen.Nil()),
			jen.Id("_").Op("=").Id("c").Dot("AbortWithError").Call(jen.Lit(50001), jen.Id("err")),
			jen.Return(),
		),
		// 	var student StudentModel
		jen.Var().Id(tableLower).Id(tableModel),
		// 	if err := db.First(&student, id).Error; err != nil {
		jen.If(jen.Id("err").Op(":=").Id("db").Dot("First").Call(jen.Op("&").Id(tableLower),
			jen.Id("id")).Dot("Error"), jen.Id("err").Op("!=").Nil()).Block(
			// if err == gorm.ErrRecordNotFound {
			jen.If(jen.Id("err").Op("==").Qual("gorm.io/gorm", "ErrRecordNotFound")).Block(
				// c.JSON(404, gin.H{"error": "Record not found"})
				jen.Id("c").Dot("JSON").Call(
					jen.Lit(404),
					jen.Qual("github.com/gohade/hade/framework/gin", "H").Values(
						jen.Dict{
							jen.Lit("error"): jen.Lit("Record not found"),
						}),
				),
				jen.Return(),
			),
			// } else {
			jen.Else().Block(
				// c.JSON(500, gin.H{"error": "Server error"})
				jen.Id("c").Dot("JSON").Call(
					jen.Lit(500),
					jen.Qual("github.com/gohade/hade/framework/gin", "H").Values(
						jen.Dict{
							jen.Lit("error"): jen.Lit("Server error"),
						}),
				),
				jen.Return(),
			),
		),
		// c.JSON(200, student)
		jen.Id("c").Dot("JSON").Call(
			jen.Lit(200),
			jen.Id(tableLower),
		),
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

func (gen *ApiGenerator) GenApiUpdateFile(ctx context.Context, file string) error {
	return nil
}
