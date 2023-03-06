package gen

import (
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/gin"
	"github.com/gohade/hade/framework/provider/orm"
	"gorm.io/gorm"
	"strconv"
)

// Update 更新学生
// @Summary 更新学生
// @Description 更新现有学生
// @Produce json
// @Tags students
// @Param student body StudentModel true "学生信息"
// @Success 200 {object} StudentModel
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /students/update [post]
func (api *StudentApi) Update(c *gin.Context) {
	logger := c.MustMakeLog()

	// 获取id参数
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid parameter"})
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

	// 绑定JSON数据到student结构体中
	if err := c.BindJSON(&student); err != nil {
		c.JSON(400, gin.H{"error": "Invalid JSON"})
		return
	}

	// 保存更新后的学生模型到数据库中
	if err := db.Save(&student).Error; err != nil {
		c.JSON(500, gin.H{"error": "Server error"})
		return
	}

	// 返回更新后的学生模型
	c.JSON(200, student)
}
