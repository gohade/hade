package gen

import (
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/gin"
	"github.com/gohade/hade/framework/provider/orm"
	"strconv"
)

// List 获取学生列表
// @Summary 获取学生列表
// @Description 获取分页的学生信息列表
// @Produce json
// @Tags students
// @Param offset query int true "偏移量，从0开始"
// @Param size query int true "每页大小"
// @Success 200 {array} Student
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /students/list [get]
func (api *StudentApi) List(c *gin.Context) {
	// 获取offset和size参数
	offset, err := strconv.Atoi(c.Query("offset"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid parameter"})
		return
	}
	size, err := strconv.Atoi(c.Query("size"))
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

	// 获取学生总数
	var total int64
	if err := db.Model(&StudentModel{}).Count(&total).Error; err != nil {
		c.JSON(500, gin.H{"error": "Server error"})
		return
	}

	// 从数据库中查找学生列表
	var students []StudentModel
	if err := db.Offset(offset).Limit(size).Find(&students).Error; err != nil {
		c.JSON(500, gin.H{"error": "Server error"})
		return
	}

	// 返回结果
	c.JSON(200, gin.H{"total": total, "students": students})
}
