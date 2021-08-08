package bot

import (
	"github.com/spf13/cobra"
	"sci_hub_p2p/cmd/bot/telegram"
)

var Cmd = &cobra.Command{
	Use: "bot",
}

func init() {
	Cmd.AddCommand(telegram.Cmd)
}
