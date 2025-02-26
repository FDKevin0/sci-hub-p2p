// Copyright 2021 Trim21 <trim21.me@gmail.com>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
// See the GNU General Public License for more details.
package torrent

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"

	"sci_hub_p2p/internal/torrent"
	"sci_hub_p2p/internal/utils"
	"sci_hub_p2p/pkg/consts"
	"sci_hub_p2p/pkg/consts/size"
	"sci_hub_p2p/pkg/logger"
	"sci_hub_p2p/pkg/vars"
)

var Cmd = &cobra.Command{
	Use:           "torrent",
	Short:         "load torrent into database",
	SilenceErrors: false,
}

var loadCmd = &cobra.Command{
	Use:           "load",
	Short:         "Load torrents into database.",
	Example:       "torrent load 1.torrent 2.torrent [--glob '/path/to/data/*.torrent']",
	SilenceErrors: false,
	PreRunE:       utils.EnsureDir(vars.GetAppBaseDir()),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		args, err = utils.MergeGlob(args, glob)
		if err != nil {
			return errors.Wrap(err, "can't load any torrent files")
		}

		db, err := bbolt.Open(vars.TorrentDBPath(), consts.DefaultFilePerm, bbolt.DefaultOptions)
		if err != nil {
			return errors.Wrap(err, "can't open Torrent DB")
		}
		defer func(db *bbolt.DB) {
			if e := db.Close(); e != nil {
				e = errors.Wrap(e, "can't save data to disk")
				if err == nil {
					err = e
				} else {
					logger.Error("", zap.Error(e))
				}
			}
		}(db)

		err = db.Batch(func(tx *bbolt.Tx) error {
			b, err := tx.CreateBucketIfNotExists(consts.TorrentBucket())
			if err != nil {
				return errors.Wrap(err, "can't create bucket in database")
			}
			for _, file := range args {
				raw, err := os.ReadFile(file)
				if err != nil {
					return errors.Wrap(err, "failed to read file content")
				}

				t, err := torrent.ParseRaw(raw)
				if err != nil {
					return errors.Wrapf(err, "failed to parse torrent %s", file)
				}

				err = b.Put(t.RawInfoHash(), raw)
				if err != nil {
					return err
				}
			}

			return nil
		})
		if err != nil {
			return errors.Wrap(err, "can't save torrent data to database")
		}
		fmt.Printf("successfully load %d torrents into database\n", len(args))

		return nil
	},
}

var getCmd = &cobra.Command{
	Use:           "get",
	Short:         "get torrent data database.",
	Example:       "torrent get ${InfoHash}",
	Args:          cobra.ExactArgs(1),
	SilenceErrors: false,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if len(args[0]) != size.Sha1Hex {
			return fmt.Errorf("%s is not a valid sha1", args[0])
		}

		var db *bbolt.DB
		db, err = bbolt.Open(vars.TorrentDBPath(), consts.DefaultFilePerm, bbolt.DefaultOptions)
		if err != nil {
			return errors.Wrap(err, "cant' open database file, maybe another process is running?")
		}
		defer func(db *bbolt.DB) {
			e := db.Close()
			if e != nil {
				if err == nil {
					err = e
				} else {
					logger.Error("", zap.Error(e))
				}
			}
		}(db)
		p, err := hex.DecodeString(args[0])
		if err != nil {
			return errors.Wrap(err, "info hash is not valid hex string")
		}

		err = db.View(func(tx *bbolt.Tx) error {
			b := tx.Bucket(consts.TorrentBucket())
			if b == nil {
				return fmt.Errorf("can't find data in database")
			}
			raw := b.Get(p)
			if raw == nil {
				return fmt.Errorf("can't find torrent in database")
			}
			t, err := torrent.ParseRaw(raw)
			if err != nil {
				return err
			}

			s, err := t.DumpIndent()
			if err != nil {
				return errors.Wrap(err, "can't dump torrent data into json format")
			}

			fmt.Println(s)

			return nil
		})

		if err != nil {
			return errors.Wrap(err, "can't get torrent from database")
		}

		return nil
	},
}

var glob string

func init() {
	Cmd.AddCommand(loadCmd, getCmd, verifyCmd, fetchCmd)

	loadCmd.Flags().StringVar(&glob, "glob", "",
		"glob pattern to search torrents to avoid 'Argument list too long' error")
}
