package gen

import (
	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/gin"
	"github.com/gohade/hade/framework/provider/orm"
	"gorm.io/gorm"
	"strconv"
)

// Delete 删除学生
// @Summary 删除学生
// @Description 根据主键id删除学生
// @Produce json
// @Tags students
// @Param id path int true "学生主键id"
// @Success 200 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /students/delete [post]
func (api *StudentApi) Delete(c *gin.Context) {
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

	// 从数据库中删除
	if err := db.Delete(&StudentModel{}, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, gin.H{"error": "Record not found"})
			return
		} else {
			c.JSON(500, gin.H{"error": "Server error"})
			return
		}
	}

	// 返回结果
	c.Status(200)
}
