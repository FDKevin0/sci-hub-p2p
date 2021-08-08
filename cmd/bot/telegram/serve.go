package telegram

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"
	"sci_hub_p2p/internal/utils"
	"sci_hub_p2p/pkg/consts"
	"sci_hub_p2p/pkg/logger"
	"sci_hub_p2p/pkg/vars"
)

var Cmd = &cobra.Command{
	Use: "telegram",
}

var serveCmd = &cobra.Command{
	Use:           "serve",
	Short:         "serve a telegram bot",
	Example:       "bot telegram serve -t xxxxxx:xxxxxxx",
	SilenceErrors: false,
	PreRunE:       utils.EnsureDir(vars.GetAppTmpDir()),
	RunE: func(cmd *cobra.Command, args []string) error {
		if token == "" {
			return errors.New("doi can't be empty string")
		}
		db, err := bbolt.Open(vars.TelegramFilesDBPath(), consts.DefaultFilePerm, bbolt.DefaultOptions)
		if err != nil {
			return errors.Wrap(err, "can't open Telegram DB")
		}

		err = db.Update(func(tx *bbolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists(consts.TelegramFileBucketName())
			if err != nil {
				return errors.Wrap(err, "can't create bucket in database")
			}
			return nil
		})
		if err != nil {
			return err
		}

		if e := db.Close(); e != nil {
			e = errors.Wrap(e, "can't save data to disk")
			if err == nil {
				err = e
			} else {
				logger.Error("", zap.Error(e))
			}
		}

		b, err := setBot(token)
		b.Start()
		return err
	},
}

var token string

func init() {
	Cmd.AddCommand(serveCmd)

	serveCmd.Flags().StringVarP(&token, "token", "t", "", "")

	if err := utils.MarkFlagsRequired(serveCmd, "token"); err != nil {
		panic(err)
	}
}
