package crawler

import (
	"digger/common"
	"digger/models"
	"errors"
	"github.com/go-resty/resty/v2"
	"github.com/hetianyi/gox/logger"
	"github.com/hetianyi/gox/timer"
	"net/http"
	"sync"
	"time"
)

type restyClient struct {
	client  *resty.Client
	lastUse time.Time
}

var (
	restyClientCache = make(map[int]*restyClient)
	restyClientLock  = new(sync.Mutex)
)

func init() {
	expireClientDetect()
}

func expireClientDetect() {
	timer.Start(0, time.Second*10, 0, func(t *timer.Timer) {
		restyClientLock.Lock()
		defer restyClientLock.Unlock()

		for taskId, c := range restyClientCache {
			if c.lastUse.Before(time.Now()) {
				delete(restyClientCache, taskId)
				logger.Debug("resty client expired: ", taskId)
				break
			}
		}
	})
}

func getClient(queue *models.Queue, project *models.Project) *resty.Client {
	restyClientLock.Lock()
	defer restyClientLock.Unlock()

	client := restyClientCache[queue.TaskId]
	if client == nil {
		restyClientCache[queue.TaskId] = &restyClient{
			client: resty.New().
				SetTimeout(time.Second * time.Duration(project.GetIntSetting(common.SETTINGS_REQUEST_TIMEOUT, 60))).
				SetRedirectPolicy(resty.RedirectPolicyFunc(func(req *http.Request, via []*http.Request) error {
					// return nil for continue redirect otherwise return error to stop/prevent redirect
					f := project.GetBoolSetting(common.SETTINGS_FOLLOW_REDIRECT)
					if f {
						return nil
					}
					return errors.New("follow redirect is disabled")
				})).
				SetRetryCount(project.GetIntSetting(common.SETTINGS_RETRY_COUNT, 0)).
				SetRetryWaitTime(time.Second * time.Duration(project.GetIntSetting(common.SETTINGS_RETRY_WAIT, 3))),
			lastUse: time.Now(),
		}
	}
	return restyClientCache[queue.TaskId].client
}
