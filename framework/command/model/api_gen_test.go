package model

import (
	"context"
	"fmt"
	"github.com/gohade/hade/framework/contract"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestApiGenerator_GenModelFile(t *testing.T) {
	gen := &ApiGenerator{
		table: "user",
		columns: []contract.TableColumn{
			{Field: "id", Type: "int"},
			{Field: "name", Type: "varchar"},
			{Field: "age", Type: "int"},
		},
	}

	// Create a temporary file for the model code
	tmpfile, err := ioutil.TempFile("", "model.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	// Generate the model code
	if err := gen.GenModelFile(context.Background(), tmpfile.Name()); err != nil {
		t.Fatal(err)
	}

	// Read the generated code from the file
	bytes, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	code := string(bytes)
	fmt.Println(code)

	// Check that the code contains the expected model name
	expectedModelName := "UserModel"
	if !strings.Contains(code, "type "+expectedModelName+" struct") {
		t.Errorf("Generated code does not contain expected model name %q", expectedModelName)
	}

	// Check that the code contains the expected table name
	expectedTableName := "user"
	if !strings.Contains(code, "return "+strconv.Quote(expectedTableName)) {
		t.Errorf("Generated code does not contain expected table name %q", expectedTableName)
	}

	// Check that the code contains the expected fields
	for _, column := range gen.columns {
		expectedFieldName := strings.Title(column.Field)
		if !strings.Contains(code, ""+expectedFieldName+" ") {
			t.Errorf("Generated code does not contain expected field name %q", expectedFieldName)
		}
	}
}

func TestApiGenerator_GenRouterFile(t *testing.T) {
	gen := &ApiGenerator{
		table: "user",
		columns: []contract.TableColumn{
			{Field: "id", Type: "int"},
			{Field: "name", Type: "varchar"},
			{Field: "age", Type: "int"},
		},
	}

	// Create a temporary file for the model code
	tmpfile, err := ioutil.TempFile("", "router.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	// Generate the model code
	if err := gen.GenRouterFile(context.Background(), tmpfile.Name()); err != nil {
		t.Fatal(err)
	}

	// Read the generated code from the file
	bytes, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	code := string(bytes)
	fmt.Println(code)

	expectedModelName := "UserApi"
	if !strings.Contains(code, "type "+expectedModelName+" struct") {
		t.Errorf("Generated code does not contain expected model name %q", expectedModelName)
	}

	expectedFuncName := "NewUserApi"
	if !strings.Contains(code, "func "+expectedFuncName+"()") {
		t.Errorf("Generated code does not contain expected func name %q", expectedFuncName)
	}

	expectedFuncName = "UserApiRegister"
	if !strings.Contains(code, "func "+expectedFuncName+"(r *gin.Engine) error") {
		t.Errorf("Generated code does not contain expected func name %q", expectedFuncName)
	}

}

func TestApiGenerator_GenCreateFile(t *testing.T) {
	gen := &ApiGenerator{
		table: "user",
		columns: []contract.TableColumn{
			{Field: "id", Type: "int"},
			{Field: "name", Type: "varchar"},
			{Field: "age", Type: "int"},
		},
	}

	// Create a temporary file for the model code
	tmpfile, err := ioutil.TempFile("", "create.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	// Generate the model code
	if err := gen.GenApiCreateFile(context.Background(), tmpfile.Name()); err != nil {
		t.Fatal(err)
	}

	// Read the generated code from the file
	bytes, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	code := string(bytes)
	t.Log(code)

	expectedFuncName := "Create"
	if !strings.Contains(code, "Create(c *gin.Context)") {
		t.Errorf("Generated code does not contain expected func name %q", expectedFuncName)
	}
}

func TestApiGenerator_GenDeleteFile(t *testing.T) {
	gen := &ApiGenerator{
		table: "user",
		columns: []contract.TableColumn{
			{Field: "id", Type: "int"},
			{Field: "name", Type: "varchar"},
			{Field: "age", Type: "int"},
		},
	}

	// Create a temporary file for the model code
	tmpfile, err := ioutil.TempFile("", "delete.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	// Generate the model code
	if err := gen.GenApiDeleteFile(context.Background(), tmpfile.Name()); err != nil {
		t.Fatal(err)
	}

	// Read the generated code from the file
	bytes, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	code := string(bytes)
	t.Log(code)

	expectedFuncName := "Delete"
	if !strings.Contains(code, expectedFuncName+"(c *gin.Context)") {
		t.Errorf("Generated code does not contain expected func name %q", expectedFuncName)
	}
}

func TestApiGenerator_GenListFile(t *testing.T) {
	gen := &ApiGenerator{
		table: "user",
		columns: []contract.TableColumn{
			{Field: "id", Type: "int"},
			{Field: "name", Type: "varchar"},
			{Field: "age", Type: "int"},
		},
	}

	// Create a temporary file for the model code
	tmpfile, err := ioutil.TempFile("", "list.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	// Generate the model code
	if err := gen.GenApiListFile(context.Background(), tmpfile.Name()); err != nil {
		t.Fatal(err)
	}

	// Read the generated code from the file
	bytes, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	code := string(bytes)
	t.Log(code)

	expectedFuncName := "List"
	if !strings.Contains(code, expectedFuncName+"(c *gin.Context)") {
		t.Errorf("Generated code does not contain expected func name %q", expectedFuncName)
	}
}

func TestApiGenerator_GenShowFile(t *testing.T) {
	gen := &ApiGenerator{
		table: "user",
		columns: []contract.TableColumn{
			{Field: "id", Type: "int"},
			{Field: "name", Type: "varchar"},
			{Field: "age", Type: "int"},
		},
	}

	// Create a temporary file for the model code
	tmpfile, err := ioutil.TempFile("", "show.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	// Generate the model code
	if err := gen.GenApiListFile(context.Background(), tmpfile.Name()); err != nil {
		t.Fatal(err)
	}

	// Read the generated code from the file
	bytes, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	code := string(bytes)
	t.Log(code)

	expectedFuncName := "Show"
	if !strings.Contains(code, expectedFuncName+"(c *gin.Context)") {
		t.Errorf("Generated code does not contain expected func name %q", expectedFuncName)
	}
}

func TestApiGenerator_GenUpdateFile(t *testing.T) {
	gen := &ApiGenerator{
		table: "user",
		columns: []contract.TableColumn{
			{Field: "id", Type: "int"},
			{Field: "name", Type: "varchar"},
			{Field: "age", Type: "int"},
		},
	}

	// Create a temporary file for the model code
	tmpfile, err := ioutil.TempFile("", "update.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	// Generate the model code
	if err := gen.GenApiListFile(context.Background(), tmpfile.Name()); err != nil {
		t.Fatal(err)
	}

	// Read the generated code from the file
	bytes, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	code := string(bytes)
	t.Log(code)

	expectedFuncName := "Update"
	if !strings.Contains(code, expectedFuncName+"(c *gin.Context)") {
		t.Errorf("Generated code does not contain expected func name %q", expectedFuncName)
	}
}
