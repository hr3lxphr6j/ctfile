package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"path"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v3"
	"go.uber.org/ratelimit"

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

type (
	aria2ClientKey  struct{}
	ctfileClientKey struct{}
	rateLimitKey    struct{}
)

type task struct {
	Done chan struct{}
	once sync.Once

	File    *ctfile.File
	CurPath string

	Gid string
	Err error

	hooks []func(task *task)
}

func (t *task) SetDone(err error) {
	t.once.Do(func() {
		t.Err = err
		close(t.Done)
		if len(t.hooks) > 0 {
			for _, hook := range t.hooks {
				hook(t)
			}
		}
	})
}

func newTask(file *ctfile.File, curPath string, hooks ...func(task *task)) *task {
	return &task{
		Done:    make(chan struct{}),
		File:    file,
		CurPath: curPath,
		hooks:   hooks,
	}
}

func waitTask(ctx context.Context, task *task) {
	client := ctx.Value(aria2ClientKey{}).(*aria2.Client)
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			status, err := client.TellStatus(task.Gid)
			if err != nil {
				// task was removed.
				task.SetDone(nil)
				return
			}
			switch status.Status {
			case aria2.StatusComplete:
				task.SetDone(nil)
				return
			case aria2.StatusError:
				task.SetDone(errors.New(status.ErrorMessage))
				return
			default:
				continue
			}
		}
	}
}

func consumer(ctx context.Context, pendingCh <-chan *task) {
	aria2Client := ctx.Value(aria2ClientKey{}).(*aria2.Client)
	ctfileClient := ctx.Value(ctfileClientKey{}).(*ctfile.Client)
	rl := ctx.Value(rateLimitKey{}).(ratelimit.Limiter)
	for {
		select {
		case <-ctx.Done():
			return
		case task := <-pendingCh:
			log.Printf("File: %s, Size: %s", task.File.Name, task.File.Size)
			rl.Take()
			var urls map[string]string
			err := backoff.Retry(func() error {
				_urls_, _err_ := ctfileClient.GetDownloadUrl(task.File)
				if _err_ != nil {
					return _err_
				}
				if len(_urls_) == 0 {
					return errors.New("url is empty")
				}
				urls = _urls_
				return nil
			}, backoff.NewExponentialBackoffBuilder().MaxRetries(3).Build())

			if err != nil {
				log.Printf("failed to get download url after max retry, filename: %s, err: %s", task.File.Name, err)
				task.SetDone(err)
				continue
			}

			gid, err := aria2Client.AddUri(
				utils.Map2slice(urls),
				aria2.Output(path.Join(task.CurPath, task.File.Name)),
				aria2.Directory(aria2Output),
			)
			if err != nil {
				log.Fatalf("failed to call aria2.AddUri, err: %s", err)
			}
			task.Gid = gid
			waitTask(ctx, task)
		}
	}
}

func main() {
	if len(shareIDs) == 0 {
		log.Fatal("no input")
	}
	if concurrent <= 0 {
		log.Fatal("concurrent must be greater than 0")
	}

	ctfileClient := ctfile.NewClient()
	if pubCookie != "" {
		if err := ctfileClient.SetCookies(pubCookie); err != nil {
			log.Fatalf("failed to set cookie, error %v", err)
		}
	}

	var (
		pendingCh   = make(chan *task, concurrent)
		ctx, cancel = context.WithCancel(context.TODO())
		wg          = sync.WaitGroup{}
		reWalk      = make(chan struct{}, 1)
	)
	ctx = context.WithValue(ctx, ctfileClientKey{}, ctfileClient)
	ctx = context.WithValue(ctx, rateLimitKey{}, ratelimit.New(30))
	ctx = context.WithValue(ctx, aria2ClientKey{}, aria2.New(aria2Endpoint, aria2Token))

	// process pending chan, add task to aria2.
	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func() {
			consumer(ctx, pendingCh)
			wg.Done()
		}()
	}

LoopShare:
	for _, id := range shareIDs {
		finishedFile := make(map[string]struct{}, 64)
		finishedFileLock := new(sync.RWMutex)
		b := backoff.NewExponentialBackoffBuilder().MaxRetries(10).Build()
	LoopWalk:
		for {
			err := ctfileClient.Walk(id, "", func(curPath string, share *ctfile.Share, file *ctfile.File) bool {
				finishedFileLock.RLock()
				if _, ok := finishedFile[path.Join(curPath, file.Name)]; ok {
					finishedFileLock.RUnlock()
					return true
				}
				finishedFileLock.RUnlock()

				task := newTask(file, curPath, func(task *task) {
					if task.Err == nil {
						finishedFileLock.Lock()
						finishedFile[path.Join(curPath, file.Name)] = struct{}{}
						finishedFileLock.Unlock()
						return
					}
					select {
					case reWalk <- struct{}{}:
					default:
					}
				})
				select {
				case pendingCh <- task:
					return true
				case <-reWalk:
					reWalk = make(chan struct{}, 1)
					return false
				}
			})

			switch err {
			case nil:
				// TODO: wait all task finish.
				continue LoopShare
			case ctfile.ErrWalkAbort:
				log.Print("some error happen, trigger reWalk...")
				continue LoopWalk
			default:
				d := b.NextBackOff()
				if d == backoff.Stop {
					log.Fatalf("failed to parse share after max retry, err: %v", err)
				}
				log.Printf("failed to parse share, id: %s, err: %v, will retry after %s", id, err, d)
				time.Sleep(d)
				continue LoopWalk
			}
		}
	}

	cancel()
	wg.Wait()
}
