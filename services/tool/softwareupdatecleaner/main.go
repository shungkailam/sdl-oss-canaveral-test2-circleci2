package main

import (
	"cloudservices/tool/softwareupdatecleaner/pkg"
	"context"
)

func init() {
	pkg.Cfg.LoadFlag()
}

func main() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	pkg.DeleteExpiredBatches(ctx)
}
