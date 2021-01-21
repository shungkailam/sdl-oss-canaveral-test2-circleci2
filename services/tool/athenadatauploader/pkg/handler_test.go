package pkg_test

import (
	"cloudservices/common/base"
	"cloudservices/tool/athenadatauploader/pkg"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAthenaDataUploader(t *testing.T) {
	ctx := context.Background()
	tableNames := []string{}
	pkg.Cfg.TableNames.Set("service_class_model")
	pkg.Cfg.TableNames.Set("log_model")
	for _, tableName := range pkg.Cfg.TableNames.Values() {
		tableNames = append(tableNames, tableName)
	}
	pkg.Cfg.S3Prefix = base.StringPtr("mytest")
	pkg.Cfg.AthenaTableSuffix = base.StringPtr("mytest")
	now := time.Now().UTC()
	uploader := pkg.NewAthenaDataUploader(pkg.Cfg)
	err := uploader.Start(ctx, now, tableNames...)
	require.NoError(t, err)
}
