package main

import (
	"cloudservices/tool/supportlogcleaner/pkg"
	"context"
	"time"
)

func init() {
	pkg.Cfg.LoadFlag()
}
func main() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	updatedBeforeDays := *pkg.Cfg.UpdatedBeforeDays
	updatedBefore := time.Now().Add(-time.Hour * time.Duration(24*updatedBeforeDays))
	pkg.DeleteLogEntries(ctx, updatedBefore, pkg.DeleteRecords)
}
