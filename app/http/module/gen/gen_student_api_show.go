package gen

import (
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/gin"
	"github.com/gohade/hade/framework/provider/orm"
	"gorm.io/gorm"
	"strconv"
)

// Show 获取指定id的学生信息
// @Summary 获取学生信息
// @Description 根据主键id获取学生信息
// @Produce json
// @Tags students
// @Param id query int true "学生主键id"
// @Success 200 {object} Student
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /students/show [get]
func (api *StudentApi) Show(c *gin.Context) {
	// 获取id参数
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid parameter"})
		return
	}

	logger := c.MustMakeLog()

	// 初始化一个orm.DB
	gormService := c.MustMake(contract.ORMKey).(contract.ORMService)
	db, err := gormService.GetDB(orm.WithConfigPath("database.default"))
	if err != nil {
		logger.Error(c, err.Error(), nil)
		_ = c.AbortWithError(50001, err)
		return
	}

	// 从数据库中查找
	var student StudentModel
	if err := db.First(&student, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, gin.H{"error": "Record not found"})
			return
		} else {
			c.JSON(500, gin.H{"error": "Server error"})
			return
		}
	}

	// 返回结果
	c.JSON(200, student)
}
