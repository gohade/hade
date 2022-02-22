package demo

import (
    "github.com/gohade/hade/framework/contract"
    "github.com/gohade/hade/framework/gin"
    "github.com/gohade/hade/framework/provider/orm"
    "github.com/gohade/hade/framework/util/goroutine"
)

// DemoGoroutine goroutine 的使用示例
func (api *DemoApi) DemoGoroutine(c *gin.Context) {
    logger := c.MustMakeLog()
    logger.Info(c, "request start", nil)

    // 初始化一个orm.DB
    gormService := c.MustMake(contract.ORMKey).(contract.ORMService)
    db, err := gormService.GetDB(orm.WithConfigPath("database.default"))
    if err != nil {
        logger.Error(c, err.Error(), nil)
        c.AbortWithError(50001, err)
        return
    }
    db.WithContext(c)

    err = goroutine.SafeGoAndWait(c, func() error {
        // 查询一条数据
        queryUser := &User{ID: 1}

        err = db.First(queryUser).Error
        logger.Info(c, "query user1", map[string]interface{}{
            "err":  err,
            "name": queryUser.Name,
        })
        return err
    }, func() error {
        // 查询一条数据
        queryUser := &User{ID: 2}

        err = db.First(queryUser).Error
        logger.Info(c, "query user2", map[string]interface{}{
            "err":  err,
            "name": queryUser.Name,
        })
        return err
    })

    if err != nil {
        c.AbortWithError(50001, err)
        return
    }
    c.JSON(200, "ok")
}
