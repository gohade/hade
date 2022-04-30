package auto

import (
    "github.com/gohade/hade/framework/cobra"
    "gopkg.in/yaml.v3"
    "io/ioutil"
    "path/filepath"
)

var configFile string

func InitAutoApiCommand() *cobra.Command {
    return nil
}

var AutoCommand = &cobra.Command{
    Use:   "auto",
    Short: "自动生成工具",
    RunE: func(c *cobra.Command, args []string) error {
        if len(args) == 0 {
            c.Help()
        }
        return nil
    },
}

// AutoApiCommand 生成api代码
var AutoApiCommand = &cobra.Command{
    Use:   "api",
    Short: "自动生成api",
    RunE: func(c *cobra.Command, args []string) error {
        // auto code
        folder := filepath.Dir(configFile)
        // read config file
        file, err := ioutil.ReadFile(configFile)
        if err != nil {
            return err
        }

        yaml.NewDecoder(file).Decode()
    },
}
