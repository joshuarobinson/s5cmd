package core

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/peak/s5cmd/log"
	"github.com/peak/s5cmd/objurl"
	"github.com/peak/s5cmd/storage"
)

func BatchDelete(ctx context.Context, job *Job) *JobResponse {
	sources := job.args

	client, err := storage.NewClient(sources[0])
	if err != nil {
		return jobResponse(err)
	}

	// do object->objurl transformation
	urlch := make(chan *objurl.ObjectURL)

	go func() {
		defer close(urlch)

		// there are multiple source files which are received from batch-rm
		// command.
		if len(sources) > 1 {
			for _, url := range sources {
				select {
				case <-ctx.Done():
					return
				case urlch <- url:
				}
			}
		} else {
			// src is a glob
			src := sources[0]
			for object := range client.List(ctx, src, true, storage.ListAllItems) {
				if object.Type.IsDir() || isCancelationError(object.Err) {
					continue
				}

				if err := object.Err; err != nil {
					printError(job, err)
					continue
				}
				urlch <- object.URL
			}
		}
	}()

	resultch := client.MultiDelete(ctx, urlch)

	// closed errch indicates that MultiDelete operation is finished.
	var merror error
	for obj := range resultch {
		if err := obj.Err; err != nil {
			if isCancelationError(obj.Err) {
				continue
			}

			merror = multierror.Append(merror, obj.Err)
			printError(job, err)
			continue
		}

		log.Logger.Info(InfoMessage{
			Operation: job.operation.String(),
			Source:    obj.URL,
		})
	}

	return jobResponse(merror)
}

func Delete(ctx context.Context, job *Job) *JobResponse {
	src := job.args[0]

	client, err := storage.NewClient(src)
	if err != nil {
		return jobResponse(err)
	}

	err = client.Delete(ctx, src)
	if err != nil {
		return jobResponse(err)
	}

	log.Logger.Info(InfoMessage{
		Operation: job.operation.String(),
		Source:    src,
	})

	return jobResponse(nil)
}
