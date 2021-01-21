package cmd_test

import (
	"cloudservices/tenantpool/cli/cmd"
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestGenerateDeleteEntityQueries(t *testing.T) {
	ctx := context.Background()
	scavenger := cmd.NewScavenger()
	defer scavenger.Close()
	dependencies, err := scavenger.GetTableDependencies(ctx)
	require.NoError(t, err)
	t.Logf("Dependencies: %+v", dependencies)
	count := 0
	edgeCertModelDeleteQuery := `DELETE FROM edge_cert_model e USING tenant_model t WHERE e.tenant_id = t.id AND t.name='Trial Tenant' AND t.external_id is null AND (SELECT count(*) FROM user_model where tenant_id = t.id AND email not like '%.ntnx-del') = 0 AND t.updated_at < :update_before`
	edgeModelDeleteQuery := `DELETE FROM edge_model e USING tenant_model t WHERE e.tenant_id = t.id AND t.name='Trial Tenant' AND t.external_id is null AND (SELECT count(*) FROM user_model where tenant_id = t.id AND email not like '%.ntnx-del') = 0 AND t.updated_at < :update_before`
	tenantModelDeleteQuery := `DELETE FROM tenant_model e USING tenant_model t WHERE e.id = t.id AND t.name='Trial Tenant' AND t.external_id is null AND (SELECT count(*) FROM user_model where tenant_id = t.id AND email not like '%.ntnx-del') = 0 AND t.updated_at < :update_before`
	err = scavenger.GenerateDeleteEntityQueries(ctx, dependencies, func(table, deleteEntityQuery string) error {
		if deleteEntityQuery == edgeCertModelDeleteQuery {
			if count != 0 {
				t.Fatalf("Expected %s to be deleted first", edgeCertModelDeleteQuery)
			}
			count++
		}
		if deleteEntityQuery == edgeModelDeleteQuery {
			if count != 1 {
				t.Fatalf("Expected %s to be deleted second", edgeModelDeleteQuery)
			}
			count++
		}
		if deleteEntityQuery == tenantModelDeleteQuery {
			if count != 2 {
				t.Fatalf("Expected %s to be deleted third", edgeModelDeleteQuery)
			}
		}
		return nil
	})
	require.NoError(t, err)
}

func TestScavengerRun(t *testing.T) {
	ctx := context.Background()
	scavenger := cmd.NewScavenger()
	defer scavenger.Close()
	count, err := scavenger.Run(ctx, true, time.Hour*24*20, "")
	require.NoError(t, err)
	t.Logf("Affected tenants count: %d", count)
}
