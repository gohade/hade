package agent

import (
	"github.com/gohade/hade/framework/cobra"
)

func newAgentRestartCommand(options *agentOptions, deps agentDependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "重新启动一个 agent 服务",
		RunE: func(c *cobra.Command, args []string) error {
			runtime, err := prepareAgentRuntime(c, options)
			if err != nil {
				return err
			}
			return withLifecycleLock(runtime.lifecycleFile, func(lock *exclusiveFileLock) error {
				options.daemon = true
				return restartAfterStop(
					func() error {
						_, err := stopAgentLocked(runtime.pidFile, runtime.readyFile, deps.process, agentStopWait(c))
						return err
					},
					func() error {
						return startAgentLocked(runtime, options, lock, nil)
					},
				)
			})
		},
	}
}
