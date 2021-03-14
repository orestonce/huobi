package huobi

import (
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"os/exec"
)

var InstallCmd = &cobra.Command{
	Use: `install`,
	Run: func(cmd *cobra.Command, args []string) {
		content := `[Service]
StartLimitInterval=3600
StartLimitBurst=10
ExecStart=/usr/local/bin/huobi collect
Restart=on-failure
RestartSec=120
KillMode=process
[Install]
WantedBy=multi-user.target`
		err := ioutil.WriteFile(`/etc/systemd/system/huobi.service`, []byte(content), 0777)
		if err != nil {
			panic(err)
		}
		exePath, err := os.Executable()
		if err != nil {
			panic(err)
		}
		const installTarget = `/usr/local/bin/huobi`
		if exePath != installTarget {
			_, err = os.Stat(installTarget)
			if err == nil {
				err = os.Remove(installTarget)
				if err != nil {
					panic(err)
				}
			}
			var exeContent []byte
			exeContent, err = ioutil.ReadFile(exePath)
			if err != nil {
				panic(err)
			}
			err = ioutil.WriteFile(installTarget, exeContent, 0777)
			if err != nil {
				panic(err)
			}
			err = os.Remove(exePath)
			if err != nil {
				panic(err)
			}
		}
		err = exec.Command("systemctl", "daemon-reload").Run()
		if err != nil {
			panic(err)
		}
		err = exec.Command("systemctl", "restart", "huobi").Run()
		if err != nil {
			panic(err)
		}
	},
}
