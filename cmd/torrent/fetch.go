package torrent

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/cheggaaa/pb/v3"
	"github.com/parnurzeal/gorequest"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sci_hub_p2p/internal/torrent"
	"sci_hub_p2p/internal/utils"
	"sci_hub_p2p/pkg/logger"
	"strings"
	"sync"
	"time"
)

var fetchCmd = &cobra.Command{
	Use:           "fetch",
	Short:         "fetch torrent from internet",
	Example:       "torrent fetch -d /path/to/torrent/ -c channel to fetch",
	SilenceErrors: false,
	Args:          cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		torrentFileNames := make([]string, 0)
		resp, _, errs := gorequest.New().Get(LibGenUrl).End()
		if errs != nil {
			return errors.Wrap(errs[0], "failed to fetch torrents links")
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			return errors.Wrap(errs[0], "failed to fetch torrents links")
		}

		doc.Find("tr>td>a[href]").Each(func(i int, s *goquery.Selection) {
			fileName, exists := s.Attr("href")
			if exists && strings.HasSuffix(fileName, "torrent") {
				torrentFileNames = append(torrentFileNames, fileName)
			}
		})

		bar := pb.StartNew(len(torrentFileNames))
		ch := make(chan bool, channel)
		logger.Info(fmt.Sprintf("fetch %d torrents from server: %s", len(torrentFileNames), LibGenUrl))
		for _, v := range torrentFileNames {
			wg.Add(1)
			go fetchTorrent(ch, bar, v)
		}
		wg.Wait()
		bar.Finish()
		fmt.Printf("successfully fetch %d torrents\n", len(torrentFileNames))

		return nil
	},
}

const LibGenUrl = "https://libgen.rs/scimag/repository_torrent/"

var wg = sync.WaitGroup{}

func fetchTorrent(ch chan bool, bar *pb.ProgressBar, fileName string) {
	defer wg.Done()
	fileLink := LibGenUrl + fileName
	ch <- true
	resp, _, errs := gorequest.New().Get(fileLink).
		Retry(3, 100*time.Millisecond, http.StatusBadRequest, http.StatusInternalServerError).
		End()
	if errs == nil {
		defer resp.Body.Close()

		out, err := os.Create(filepath.Join(torrentPath, fileName))
		if err != nil {
			logger.Fatal(err.Error())
		}
		defer out.Close()

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			logger.Fatal(err.Error())
		}
		raw, err := os.ReadFile(filepath.Join(torrentPath, fileName))
		if err == nil {
			_, err = torrent.ParseRaw(raw)
			if err == nil {
				bar.Increment()
				<-ch
				return
			}
		}
	}
	logger.Info(fmt.Sprintf("fetch torrent %s error, will retry at last", fileName))
	wg.Add(1)
	go fetchTorrent(ch, bar, fileName)
	<-ch
}

var channel int

func init() {
	fetchCmd.Flags().StringVarP(&torrentPath, "torrent", "t", "", "torrent path")
	fetchCmd.Flags().IntVarP(&channel, "channel", "c", 10, "channel to download")

	if err := utils.MarkFlagsRequired(fetchCmd, "torrent"); err != nil {
		panic(err)
	}
}
