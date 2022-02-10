package env

// HadeEnv 是 Env 的具体实现
type HadeTestingEnv struct {
    folder string            // 代表.env所在的目录
    maps   map[string]string // 保存所有的环境变量
}

// NewHadeEnv 有一个参数，.env文件所在的目录
// example: NewHadeEnv("/envfolder/") 会读取文件: /envfolder/.env
// .env的文件格式 FOO_ENV=BAR
func NewHadeTestingEnv(params ...interface{}) (interface{}, error) {

    // 返回实例
    return &HadeTestingEnv{}, nil
}

// AppEnv 获取表示当前APP环境的变量APP_ENV
func (en *HadeTestingEnv) AppEnv() string {
    return "testing"
}

// IsExist 判断一个环境变量是否有被设置
func (en *HadeTestingEnv) IsExist(key string) bool {
    _, ok := en.maps[key]
    return ok
}

// Get 获取某个环境变量，如果没有设置，返回""
func (en *HadeTestingEnv) Get(key string) string {
    if val, ok := en.maps[key]; ok {
        return val
    }
    return ""
}

// All 获取所有的环境变量，.env和运行环境变量融合后结果
func (en *HadeTestingEnv) All() map[string]string {
    return en.maps
}
