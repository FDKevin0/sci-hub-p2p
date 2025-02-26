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
package paper

import (
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"sci_hub_p2p/internal/client"
	"sci_hub_p2p/internal/utils"
	"sci_hub_p2p/pkg/consts"
	"sci_hub_p2p/pkg/persist"
	"sci_hub_p2p/pkg/vars"
)

var Cmd = &cobra.Command{
	Use:           "paper",
	SilenceErrors: false,
}

var fetchCmd = &cobra.Command{
	Use:           "fetch",
	Short:         "fetch a paper from p2p network",
	Example:       "paper fetch --doi '10.1145/1327452.1327492' -o map-reduce.pdf",
	SilenceErrors: false,
	PreRunE:       utils.EnsureDir(vars.GetAppTmpDir()),
	RunE: func(cmd *cobra.Command, args []string) error {
		if doi == "" {
			return errors.New("doi can't be empty string")
		}

		doi = strings.TrimSuffix(doi, ".pdf")
		r, err := persist.GetIndexRecord([]byte(doi))
		if err != nil {
			return err
		}

		t, err := persist.GetTorrent(r.InfoHash[:])
		if err != nil {
			return err
		}

		p, err := r.Build(doi, t)
		if err != nil {
			return err
		}

		b, err := client.Fetch(p, t.Raw())
		if err != nil {
			return err
		}
		err = os.WriteFile(out, b, consts.DefaultFilePerm)

		return err
	},
}

var doi string
var out string

func init() {
	Cmd.AddCommand(fetchCmd)

	fetchCmd.Flags().StringVar(&doi, "doi", "", "")
	fetchCmd.Flags().StringVarP(&out, "output", "o", "", "output file path")

	if err := utils.MarkFlagsRequired(fetchCmd, "doi", "output"); err != nil {
		panic(err)
	}
}
