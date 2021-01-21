package main

import (
	"cloudservices/tool/athenadatauploader/pkg"
	"context"
	"time"
)

func init() {
	pkg.Cfg.LoadFlag()
}

func main() {
	ctx := context.Background()
	tableNames := []string{}
	for _, tableName := range pkg.Cfg.TableNames.Values() {
		tableNames = append(tableNames, tableName)
	}
	now := time.Now().UTC()
	uploader := pkg.NewAthenaDataUploader(pkg.Cfg)
	err := uploader.Start(ctx, now, tableNames...)
	if err != nil {
		panic(err)
	}
}
