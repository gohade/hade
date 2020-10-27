package command

import (
	"fmt"
	"log"
	"os/exec"

	"hade/framework/cobra"
)

var buildCommand = &cobra.Command{
	Use:   "build",
	Short: "build hade command",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) == 0 {
			c.Help()
		}
		return nil
	},
}

var buildSelfCommand = &cobra.Command{
	Use:   "self",
	Short: "build ./hade command",
	RunE: func(c *cobra.Command, args []string) error {
		path, err := exec.LookPath("go")
		if err != nil {
			log.Fatalln("hade go: please install go in path first")
		}

		cmd := exec.Command(path, "build", "-o", "hade", "./")
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println("go build error:")
			fmt.Println(string(out))
			fmt.Println("--------------")
			return err
		}
		fmt.Println("build success please run ./hade direct")
		return nil
	},
}

var buildBackendCommand = &cobra.Command{
	Use:   "backend",
	Short: "build backend use go",
	RunE: func(c *cobra.Command, args []string) error {
		return buildSelfCommand.RunE(c, args)
	},
}

var buildFrontendCommand = &cobra.Command{
	Use:   "frontend",
	Short: "build frontend use npm",
	RunE: func(c *cobra.Command, args []string) error {
		path, err := exec.LookPath("npm")
		if err != nil {
			log.Fatalln("hade npm: should install npm in your PATH")
		}

		cmd := exec.Command(path, "run", "build")
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println("npm build error:")
			fmt.Println(string(out))
			fmt.Println("--------------")
			return err
		}
		fmt.Print(string(out))
		fmt.Println("front end build success")
		return nil
	},
}

var buildAllCommand = &cobra.Command{
	Use:   "all",
	Short: "build fronend and backend",
	RunE: func(c *cobra.Command, args []string) error {
		err := buildFrontendCommand.RunE(c, args)
		if err != nil {
			return err
		}
		err = buildBackendCommand.RunE(c, args)
		if err != nil {
			return err
		}
		return nil
	},
}
