package gen

import (
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/gin"
	"github.com/gohade/hade/framework/provider/orm"
)

// Create 创建学生
// @Summary 创建学生
// @Description 创建一个新的学生
// @Produce json
// @Tags students
// @Param student body StudentModel true "学生信息"
// @Success 201 {object} StudentModel
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /students [post]
func (api *StudentApi) Create(c *gin.Context) {
	logger := c.MustMakeLog()

	// 绑定JSON数据到student结构体中
	var student StudentModel
	if err := c.BindJSON(&student); err != nil {
		c.JSON(400, gin.H{"error": "Invalid JSON"})
		return
	}

	// 初始化一个orm.DB
	gormService := c.MustMake(contract.ORMKey).(contract.ORMService)
	db, err := gormService.GetDB(orm.WithConfigPath("database.default"))
	if err != nil {
		logger.Error(c, err.Error(), nil)
		_ = c.AbortWithError(50001, err)
		return
	}

	// 向数据库中添加新的学生模型
	if err := db.Create(&student).Error; err != nil {
		c.JSON(500, gin.H{"error": "Server error"})
		return
	}

	// 返回新创建的学生模型
	c.JSON(200, student)
}
