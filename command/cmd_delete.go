package command

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/urfave/cli/v2"

	"github.com/peak/s5cmd/log"
	"github.com/peak/s5cmd/objurl"
	"github.com/peak/s5cmd/parallel"
	"github.com/peak/s5cmd/storage"
)

var DeleteCommand = &cli.Command{
	Name:     "rm",
	HelpName: "delete",
	Usage:    "TODO",
	Before: func(c *cli.Context) error {
		if !c.Args().Present() {
			return fmt.Errorf("expected at least 1 object to remove")
		}
		return nil
	},
	OnUsageError: func(c *cli.Context, err error, isSubcommand bool) error {
		if err != nil {
			printError(givenCommand(c), "delete", err)
		}
		return err
	},
	Action: func(c *cli.Context) error {
		doDelete := func() error {
			return Delete(
				c.Context,
				givenCommand(c),
				c.Args().Slice()...,
			)
		}
		parallel.Run(doDelete)
		return nil
	},
}

func Delete(ctx context.Context, fullCommand string, args ...string) error {
	sources := make([]*objurl.ObjectURL, len(args))
	for i, arg := range args {
		url, _ := objurl.New(arg)
		sources[i] = url
	}

	client, err := storage.NewClient(sources[0])
	if err != nil {
		return err
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
					printError(fullCommand, "delete", err)
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
			printError(fullCommand, "delete", err)
			continue
		}

		msg := log.InfoMessage{
			Operation: "delete",
			Source:    obj.URL,
		}
		log.Info(msg)
	}

	return merror
}