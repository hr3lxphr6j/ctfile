package main

import (
	"flag"
	"log"
	"path"
	"time"

	"github.com/hr3lxphr6j/ctfile/aria2"
	"github.com/hr3lxphr6j/ctfile/ctfile"
	"github.com/hr3lxphr6j/ctfile/utils"
)

var (
	shareIDs      []string
	pubCookie     string
	aria2Endpoint string
	aria2Token    string
	aria2Output   string
	concurrent    int
)

func init() {
	flag.StringVar(&pubCookie, "cookie", "", "pub cookie of ctfile")
	flag.StringVar(&aria2Endpoint, "aria2-endpoint", "http://127.0.0.1:6800/jsonrpc", "endpoint of aria2 rpc")
	flag.StringVar(&aria2Token, "aria2-token", "", "token of aria2 rpc")
	flag.StringVar(&aria2Output, "aria2-output", "", "output path")
	flag.IntVar(&concurrent, "concurrent", 5, "concurrent of download")
	flag.Parse()
	shareIDs = flag.Args()
}

func main() {
	if len(shareIDs) == 0 {
		log.Fatal("no input")
	}

	ct := ctfile.NewClient()
	if pubCookie != "" {
		if err := ct.SetCookies(pubCookie); err != nil {
			log.Fatalf("failed to set cookie, error %v", err)
		}
	}

	aria2C := aria2.New(aria2Endpoint, aria2Token)

	var (
		sem   = make(chan struct{}, concurrent)
		gidCh = make(chan string, concurrent)
		done  = make(chan struct{})
	)
	// fill
	for {
		select {
		case sem <- struct{}{}:
			continue
		default:
		}
		break
	}

	go func() {
		for {
			select {
			case <-done:
				return
			case gid := <-gidCh:
				go func() {
					t := time.NewTicker(time.Second)
					defer t.Stop()
					for {
						select {
						case <-done:
							return
						case <-t.C:
							status, err := aria2C.TellStatus(gid)
							if err != nil {
								// maybe task was be del
								sem <- struct{}{}
								return
							}
							// TODO: parse other status, add retry when error happen.
							if status.Status != aria2.StatusComplete {
								continue
							}
							sem <- struct{}{}
							return
						}
					}
				}()
			}
		}
	}()

	for _, id := range shareIDs {
		err := ct.Walk(id, "", func(curPath string, share *ctfile.Share, file *ctfile.File) bool {
			<-sem
			log.Printf("Name: %s\t Size: %s\n", file.Name, file.Size)
			urls, err := ct.GetDownloadUrl(file)
			if err != nil || len(urls) == 0 {
				log.Fatalf("failed to get download url, filename: %s, id: %s, err: %v", file.Name, file.ID, err)
			}
			gid, err := aria2C.AddUri(
				utils.Map2slice(urls),
				aria2.Directory(aria2Output),
				aria2.Output(path.Join(curPath, file.Name)),
			)
			if err != nil {
				log.Fatalf("failed to add uri to aria2, err: %v", err)
			}
			gidCh <- gid
			return true
		})
		if err != nil {
			log.Fatalf("%v", err)
		}
	}
	close(done)
}
