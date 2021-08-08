package telegram

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"go.etcd.io/bbolt"
	tb "gopkg.in/tucnak/telebot.v2"
	"os"
	"path/filepath"
	"sci_hub_p2p/internal/client"
	"sci_hub_p2p/pkg/consts"
	"sci_hub_p2p/pkg/logger"
	"sci_hub_p2p/pkg/persist"
	"sci_hub_p2p/pkg/vars"
	"strings"
	"time"
)

func setBot(token string) (*tb.Bot, error) {
	b, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		logger.Fatal(err.Error())
		return nil, err
	}

	b.Handle("/doi", func(m *tb.Message) {
		doi := strings.TrimSuffix(m.Payload, ".pdf")
		filePath := filepath.Join(vars.GetAppBaseDir(), "papers", doi+".pdf")
		file, err := getFileFromBucket(doi)
		msg := new(tb.Message)
		if err != nil || file == nil {
			msg, _ = b.Reply(m, errors.Wrap(err, "failed to get file from cloud, start index"))
		} else {
			msg, _ = b.Reply(m, "get file from cloud, start to upload")
		}
		if file == nil || err != nil || !file.InCloud() {
			r, err := persist.GetIndexRecord([]byte(doi))
			if err != nil {
				_, _ = b.Edit(msg, err.Error())
				return
			} else {
				_, _ = b.Edit(msg, "get doi from index")
			}
			t, err := persist.GetTorrent(r.InfoHash[:])
			if err != nil {
				_, _ = b.Edit(msg, err.Error())
				return
			} else {
				_, _ = b.Edit(msg, "downloading")
			}

			p, err := r.Build(doi, t)
			if err != nil {
				_, _ = b.Edit(msg, err.Error())
				return
			}

			fileByte, err := client.Fetch(p, t.Raw())
			if err != nil {
				_, _ = b.Edit(msg, err.Error())
				return
			}
			directory, _ := filepath.Split(filePath)
			err = os.MkdirAll(directory, os.ModePerm)
			if err != nil {
				_, _ = b.Edit(msg, err.Error())
				return
			}
			err = os.WriteFile(filePath, fileByte, consts.DefaultFilePerm)
			if err != nil {
				_, _ = b.Edit(msg, err.Error())
				return
			}
			f := tb.FromDisk(filePath)
			file = &f
			_, _ = b.Edit(msg, "uploading")
		}

		a := &tb.Document{File: *file, FileName: doi + ".pdf"}

		_, err = b.Reply(m, a)
		if err != nil {
			_, _ = b.Send(m.Sender, err.Error())
			return
		}

		_ = b.Delete(msg)

		if a.InCloud() {
			err = saveFileFromBucket(doi, a.File)
			if err != nil {
				fmt.Println(err)
				return
			}
		}
	})

	return b, nil
}

func getFileFromBucket(doi string) (*tb.File, error) {
	file := &tb.File{}
	fDB, err := bbolt.Open(vars.TelegramFilesDBPath(), consts.DefaultFilePerm, bbolt.DefaultOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open indexes database")
	}
	defer fDB.Close()
	err = fDB.View(func(tx *bbolt.Tx) error {
		if v := tx.Bucket(consts.TelegramFileBucketName()).Get([]byte(doi + ".pdf")); v != nil {
			err = json.Unmarshal(v, file)
			if err != nil {
				return errors.Wrapf(err, "failed to unmarshal file: %s.pdf", doi)
			}
		}
		return nil
	})
	return file, err
}

func saveFileFromBucket(doi string, file tb.File) error {
	fDB, err := bbolt.Open(vars.TelegramFilesDBPath(), consts.DefaultFilePerm, bbolt.DefaultOptions)
	if err != nil {
		return errors.Wrap(err, "failed to open indexes database")
	}
	defer fDB.Close()
	err = fDB.Update(func(tx *bbolt.Tx) error {
		fileBytes, err := json.Marshal(file)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal file")
		}
		err = tx.Bucket(consts.TelegramFileBucketName()).Put([]byte(doi+".pdf"), fileBytes)
		if err != nil {
			return errors.Wrapf(err, "can't save file to database")
		}
		return nil
	})
	return err
}
